# Hippo - Azure DevOps TUI

A terminal user interface (TUI) for Azure DevOps task management, built with Go and Bubbletea.

## TODOs

- auto release on main branch push (use GoReleaser)
- support curl install for first installation (support MacOS, Windows, Linux)
- In-app update check 
- Use go-selfupdate if update available (and offer "skip version")

- changelog generation
- changelog support in-app
- justfile

## Features

- 📋 View all your Azure DevOps work items in a clean terminal interface
- 🌲 Hierarchical tree view showing parent-child task relationships
- 🏃 Sprint-based navigation (Previous, Current, Next sprint tabs)
- 🔍 Real-time search by title or work item ID
- 📊 Detailed work item cards with all information including:
  - Parent task information
  - State, priority, tags, assigned user
  - Relative timestamps (e.g., "2 days ago", "3 weeks ago")
  - Full description and comments
- 🎨 Color-coded states (Proposed, InProgress, Completed, Removed)
- 🌐 Open work items directly in browser
- ✏️ Change work item state from the TUI
- ⚡ Lazy loading with "Load More" functionality
- 🔐 Secure authentication via Azure CLI (no PAT needed)
- 🚫 Automatically filters out closed and removed items

## Prerequisites

- Go 1.21 or higher
- Azure CLI
- Azure DevOps Board

## Setup

1. **Install Azure CLI** (if not already installed):
   - macOS: `brew install azure-cli`
   - Windows: Download from [Microsoft Docs](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
   - Linux: Follow instructions at [Microsoft Docs](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)

2. **Login to Azure**:
```bash
az login
```

3. **Install Go dependencies**:
```bash
go mod download
```

4. **Build and run Hippo**:
```bash
cd app
go build -o hippo
./hippo
```

On first run, the setup wizard will start automatically and prompt you for:
- Azure DevOps organization URL (e.g., `https://dev.azure.com/your-org`)
- Project name
- Team name (optional)

Your configuration is saved to `~/.config/hippo/config.yaml`.

To reconfigure later, run: `./hippo --init`

## Building

```bash
cd app
go build -o hippo
```

## Running

```bash
cd app
./hippo
```

Or run directly:
```bash
cd app
go run .
```

## Testing

Hippo includes comprehensive unit tests and benchmarks for core functionality:

```bash
cd app

# Run all tests
go test -v

# Run specific test
go test -run TestTreeCache -v

# Run benchmarks
go test -bench=. -benchmem

# Compare cache performance
go test -bench=BenchmarkTreeCacheVsNoCacheScrolling -benchmem

# Check test coverage
go test -cover

# Generate coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

The test suite includes:
- Unit tests for tree building and flattening
- Cache hit/miss verification tests
- Cache invalidation tests across operations
- Integration tests for multi-list caching
- Performance benchmarks comparing cached vs uncached tree operations

## Keyboard Shortcuts

### List View
- `↑/↓` or `j/k` - Navigate up/down
- `→/l` or `enter` - Open work item details
- `tab` - Cycle between sprint tabs (Previous, Current, Next)
- `o` - Open selected work item in browser
- `/` - Search work items by title or ID
- `f` - Filter work items (coming soon)
- `r` - Refresh all data
- `q` or `ctrl+c` - Quit

### Detail View
- `←/h` or `esc` - Return to list view
- `o` - Open work item in browser
- `s` - Change work item state
- `q` or `ctrl+c` - Quit

### Search View
- Type to search by title or ID
- `↑/↓` or `ctrl+j/k` - Navigate search results
- `ctrl+d/u` - Jump half page up/down
- `enter` - Open selected work item details
- `esc` - Cancel search and return to list

### State Picker View
- `↑/↓` or `j/k` - Navigate available states
- `enter` - Select new state
- `esc` - Cancel state change

## Configuration

### Configuration File

Hippo stores configuration in a standard location:
- **macOS/Linux:** `~/.config/hippo/config.yaml`
- **Windows:** `%APPDATA%\hippo\config.yaml`

Example configuration:
```yaml
config_version: 1
organization_url: "https://dev.azure.com/your-org"
project: "your-project"
team: "your-team"  # optional
```

See `app/config.example.yaml` for a complete example.

### Configuration Sources & Precedence

Hippo supports multiple configuration sources with the following precedence (highest to lowest):

1. **Command-line flags** - Temporary overrides for single runs
2. **Environment variables** - For CI/CD, development, containers
3. **Config file** - Persistent user configuration

This means you can have a config file for daily use, but override values with environment variables in CI/CD or flags for quick tests.

### First-Time Setup

Simply run Hippo, and the setup wizard will start automatically:
```bash
./hippo
```

The wizard will prompt you for your Azure DevOps organization URL, project name, and team name (optional).

### Reconfiguring

To update your configuration, force the wizard to run:
```bash
./hippo --init
```

Or manually edit the config file:
```bash
# macOS/Linux
vim ~/.config/hippo/config.yaml

# Windows
notepad %APPDATA%\hippo\config.yaml
```

### Command-Line Flags

Override configuration for a single run:
```bash
# Override organization and project
hippo --org https://dev.azure.com/other-org --project OtherProject

# Use a different config file
hippo --config /path/to/custom-config.yaml
```

Available flags:
- `--org` - Override organization URL
- `--project` - Override project name
- `--team` - Override team name
- `--config` - Use custom config file path
- `--init` - Run setup wizard
- `--version` - Show version
- `--help` - Show help

### Environment Variables

Environment variables are fully supported and useful for:
- **CI/CD pipelines** (GitHub Actions, Azure Pipelines, etc.)
- **Docker containers**
- **Local development** with `.env` files
- **Testing** different configurations

Supported variables:
```bash
export HIPPO_ADO_ORG_URL="https://dev.azure.com/your-org"
export HIPPO_ADO_PROJECT="your-project"
export HIPPO_ADO_TEAM="your-team"
```

Example: Override project in CI/CD:
```bash
# Config file has your default project
# Override just the project for this run
export HIPPO_ADO_PROJECT="CI-Test-Project"
./hippo
```

Example: Use `.env` file for development:
```bash
# Create .env file
echo "HIPPO_ADO_PROJECT=DevProject" > .env

# godotenv automatically loads .env
./hippo
```

### Configuration Examples

**Example 1: Single project user**
```bash
# First run - wizard starts automatically
./hippo

# Daily use
./hippo
```

**Example 2: Multiple projects**
```bash
# First run sets up main project (wizard runs automatically)
./hippo

# Switch projects temporarily
./hippo --project "OtherProject"

# Or use environment variables
export HIPPO_ADO_PROJECT="OtherProject"
./hippo

# Reconfigure to different project permanently
./hippo --init
```

**Example 3: CI/CD pipeline**
```yaml
# .github/workflows/check-work-items.yml
steps:
  - name: Run Hippo
    env:
      HIPPO_ADO_ORG_URL: https://dev.azure.com/my-org
      HIPPO_ADO_PROJECT: CI-Project
    run: ./hippo
```

**Example 4: Docker container**
```bash
# Option 1: Mount config file from host
docker run -v ~/.config/hippo:/root/.config/hippo hippo

# Option 2: Use environment variables (must provide ALL required fields)
docker run \
  -e HIPPO_ADO_ORG_URL="https://dev.azure.com/my-org" \
  -e HIPPO_ADO_PROJECT="MyProject" \
  -e HIPPO_ADO_TEAM="MyTeam" \
  hippo
```

### Migrating from .env Files

If you previously used `.env` files for configuration:

**Option 1: Switch to config file (recommended for regular use)**

1. Run the setup wizard:
   ```bash
   hippo --init
   ```

2. Enter the same values you had in your `.env` file:
   - `HIPPO_ADO_ORG_URL` → `organization_url`
   - `HIPPO_ADO_PROJECT` → `project`
   - `HIPPO_ADO_TEAM` → `team`

3. (Optional) Remove old `.env` file:
   ```bash
   rm .env
   ```

**Option 2: Keep using environment variables**

Environment variables are fully supported! You can continue using `.env` files, environment variables, or a mix of both. This is particularly useful for:
- Development environments
- CI/CD pipelines
- Docker deployments

No migration needed - your existing setup will continue to work.

## Project Structure

```
.
├── app/
│   ├── main.go       # Main application and Bubbletea UI logic
│   ├── main_test.go  # Unit tests and benchmarks for tree operations
│   ├── client.go     # Azure DevOps API client
│   └── go.mod        # Go module definition
├── README.md         # This file
└── AGENTS.md         # Architecture documentation for AI agents
```

## Architecture

### UI Layout (Fixed Three-Part Structure)

All views in Hippo follow a consistent three-part layout:

```
┌─────────────────────────────────────┐
│  Title Bar (always present)         │ ← renderTitleBar()
│  Shows view title + version         │
├─────────────────────────────────────┤
│                                     │
│  Dynamic Content Area               │ ← Changes per view
│  (list, detail, search, etc.)       │
│                                     │
├─────────────────────────────────────┤
│  Action Log (last action timestamp) │
│  ─────────────────────────────────  │ ← renderFooter()
│  Keybindings (context-specific)     │
└─────────────────────────────────────┘
```

### View Types

1. **List View** - Shows work items in a tree structure with sprint tabs
2. **Detail View** - Displays a card with complete work item information including parent
3. **Search View** - Filtered view of work items with search input
4. **State Picker View** - Select new state for a work item
5. **Filter View** - Custom query input (coming soon)

### Key Components

- **`renderTitleBar(title)`** - Renders consistent header across all views
- **`renderFooter(keybindings)`** - Renders consistent footer with log and help
- **`buildDetailContent()`** - Creates the work item detail card
- **`getRelativeTime(date)`** - Formats dates with human-readable relative time

See [AGENTS.md](./AGENTS.md) for detailed architecture documentation.

## Detailed Architecture

### Data Models

**WorkItem** (`app/main.go:80-95`)
```go
type WorkItem struct {
    ID, Title, State, AssignedTo, WorkItemType, Description, Tags string
    Priority int
    CreatedDate, ChangedDate, IterationPath string
    ParentID *int
    Children []*WorkItem
    Comments string
}
```

**TreeItem** - Flattened tree with depth info for rendering
**Sprint** - Sprint metadata (Name, Path, StartDate, EndDate)

### Key Functions

**Layout System:**
- `renderTitleBar(title)` - Consistent purple header with version
- `renderFooter(keybindings)` - Footer with action log and keybindings

**Tree Building:**
- `buildTreeStructure()` - Organizes items into parent-child hierarchy
- `flattenTree()` - Converts tree to flat list with depth info
- `getTreePrefix()` - Returns tree drawing characters (│, ├, ╰)

**Detail View:**
- `buildDetailContent()` - Creates work item card
- `getParentTask()` - Finds parent work item
- `getRelativeTime()` - Formats dates (< day ago, 2 weeks ago, etc.)

**Data Filtering:**
- `getVisibleTasks()` - Applies search and sprint filters
- `getVisibleTreeItems()` - Returns filtered tree structure

### File Structure

**app/main.go** (~1700 lines)
- Constants & Types (1-136)
- Initialization (137-275)
- Update Logic (277-756)
- Tree Building (758-816)
- Data Helpers (818-922)
- Detail View Logic (924-1066)
- Rendering Framework (1112-1141)
- View Renderers (1143-1700)

**app/client.go** (~550 lines)
- Azure DevOps API client
- Authentication via Azure CLI
- Work item CRUD operations
- Sprint/iteration queries

### Adding a New View

1. Create render function:
```go
func (m model) renderNewView() string {
    var content strings.Builder
    content.WriteString(m.renderTitleBar("View Title"))
    content.WriteString("...content...")
    content.WriteString(m.renderFooter("keybindings"))
    return content.String()
}
```

2. Add to `viewState` constants and `View()` switch
3. Add keyboard handlers in `Update()`

## Troubleshooting

### First Run Setup

If you need to reconfigure or update your settings, run:
```bash
hippo --init
```

The setup wizard will guide you through the configuration process.

### Configuration file location

Your config file is located at:
- **macOS/Linux:** `~/.config/hippo/config.yaml`
- **Windows:** `%APPDATA%\hippo\config.yaml`

To use a custom location:
```bash
hippo --config /path/to/custom-config.yaml
```

You can manually edit this file if needed. Required fields are:
- `config_version: 1`
- `organization_url: "https://dev.azure.com/your-org"`
- `project: "your-project"`

### "failed to get Azure CLI token"
- Make sure you're logged in: `az login`
- Check your Azure CLI installation: `az --version`
- Verify you have access to Azure DevOps

### "failed to query work items"
- Verify your organization URL is correct
- Ensure your project name matches exactly (case-sensitive)
- Make sure your Azure account has access to the Azure DevOps project

### Testing configuration

Test your configuration with flag overrides:
```bash
# Test with different project
hippo --project "TestProject"

# Test with different organization
hippo --org "https://dev.azure.com/test-org" --project "TestProject"
```

### Empty list
If you see "No tasks found", the query might not match any work items. Check that:
- Your project has work items
- The work items are not in "Closed" or "Removed" state

## Future Enhancements

- ✅ ~~Filter by assigned user~~ (Implemented - filters @Me by default)
- ✅ ~~Search functionality~~ (Implemented)
- ✅ ~~Update work item state~~ (Implemented)
- ✅ ~~Sprint/iteration view~~ (Implemented)
- 🚧 Advanced custom WIQL queries
- 🚧 Add comments to work items
- 🚧 Create new work items
- 🚧 Bulk operations
- 🚧 Work item linking
- 🚧 Attachments view
- 🚧 Time tracking integration
- 🚧 Keyboard shortcuts customization
