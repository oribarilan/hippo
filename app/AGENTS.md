# Hippo - Coding Guide for AI Agents

Quick reference for agents working on the Hippo codebase.

## Build/Run/Test

- **Build:** `go build -o hippo` or `go build`
- **Run:** `go run .` or `./hippo`
- **Format:** `go fmt ./...` (required before commits)
- **Lint:** `go vet ./...`
- **Tests:** None currently - TUI requires manual testing

## Code Style

**Imports:** stdlib first, then external packages (grouped, alphabetical)
**Types:** Explicit types, avoid `interface{}` (use `any` if needed)
**Naming:** camelCase (private), PascalCase (exported)
**Error Handling:** Always check errors, wrap with `fmt.Errorf("context: %w", err)`
**Comments:** Exported functions need doc comments starting with function name
**Constants:** Group in `const ()` blocks
**Receivers:** Use consistent names (e.g., `m` for `model`)

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
- Tree building: `buildTreeStructure()`, `flattenTree()` at `main.go:758-816`
- Detail card: `buildDetailContent()` at `main.go:973-1066`

See [README.md](./README.md) for detailed architecture and user documentation.
