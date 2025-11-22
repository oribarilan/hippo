# Testing Instructions

## Step 1: Download Dependencies

First, download the yaml.v3 dependency:

```bash
cd app
go mod download
```

Or run:

```bash
cd app
go mod tidy
```

## Step 2: Run the Tests

```bash
cd app
go test -v
```

## Expected Test Results

All tests should pass. The test suite includes:

1. **TestValidateConfig** - Tests config validation and team defaulting
2. **TestIsConfigVersionCompatible** - Tests version compatibility checking
3. **TestLoadConfig_Precedence** - Tests config loading with proper precedence (flags > env > file)
4. **TestSaveConfig** - Tests config file saving with proper permissions
5. **TestLoadConfig_NotFound** - Tests error handling when config file doesn't exist
6. **TestLoadConfig_IncompatibleVersion** - Tests incompatible version detection

## Potential Issues and Fixes

### Issue 1: Missing go.sum entry

**Error:**
```
missing go.sum entry for module providing package gopkg.in/yaml.v3
```

**Fix:**
```bash
cd app
go mod tidy
```

### Issue 2: Test failures due to environment variables

If tests fail because environment variables are set in your shell, you can either:

1. Unset them temporarily:
```bash
unset HIPPO_ADO_ORG_URL
unset HIPPO_ADO_PROJECT
unset HIPPO_ADO_TEAM
go test -v
```

2. Or the tests should handle this automatically now.

### Issue 3: Permission issues on config file tests

On Windows, the permission tests might behave differently. This is expected and the code handles it (checks are skipped on Windows).

## Step 3: Build the Application

After tests pass, build the application:

```bash
cd app
go build -o hippo
```

## Step 4: Test the Auto-Wizard Flow

### Test Scenario 1: First Run (No Config)

```bash
# Make sure no config exists
rm ~/.config/hippo/config.yaml 2>/dev/null || true

# Run hippo - should automatically start wizard
./hippo
```

**Expected:**
- Wizard starts automatically
- Prompts for org URL, project, team
- Saves config
- Prints "Configuration saved! Starting Hippo..."
- Starts the TUI

### Test Scenario 2: Force Reconfiguration

```bash
# Run with --init flag to force wizard
./hippo --init
```

**Expected:**
- Shows current configuration
- Asks for confirmation to overwrite
- Runs wizard
- Saves config
- Prints "Configuration saved! Run 'hippo' to start."
- Does NOT start TUI (exits after wizard)

### Test Scenario 3: Normal Run (Config Exists)

```bash
# Run hippo with existing config
./hippo
```

**Expected:**
- Loads config file
- Starts TUI directly
- No wizard

### Test Scenario 4: Incompatible Config Version

```bash
# Manually create incompatible config
cat > ~/.config/hippo/config.yaml << EOF
config_version: 999
organization_url: "https://dev.azure.com/test"
project: "test"
EOF

# Run hippo
./hippo
```

**Expected:**
- Detects incompatible version
- Prints "Configuration file is incompatible with this version."
- Automatically starts wizard
- Updates config
- Starts TUI

## Troubleshooting

If any test fails, please share the error output and I can help diagnose whether it's a test issue or an implementation issue.
