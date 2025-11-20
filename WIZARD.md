# Bubbletea Configuration Wizard

## Overview

The configuration wizard has been completely rewritten to use Bubbletea, matching Hippo's UI style and providing a consistent user experience.

## Features

### Visual Design
- **Title Bar**: Purple background with white text (matching Hippo's main UI)
- **Field Labels**: Green and bold when focused, purple when active
- **Input Fields**: Uses Bubbletea's `textinput` component with 60-char width
- **Error Messages**: Red and bold with ✗ symbol
- **Help Footer**: Gray text with key bindings
- **Existing Config Warning**: Orange warning with current values displayed

### Navigation
- **↑/↓ or Tab/Shift+Tab**: Navigate between fields
- **Enter**: Move to next field or save configuration
- **Esc**: Cancel wizard and exit

### Fields

1. **Organization URL**
   - Required field
   - Must start with `https://`
   - Placeholder: `https://dev.azure.com/your-org`
   - Hint: "e.g., https://dev.azure.com/your-org"

2. **Project Name**
   - Required field
   - Placeholder: `MyProject`
   - Hint: "The name of your Azure DevOps project"

3. **Team Name**
   - Optional field (defaults to project name)
   - Placeholder: `MyTeam (optional, defaults to project name)`
   - Hint: "Optional - defaults to project name"

4. **Save Configuration**
   - Green background when focused
   - Validates all fields before saving
   - Shows errors inline if validation fails

### Validation

- **Organization URL**: Must be non-empty and start with `https://`
- **Project**: Must be non-empty
- **Team**: Optional, no validation

If validation fails, the wizard:
1. Shows an error message at the bottom
2. Returns focus to the invalid field
3. Allows the user to correct the input

### Existing Configuration

If a config file already exists:
- Shows a warning at the top in orange
- Displays current configuration values
- Pre-fills form fields with existing values
- Note: "Current settings will be overwritten if you continue."

## Usage

### From Code

```go
// Run the wizard
if err := RunConfigWizard(); err != nil {
    fmt.Printf("Setup failed: %v\n", err)
    os.Exit(1)
}
```

### User Experience

**First Run:**
```
╔══════════════════════════════════════════════════════════════════════════════════╗
║ Hippo Configuration Wizard                                                       ║
╚══════════════════════════════════════════════════════════════════════════════════╝

Please provide your Azure DevOps configuration:

Organization URL:    [https://dev.azure.com/myorg________________]
                     e.g., https://dev.azure.com/your-org

Project Name:        MyProject

Team Name:           

  Save Configuration

────────────────────────────────────────────────────────────────────────────────────
↑/↓, tab/shift+tab navigate  •  enter next/save  •  esc cancel
```

**With Existing Config:**
```
╔══════════════════════════════════════════════════════════════════════════════════╗
║ Hippo Configuration Wizard                                                       ║
╚══════════════════════════════════════════════════════════════════════════════════╝

⚠ Configuration file already exists
  Current settings will be overwritten if you continue.

  Organization: https://dev.azure.com/oldorg
  Project:      OldProject
  Team:         OldTeam

Please provide your Azure DevOps configuration:

[form fields...]
```

**With Validation Error:**
```
[form fields...]

✗ URL must start with https://

────────────────────────────────────────────────────────────────────────────────────
```

## Implementation Details

### Model Structure

```go
type wizardModel struct {
    currentField   wizardField      // Current focused field
    orgInput       textinput.Model  // Organization URL input
    projectInput   textinput.Model  // Project input
    teamInput      textinput.Model  // Team input
    existingConfig *Config          // Existing config (if any)
    err            string           // Current error message
    styles         Styles           // Hippo's styles
    confirmed      bool             // True if user confirmed
    cancelled      bool             // True if user cancelled
}
```

### Key Methods

- `initialWizardModel()`: Creates wizard with pre-filled values if config exists
- `Update()`: Handles keyboard navigation and input
- `View()`: Renders the wizard UI with Hippo's styling
- `updateFocus()`: Manages focus between input fields
- `renderField()`: Renders a single form field with label and hint

### Integration with Main App

The wizard integrates seamlessly with Hippo's main.go:

```go
// Auto-run wizard on first run
if errors.Is(err, ErrConfigNotFound) {
    fmt.Println("No configuration found. Starting setup wizard...\n")
    if err := RunConfigWizard(); err != nil {
        fmt.Printf("Setup failed: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("\nConfiguration saved! Starting Hippo...\n")
    // ... reload config and start TUI
}
```

## Benefits

1. **Consistency**: Matches Hippo's visual style perfectly
2. **Better UX**: Familiar navigation for Hippo users
3. **Validation**: Real-time error feedback
4. **Keyboard-First**: Full keyboard navigation (no mouse needed)
5. **Pre-filled Values**: Shows existing config when updating
6. **Clear State**: Visual indicators for focused fields
7. **Professional Look**: Beautiful TUI that looks native to Hippo

## Testing

To test the wizard:

```bash
cd app
go build -o hippo

# Test first run
rm ~/.config/hippo/config.yaml 2>/dev/null
./hippo

# Test forced reconfiguration
./hippo --init

# Test with existing config
./hippo --init
```
