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
