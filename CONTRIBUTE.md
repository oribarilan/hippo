# Contributing to Hippo

Thank you for your interest in contributing to Hippo! This guide will help you set up your development environment and understand the release process.

## Table of Contents

- [Development Setup](#development-setup)
- [Building from Source](#building-from-source)
- [Testing](#testing)
- [Code Style](#code-style)
- [Project Structure](#project-structure)
- [Architecture Overview](#architecture-overview)
- [Release Process](#release-process)
- [Submitting Changes](#submitting-changes)

## Development Setup

### Prerequisites

- **Go 1.21 or higher** - [Install Go](https://go.dev/doc/install)
- **Azure CLI** - For authentication
  - macOS: `brew install azure-cli`
  - Windows: Download from [Microsoft Docs](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
  - Linux: Follow instructions at [Microsoft Docs](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
- **Azure DevOps account** - With access to a project for testing
- **Git** - For version control

### Initial Setup

1. **Clone the repository**:
```bash
git clone https://github.com/oribarilan/hippo.git
cd hippo
```

2. **Install Go dependencies**:
```bash
cd app
go mod download
```

3. **Login to Azure**:
```bash
az login
```

4. **Run the setup wizard**:
```bash
cd app
go run . --init
```

The wizard will prompt you for:
- Azure DevOps organization URL (e.g., `https://dev.azure.com/your-org`)
- Project name
- Team name (optional)

Your configuration will be saved to `~/.config/hippo/config.yaml`.

## Building from Source

### Standard Build

Build without version information (shows "dev"):

```bash
cd app
go build -o hippo
./hippo --version  # Output: Hippo dev
```

### Build with Custom Version

Inject a custom version at build time:

```bash
cd app
go build -ldflags="-X main.Version=v0.3.0-custom" -o hippo
./hippo --version  # Output: Hippo v0.3.0-custom
```

### Run Without Building

For quick testing during development:

```bash
cd app
go run .
```

## Testing

Hippo includes comprehensive unit tests and benchmarks for core functionality.

### Running Tests

```bash
cd app

# Run all tests with verbose output
go test -v

# Run specific test pattern
go test -run TestTreeCache -v

# Run tests with race detection
go test -v -race ./...

# Check test coverage
go test -cover

# Generate HTML coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Running Benchmarks

```bash
cd app

# Run all benchmarks with memory stats
go test -bench=. -benchmem

# Compare cache vs no-cache performance
go test -bench=BenchmarkTreeCacheVsNoCacheScrolling -benchmem
```

### Test Coverage

The test suite includes:
- Unit tests for tree building and flattening
- Cache hit/miss verification tests
- Cache invalidation tests across operations
- Integration tests for multi-list caching
- Performance benchmarks comparing cached vs uncached tree operations

See `app/main_test.go` and `app/fixtures_test.go` for test implementations.

## Code Style

### General Guidelines

- **Imports:** Standard library first, then external packages (grouped, alphabetical)
- **Types:** Use explicit types, avoid `interface{}` (use `any` if needed)
- **Naming:** 
  - camelCase for private functions/variables
  - PascalCase for exported functions/variables
- **Error Handling:** Always check errors, wrap with `fmt.Errorf("context: %w", err)`
- **Comments:** Exported functions need doc comments starting with the function name
- **Constants:** Group related constants in `const ()` blocks
- **Receivers:** Use consistent receiver names (e.g., `m` for `model`)

### File Organization

**Keep files simple with clear single responsibility.**

- Each file should have ONE clear purpose
- Split large files into smaller, focused modules
- Extract related functionality into separate files
- Name files clearly to reflect their single responsibility

Examples:
- `client_*.go` - Each client file handles a specific API domain
- `model.go` - Data structures only
- `views.go` - View rendering logic
- `handlers.go` - Request handlers
- `styles.go` - Styling/colors

### Code Formatting

Before committing, always format and vet your code:

```bash
cd app

# Format all code
go fmt ./...

# Run static analysis
go vet ./...

# Run tests
go test -v ./...
```

These checks are also run automatically by GoReleaser during the release process.

## Project Structure

```
.
├── app/                          # Main application directory
│   ├── main.go                   # Bubbletea MVC (model, update, view)
│   ├── main_test.go              # Unit tests and benchmarks
│   ├── client.go                 # Azure DevOps API client base
│   ├── client_auth.go            # Azure CLI authentication
│   ├── client_backlog.go         # Backlog API operations
│   ├── client_sprints.go         # Sprint/iteration API operations
│   ├── client_workitems.go       # Work item API operations
│   ├── config.go                 # Configuration management
│   ├── config_wizard.go          # Interactive setup wizard
│   ├── view_config_wizard.go     # Config wizard TUI view
│   ├── view_*.go                 # Individual view renderers
│   ├── handlers_*.go             # Event handlers for different views
│   ├── model.go                  # Core data models
│   ├── types.go                  # Type definitions
│   ├── constants.go              # Application constants
│   ├── styles.go                 # Color palette and styling
│   ├── utils.go                  # Utility functions
│   ├── fixtures_test.go          # Test fixtures
│   ├── go.mod                    # Go module definition
│   └── go.sum                    # Go module checksums
├── .github/
│   └── workflows/
│       └── ci.yml                # CI workflow (tests, linting)
├── .goreleaser.yml               # GoReleaser configuration
├── README.md                     # User documentation
├── CONTRIBUTE.md                 # This file
├── AGENTS.md                     # Architecture guide for AI agents
├── TESTING.md                    # Detailed testing documentation
├── WIZARD.md                     # Setup wizard documentation
└── LICENSE.md                    # MIT License

```

## Architecture Overview

### UI Layout Pattern

All views follow a consistent three-part structure:

```go
func (m model) renderSomeView() string {
    var content strings.Builder
    content.WriteString(m.renderTitleBar("Title"))  // Header
    content.WriteString("...dynamic content...")    // Content
    content.WriteString(m.renderFooter("keys"))     // Footer
    return content.String()
}
```

### Key Components

- **Layout helpers:**
  - `renderTitleBar(title)` - Consistent purple header with version
  - `renderFooter(keybindings)` - Footer with action log and keybindings

- **Tree building:**
  - `buildTreeStructure()` - Organizes items into parent-child hierarchy
  - `flattenTree()` - Converts tree to flat list with depth info
  - `getTreePrefix()` - Returns tree drawing characters (│, ├, ╰)

- **Detail view:**
  - `buildDetailContent()` - Creates work item card
  - `getParentTask()` - Finds parent work item
  - `getRelativeTime()` - Formats dates (< day ago, 2 weeks ago, etc.)

- **Tree caching:**
  - `WorkItemList.treeCache` - Caches built tree structures
  - `invalidateTreeCache()` - Clears cache when data changes

### Color Palette

Defined in `styles.go`:
- Purple #62 (headers/selections)
- White #230 (text)
- Gray #241 (borders)
- Green #86 (InProgress state)
- Pink #212 (icons)
- Blue #39 (headers)

### Adding a New View

1. Create render function following the three-part layout pattern
2. Add to `viewState` constants
3. Add to `View()` switch statement
4. Add keyboard handlers in `Update()`

For detailed architecture documentation, see [AGENTS.md](./AGENTS.md).

## Release Process

Hippo uses GoReleaser for automated releases triggered by git tags.

### Prerequisites for Releases

- Push access to the repository
- All tests passing on main branch
- Changes committed and pushed to main

### Creating a Release

1. **Ensure all changes are committed and pushed to main**:
```bash
git status  # Should be clean
git push origin main
```

2. **Create and push a version tag**:
```bash
# Create a semantic version tag
git tag v0.3.0

# Push the tag to trigger release
git push origin v0.3.0
```

3. **GitHub Actions automatically**:
   - Runs all tests with race detection
   - Formats and vets code
   - Builds binaries for all platforms:
     - Linux (amd64, arm64)
     - macOS (amd64, arm64)
     - Windows (amd64)
   - Generates SHA256 checksums
   - Creates GitHub Release with artifacts
   - Auto-generates changelog from commit messages

4. **Monitor the release**:
   - Go to [GitHub Actions](https://github.com/oribarilan/hippo/actions)
   - Watch the "Release" workflow
   - Once complete, verify the release at [Releases](https://github.com/oribarilan/hippo/releases)

### Version Format

Use semantic versioning: `vMAJOR.MINOR.PATCH`

- **MAJOR:** Breaking changes
- **MINOR:** New features (backward compatible)
- **PATCH:** Bug fixes

Examples: `v0.3.0`, `v1.0.0`, `v1.2.3`

### Version Embedding

The version is automatically injected at build time:

- **Variable:** `main.Version` in `app/constants.go`
- **Local builds:** Show "dev"
- **Released builds:** Show actual version (e.g., "v0.3.0")
- **Build command:** GoReleaser uses `-ldflags="-X main.Version={{.Version}}"`

### Testing a Release Locally

Before creating an official release, you can test the release process:

```bash
# Install GoReleaser
brew install goreleaser

# Test release build (doesn't publish)
goreleaser release --snapshot --clean

# Check generated artifacts in dist/
ls -la dist/
```

### Release Checklist

Before creating a release:

- [ ] All tests pass: `go test -v ./...`
- [ ] Code is formatted: `go fmt ./...`
- [ ] Code is vetted: `go vet ./...`
- [ ] Version number follows semantic versioning
- [ ] CHANGELOG or commit messages are clear
- [ ] Documentation is up to date
- [ ] All changes committed and pushed to main

### Key Release Files

- `.goreleaser.yml` - Build configuration for releases
- `.github/workflows/release.yml` - CI/CD workflow (not yet created in Phase 2)
- `app/constants.go` - Version variable

## Submitting Changes

### Workflow

1. **Fork the repository** (or create a branch if you have access)

2. **Create a feature branch**:
```bash
git checkout -b feature/your-feature-name
```

3. **Make your changes** following the code style guidelines

4. **Test your changes**:
```bash
cd app
go fmt ./...
go vet ./...
go test -v ./...
```

5. **Commit your changes**:
```bash
git add .
git commit -m "feat: add your feature description"
```

Use conventional commit messages:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Test additions or changes
- `chore:` - Maintenance tasks

6. **Push to your fork/branch**:
```bash
git push origin feature/your-feature-name
```

7. **Create a Pull Request** on GitHub with:
   - Clear description of changes
   - Reference to related issues
   - Screenshots/demos if UI changes

### Pull Request Guidelines

- Keep PRs focused on a single feature or fix
- Update documentation if needed
- Add tests for new functionality
- Ensure all tests pass
- Follow the existing code style
- Be responsive to review feedback

## Getting Help

- **Issues:** [GitHub Issues](https://github.com/oribarilan/hippo/issues)
- **Discussions:** [GitHub Discussions](https://github.com/oribarilan/hippo/discussions)
- **Documentation:** See [AGENTS.md](./AGENTS.md) for detailed architecture

## License

By contributing to Hippo, you agree that your contributions will be licensed under the MIT License.
