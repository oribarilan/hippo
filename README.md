# Hippo - Azure DevOps TUI

A terminal user interface (TUI) for Azure DevOps task management, built with Go and Bubbletea.

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

- Azure CLI installed and configured
- Azure DevOps account

## Installation

### Quick Install (Recommended)

Install Hippo with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/orbarila/hippo/main/install.sh | bash
```

The installer will:
- Detect your platform (Linux, macOS, Windows)
- Download the latest release
- Install to the appropriate directory:
  - **Linux:** `~/.local/bin` (or `/usr/local/bin` if root)
  - **macOS:** `/usr/local/bin`
  - **Windows:** `~/bin`

**Custom installation directory:**
```bash
curl -sSL https://raw.githubusercontent.com/orbarila/hippo/main/install.sh | INSTALL_DIR=/custom/path bash
```

**Note:** On macOS, you may need to run with `sudo` if `/usr/local/bin` requires elevated permissions, or specify a user directory:
```bash
curl -sSL https://raw.githubusercontent.com/orbarila/hippo/main/install.sh | INSTALL_DIR=$HOME/.local/bin bash
```

**Important:** The quick install will be available once the first release (v0.3.0) is published. Until then, use [Building from Source](#building-from-source).

### Manual Installation

If you prefer to download and install manually:

1. **Download the latest release** for your platform from:
   [https://github.com/orbarila/hippo/releases/latest](https://github.com/orbarila/hippo/releases/latest)

2. **Extract the archive:**
   ```bash
   # Linux/macOS
   tar -xzf hippo_*_linux_amd64.tar.gz
   
   # Windows
   # Extract hippo_*_windows_amd64.zip using your preferred tool
   ```

3. **Move to a directory in your PATH:**
   ```bash
   # Linux/macOS (system-wide)
   sudo mv hippo /usr/local/bin/
   
   # Linux/macOS (user-only)
   mkdir -p ~/.local/bin
   mv hippo ~/.local/bin/
   
   # Windows
   # Move hippo.exe to a directory in your PATH
   ```

4. **Verify installation:**
   ```bash
   hippo --version
   ```

### Building from Source

**Prerequisites:** Go 1.21 or higher

1. **Clone and build**:
```bash
git clone https://github.com/orbarila/hippo.git
cd hippo/app
go build -o hippo
```

2. **Move to your PATH** (optional):
```bash
# macOS/Linux
sudo mv hippo /usr/local/bin/

# Or to user directory
mv hippo ~/.local/bin/
```

## Getting Started

1. **Install Azure CLI** (if not already installed):
   - macOS: `brew install azure-cli`
   - Windows: Download from [Microsoft Docs](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
   - Linux: Follow instructions at [Microsoft Docs](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)

2. **Login to Azure**:
```bash
az login
```

3. **Run Hippo**:
```bash
hippo
```

On first run, the setup wizard will start automatically and prompt you for:
- Azure DevOps organization URL (e.g., `https://dev.azure.com/your-org`)
- Project name
- Team name (optional)

Your configuration is saved to `~/.config/hippo/config.yaml`.

To reconfigure later, run: `hippo --init`

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
export AZURE_DEVOPS_ORG_URL="https://dev.azure.com/your-org"
export AZURE_DEVOPS_PROJECT="your-project"
export AZURE_DEVOPS_TEAM="your-team"  # optional
```

Example: Override project in CI/CD:
```bash
# Config file has your default project
# Override just the project for this run
export AZURE_DEVOPS_PROJECT="CI-Test-Project"
./hippo
```

Example: Use `.env` file for development:
```bash
# Create .env file
echo "AZURE_DEVOPS_PROJECT=DevProject" > .env

# godotenv automatically loads .env
./hippo
```

**For more configuration examples and advanced usage, see the full configuration section below.**

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
export AZURE_DEVOPS_PROJECT="OtherProject"
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
      AZURE_DEVOPS_ORG_URL: https://dev.azure.com/my-org
      AZURE_DEVOPS_PROJECT: CI-Project
    run: ./hippo
```

**Example 4: Docker container**
```bash
# Option 1: Mount config file from host
docker run -v ~/.config/hippo:/root/.config/hippo hippo

# Option 2: Use environment variables (must provide ALL required fields)
docker run \
  -e AZURE_DEVOPS_ORG_URL="https://dev.azure.com/my-org" \
  -e AZURE_DEVOPS_PROJECT="MyProject" \
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
   - `AZURE_DEVOPS_ORG_URL` → `organization_url`
   - `AZURE_DEVOPS_PROJECT` → `project`
   - `AZURE_DEVOPS_TEAM` → `team`

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

## Contributing

Interested in contributing to Hippo? Check out our [Contributing Guide](./CONTRIBUTE.md) for:
- Development setup instructions
- Code style guidelines
- Testing procedures
- Release process
- How to submit changes

## Project Structure

```
.
├── app/             # Main application code
├── README.md        # This file (user documentation)
├── CONTRIBUTE.md    # Contributing guide
├── AGENTS.md        # Architecture documentation
├── TESTING.md       # Testing documentation
├── WIZARD.md        # Setup wizard documentation
└── LICENSE.md       # MIT License
```

For detailed architecture and development information, see [CONTRIBUTE.md](./CONTRIBUTE.md) and [AGENTS.md](./AGENTS.md).

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
