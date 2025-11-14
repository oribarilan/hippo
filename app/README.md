# Hippo - Azure DevOps TUI

A terminal user interface (TUI) for Azure DevOps task management, built with Go and Bubbletea.

## Features

- View all your Azure DevOps work items in a clean terminal interface
- Navigate through tasks with keyboard shortcuts
- View detailed information about each work item
- Filters out closed and removed items automatically

## Prerequisites

- Go 1.21 or higher
- Azure CLI installed and configured
- Azure DevOps account

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

4. **Set up your Azure DevOps environment variables**:
```bash
export AZURE_DEVOPS_ORG_URL="https://dev.azure.com/your-organization"
export AZURE_DEVOPS_PROJECT="your-project-name"
```

To make this permanent, add these to your `~/.bashrc` or `~/.zshrc`.

## Building

```bash
go build -o hippo
```

## Running

```bash
./hippo
```

Or run directly:
```bash
go run .
```

## Keyboard Shortcuts

### List View
- `↑/k` - Move selection up
- `↓/j` - Move selection down
- `Enter` - View task details
- `q` - Quit

### Detail View
- `Esc/Backspace` - Return to list
- `q` - Quit

## Configuration

You can create a `.env` file in the project root or set environment variables:
```
AZURE_DEVOPS_ORG_URL=https://dev.azure.com/your-organization
AZURE_DEVOPS_PROJECT=your-project-name
```

The app uses Azure CLI for authentication, so no PAT is needed!

## Project Structure

```
.
├── main.go       # Main application and Bubbletea UI logic
├── client.go     # Azure DevOps API client
├── go.mod        # Go module definition
└── README.md     # This file
```

## Troubleshooting

### "failed to get Azure CLI token"
- Make sure you're logged in: `az login`
- Check your Azure CLI installation: `az --version`
- Verify you have access to Azure DevOps

### "AZURE_DEVOPS_* environment variable is not set"
Make sure you've exported both required environment variables (ORG_URL and PROJECT).

### "failed to query work items"
- Verify your organization URL is correct
- Ensure your project name matches exactly (case-sensitive)
- Make sure your Azure account has access to the Azure DevOps project

### Empty list
If you see "No tasks found", the query might not match any work items. Check that:
- Your project has work items
- The work items are not in "Closed" or "Removed" state

## Future Enhancements

- Filter by assigned user
- Filter by work item type
- Search functionality
- Update work item state
- Add comments
- Create new work items
- Sprint/iteration view
