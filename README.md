# Hippo - Azure DevOps TUI

A simplified terminal app for Azure DevOps task management with the goal of quickly getting in and out of your work items.

## Features

- View all your Azure DevOps work items in a clean terminal interface
- Hierarchical tree view showing parent-child task relationships
- Sprint-based navigation (Previous, Current, Next sprint tabs)
- Real-time search by title or work item ID
- Detailed work item cards with all information including:
  - Parent task information
  - State, priority, tags, assigned user
  - Relative timestamps (e.g., "2 days ago", "3 weeks ago")
  - Full description and comments
- and more...

## Prerequisites

- Azure CLI installed and configured
- Azure DevOps account

## Installation

### Quick Install (Recommended)

Install Hippo with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/oribarilan/hippo/main/install.sh | bash
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
curl -sSL https://raw.githubusercontent.com/oribarilan/hippo/main/install.sh | INSTALL_DIR=/custom/path bash
```

**Note:** On macOS, you may need to run with `sudo` if `/usr/local/bin` requires elevated permissions, or specify a user directory:
```bash
curl -sSL https://raw.githubusercontent.com/oribarilan/hippo/main/install.sh | INSTALL_DIR=$HOME/.local/bin bash
```

## Getting Started

1. **Install Azure CLI** [Microsoft Docs](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
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

keybindings are visible in the help menu (`?`), with common actions also seen in the footer (bottom bar).

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

## Contributing

Interested in contributing to Hippo? Check out our [Contributing Guide](./CONTRIBUTE.md)
