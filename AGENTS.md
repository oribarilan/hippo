# Hippo - Coding Guide for AI Agents

Quick reference for agents working on the Hippo codebase.

## Build/Run/Test

All commands should be run from the `app/` directory:

```bash
cd app
```

- **Build:** `go build -o hippo` or `go build`
- **Run:** `go run .` or `./hippo`
- **Format:** `go fmt ./...` (required before commits)
- **Lint:** `go vet ./...`
- **Tests:** `go test -v` (unit tests for tree building/caching)
- **Benchmarks:** `go test -bench=. -benchmem` (performance tests)

## Code Style

**Imports:** stdlib first, then external packages (grouped, alphabetical)
**Types:** Explicit types, avoid `interface{}` (use `any` if needed)
**Naming:** camelCase (private), PascalCase (exported)
**Error Handling:** Always check errors, wrap with `fmt.Errorf("context: %w", err)`
**Comments:** Exported functions need doc comments starting with function name
**Constants:** Group in `const ()` blocks
**Receivers:** Use consistent names (e.g., `m` for `model`)

## File Organization Principle

**Keep files simple with clear single responsibility.**

When working on this codebase:
- Each file should have ONE clear purpose
- Split large files into smaller, focused modules
- Extract related functionality into separate files
- Name files clearly to reflect their single responsibility
- Avoid creating monolithic files that do multiple things

Examples:
- `client_*.go` - Each client file handles a specific API domain
- `model.go` - Data structures only
- `views.go` - View rendering logic
- `handlers.go` - Request handlers
- `styles.go` - Styling/colors

## Architecture Basics

**Configuration:**

Hippo uses YAML configuration files stored in:
- macOS/Linux: `~/.config/hippo/config.yaml`
- Windows: `%APPDATA%\hippo\config.yaml`

Configuration loading and merging precedence (highest to lowest):
1. Command-line flags (temporary overrides)
2. Environment variables (CI/CD, development)
3. Config file (persistent configuration)

Setup wizard (`--init`):
- Integrated as a TUI view (configWizardView)
- Starts automatically if no config exists
- Exits to loading view after saving config

Key configuration files:
- `config.go` - Configuration loading, merging, and validation
- `view_config_wizard.go` - Integrated config wizard view
- `flags.go` - CLI flag parsing
- `client_auth.go` - Azure CLI token authentication

**Layout Pattern:** All views use 3-part structure:
```go
func (m model) renderSomeView() string {
    var content strings.Builder
    content.WriteString(m.renderTitleBar("Title"))  // Header
    content.WriteString("...dynamic content...")    // Content
    content.WriteString(m.renderFooter("keys"))     // Footer
    return content.String()
}
```

**Files:**
- `main.go` - Bubbletea MVC (model, update, view)
- `client.go` - Azure DevOps API client
- `config.go` - Configuration management
- `view_config_wizard.go` - Integrated setup wizard view
- `flags.go` - CLI flags

**Color Palette:**
- Purple #62 (headers/selections), White #230 (text), Gray #241 (borders)
- Green #86 (InProgress), Pink #212 (icons), Blue #39 (headers)

## Key Locations

- View renderers: `main.go:1143-1700`
- Layout helpers: `renderTitleBar()` at `main.go:1074`, `renderFooter()` at `main.go:1095`
- Data models: `WorkItem`, `TreeItem`, `Sprint` at `main.go:39-102`
- Tree building: `buildTreeStructure()`, `flattenTree()` at `main.go:1604-1662`
- Detail card: `buildDetailContent()` at `main.go:973-1066`
- Tree caching: `WorkItemList.treeCache`, `invalidateTreeCache()` at `main.go:156-168, 1856-1859`
- Tests: `main_test.go` - unit, integration, and benchmark tests for tree operations

## Testing

**Test Coverage:**
- Unit tests for tree building and flattening
- Cache hit/miss verification tests
- Cache invalidation tests across operations
- Integration tests for multi-list caching
- Benchmarks comparing cached vs uncached performance

**Running Tests:**
```bash
cd app

# Run all tests with verbose output
go test -v

# Run specific test pattern
go test -run TestCache -v

# Run all benchmarks with memory stats
go test -bench=. -benchmem

# Compare cache vs no-cache performance
go test -bench=BenchmarkTreeCacheVsNoCacheScrolling -benchmem

# Show test coverage percentage
go test -cover

# Generate HTML coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

See [README.md](./README.md) for detailed architecture and user documentation.

## Releasing

Hippo uses GoReleaser for automated releases triggered by git tags.

**Release Process:**

1. Ensure all changes are committed and pushed to main
2. Create and push a version tag:
   ```bash
   git tag v0.3.0
   git push origin v0.3.0
   ```
3. GitHub Actions automatically:
   - Runs tests
   - Builds binaries for all platforms
   - Creates GitHub Release with artifacts
   - Generates changelog

**Version Format:** Semantic versioning (v0.3.0, v1.0.0, etc.)

**Version Embedding:**
- Version is injected at build time via `-ldflags`
- Variable: `main.Version` in `app/constants.go`
- Local builds show "dev"
- Released builds show actual version (e.g., "v0.3.0")

**Key Files:**
- `.goreleaser.yml` - Build configuration
- `.github/workflows/release.yml` - CI/CD workflow
- `install.sh` - Installation script
- `app/constants.go` - Version variable

For detailed release procedures and testing, see [CONTRIBUTE.md](./CONTRIBUTE.md).
