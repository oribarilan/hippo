# Deployment Configuration Implementation Plan

**Status:** Ready for Implementation  
**Created:** 2025-11-18  
**Updated:** 2025-11-18 (All decisions resolved)
**Goal:** Make Hippo deployment-ready with configuration file support

---

## Overview

Transform Hippo from development mode (env vars + .env files) to deployment-ready with configuration file support, while maintaining security best practices and great UX.

## Key Design Decisions

1. **Configuration sources with standard precedence:** Flags > Environment Variables > Config File
2. **Setup wizard creates config and exits** - No automatic wizard, user explicitly runs `--init`
3. **Config file versioning** - Enables future schema migrations
4. **Environment variables are permanent** - Not deprecated, useful for CI/CD and development
5. **Azure CLI only** - No PAT storage, maximum security
6. **Partial config merging** - Each source can provide some fields, merged by precedence

## Current State

**Authentication:**
- Uses Azure CLI (`az account get-access-token`) to fetch access tokens
- Token stored only in RAM during runtime, discarded on exit
- No persistent credentials stored by Hippo ✅ Secure

**Configuration (Environment Variables):**
- `AZURE_DEVOPS_ORG_URL` - Organization URL (e.g., `https://dev.azure.com/org-name`)
- `AZURE_DEVOPS_PROJECT` - Project name
- `AZURE_DEVOPS_TEAM` - Team name (optional, defaults to project)

**Key Files:**
- `app/main.go:14-15` - Loads `.env` using `godotenv`
- `app/model_init.go:56-57` - Reads env vars into model
- `app/client.go:30-45` - Reads env vars for client initialization
- `app/client_auth.go:12-36` - Gets token from Azure CLI

## Goals

1. ✅ Replace `.env` dependency with proper config file
2. ✅ Use cross-platform standard config locations
3. ✅ Keep Azure CLI as primary auth (most secure)
4. ✅ Implement first-run setup wizard
5. ✅ Support standard configuration precedence (flags > env > config file)
6. ✅ Security: Set proper file permissions, warn on insecure configs
7. ✅ Update all documentation

## Design Decisions

### Configuration Format: YAML ✅

**Why YAML:**
- Human-friendly (no JSON brackets/quotes clutter)
- Supports comments for inline documentation
- Standard for CLI tools (kubectl, docker-compose, gh, etc.)
- `gopkg.in/yaml.v3` is stable and well-maintained

### File Location

**Primary:** `~/.config/hippo/config.yaml` (XDG Base Directory spec)

**Cross-platform:**
- Use `os.UserConfigDir()` for standard location
- Windows: `%APPDATA%\hippo\config.yaml`
- macOS/Linux: `~/.config/hippo/config.yaml`

### Configuration Precedence (highest to lowest)

1. **Command-line flags** (e.g., `--org`, `--project`) - Quick overrides for single runs
2. **Environment variables** - For CI/CD, development, and containerized deployments
3. **Config file** - Primary persistent user configuration

**Rationale:**
This is the standard convention used by most CLI tools (Docker, kubectl, AWS CLI, gh, etc.) because:
- CLI flags are perfect for temporary overrides without changing files
- Environment variables enable CI/CD pipelines, Docker containers, and local dev environments
- Config files provide stable, persistent configuration for regular users
- Each level can override the one below it, giving maximum flexibility

**Configuration Loading Flow:**
1. Load config file (if exists and compatible)
2. Merge with environment variables (env vars override config file)
3. Merge with CLI flags (flags override everything)
4. Validate that all required fields are present
5. If validation fails → Show error and instruct user to run `hippo --init`

**First-Run Setup:**
- If no config file exists, show friendly error: "No configuration found. Run: hippo --init"
- Wizard (`--init`) creates config file and exits
- User runs `hippo` again, now loads the created config

**Example use cases:**
```bash
# First run - no config
./hippo
# Error: No configuration found. Run: hippo --init

# Run setup wizard
./hippo --init
# Creates ~/.config/hippo/config.yaml and exits

# Now works normally
./hippo

# Override project via environment variable (CI/CD)
export AZURE_DEVOPS_PROJECT="CI-Project"
./hippo

# Override for a single run via flag (testing)
./hippo --project "TestProject" --org "https://dev.azure.com/test-org"
```

### Configuration Versioning

**Strategy:**
- Each config file includes a `config_version` field
- Current version starts at 1
- Application checks compatibility on load
- If version is incompatible (too old or unknown), trigger setup wizard
- This enables smooth migrations when config schema changes in future versions

**Benefits:**
- Automatic migration for users upgrading Hippo
- No manual config file editing required
- Clear upgrade path for breaking changes
- Future-proof for new configuration options

### Configuration Schema

```yaml
# ~/.config/hippo/config.yaml

# Configuration version (for compatibility checking)
config_version: 1

# Azure DevOps connection (required)
organization_url: "https://dev.azure.com/your-org"
project: "your-project-name"
team: "your-team-name"  # optional, defaults to project

# Future optional settings
# default_sprint: "current"
# cache_duration: 300
```

### Security

1. **File Permissions:**
   - Set config file to `0600` (user read/write only)
   - Warn if permissions are too permissive

2. **Validation:**
   - Validate config structure on load
   - Clear error messages for missing/invalid fields

### Migration Strategy

**Decision: NO automatic migration support**
- Users manually copy 3 env var values into wizard
- Simpler code, less complexity
- Migration is trivial (3 values, one-time operation)
- Clear docs explain the process

## Implementation Plan

### Phase 1: Core Configuration Infrastructure

#### New Files to Create:

**1. `app/config.go`** - Configuration loading and management
```go
type Config struct {
    OrganizationURL string `yaml:"organization_url"`
    Project         string `yaml:"project"`
    Team            string `yaml:"team"`
    ConfigVersion   int    `yaml:"config_version"` // For future compatibility
}

const CurrentConfigVersion = 1

// Error types
var (
    ErrConfigNotFound     = errors.New("config file not found")
    ErrConfigIncompatible = errors.New("config version incompatible")
    ErrConfigInvalid      = errors.New("config file invalid")
)

// Functions:
- LoadConfig(flags *FlagConfig) (*Config, error)
- SaveConfig(config *Config) error
- GetConfigPath() (string, error)
- ValidateConfig(config *Config) error
- IsConfigVersionCompatible(config *Config) bool
- setConfigPermissions(path string) error
- checkConfigPermissions(path string) error
```

**Key behaviors:**
- Load config file first (if exists and compatible)
- Merge with env vars (only non-empty env vars override config file fields)
- Merge with flags (only explicitly set flags override everything)
- Each level can provide partial config (e.g., config file has org/project, env var overrides project)
- Validate merged config has all required fields
- Apply team default in validation (if empty, set to project name)
- If validation fails → Error with message "Run: hippo --init"
- Wizard only runs when explicitly invoked with `--init`
- Wizard creates/overwrites config file atomically and exits (doesn't start TUI)

**Config Merging Rules:**
1. Empty string in env var is **ignored** (doesn't override config file)
2. Empty string in flag **clears** config value (explicit override)
3. Config version 0 or missing = incompatible (trigger --init)
4. Unknown fields in config file = **ignored** (forward compatible)

**File Permission Handling:**
- Set config to 0600 on Unix-like systems (Windows: skip chmod)
- Check permissions on load and warn if too permissive (Unix only)
- Warning message shows current permissions and recommended fix

**2. `app/config_wizard.go`** - First-run setup wizard
```go
// Functions:
- RunConfigWizard() error
- promptInput(label, defaultValue string) string
- promptPassword(label string) string
- promptYesNo(question string, defaultYes bool) bool
```

**Wizard flow:**
1. Check if config file exists, show warning if overwriting (with current values + confirmation)
2. Welcome message
3. Prompt for Organization URL (with basic format validation: must start with https://)
4. Prompt for Project (required, non-empty validation)
5. Prompt for Team (optional, can be empty)
6. Show summary of configuration and ask for confirmation
7. Write to temporary file first
8. Validate temporary file
9. Set proper permissions on temp file
10. Atomic rename to config location
11. Success message with config file path
12. Exit (don't start TUI)

**Error Handling:**
- If any step fails, temp file is removed
- Original config (if exists) is never touched until success
- Ctrl+C during wizard cleans up temp file and exits
- Invalid input prompts retry (with helpful format hints)

**URL Validation:**
- Check that URL starts with `https://`
- Simple regex: `^https://`
- This prevents typos but allows flexibility for server installations
- If invalid, show error: "URL must start with https://" and retry

**3. `app/config_test.go`** - Unit tests
```go
// Test cases:
- TestLoadConfig_FromFile
- TestLoadConfig_WithEnvVarFallback
- TestLoadConfig_WithFlagOverrides
- TestConfigPrecedence
- TestConfigMerging_EmptyEnvVarIgnored
- TestConfigMerging_EmptyFlagClears
- TestValidateConfig_RequiredFields
- TestValidateConfig_TeamDefaultsToProject
- TestIsConfigVersionCompatible
- TestLoadConfig_IncompatibleVersion
- TestLoadConfig_UnversionedConfig
- TestGetConfigPath_CrossPlatform
- TestSaveConfig_FilePermissions
- TestSaveConfig_AtomicWrite
- TestCheckConfigPermissions_Unix
- TestCheckConfigPermissions_Windows
- TestConfigFileCorrupted
- TestConfigFileTruncated
- TestInvalidYAML
- TestUnknownFieldsIgnored
```

#### Files to Modify:

**4. `app/client.go`**

Changes needed:
```go
// OLD: Line 30-40
organizationURL := os.Getenv("AZURE_DEVOPS_ORG_URL")
project := os.Getenv("AZURE_DEVOPS_PROJECT")
team := os.Getenv("AZURE_DEVOPS_TEAM")

// NEW:
func NewAzureDevOpsClient(config *Config) (*AzureDevOpsClient, error) {
    organizationURL := config.OrganizationURL
    project := config.Project
    team := config.Team
    // ... rest unchanged
}
```

**5. `app/client_auth.go`**

No changes needed - keep existing Azure CLI implementation:
```go
// Existing implementation at Line 12
func getAzureCliToken() (string, error) {
    // ... existing implementation unchanged
}
```

**6. `app/main.go`**

Major changes to startup flow:
```go
func main() {
    // Keep godotenv for .env file support (useful for development)
    _ = godotenv.Load()
    _ = godotenv.Load("../.env")
    
    // 1. Parse CLI flags
    flags := parseFlags()
    
    // 2. Handle special flags
    if flags.ShowVersion {
        fmt.Println("Hippo v0.2.0")
        return
    }
    
    if flags.ShowHelp {
        printHelp()
        return
    }
    
    if flags.RunWizard {
        if err := RunConfigWizard(); err != nil {
            fmt.Printf("Setup failed: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("\nConfiguration saved! Run 'hippo' to start.")
        return
    }
    
    // 3. Load and merge configuration from all sources
    config, err := LoadConfig(flags)
    if err != nil {
        if errors.Is(err, ErrConfigNotFound) {
            fmt.Println("No configuration found.")
            fmt.Println("Run: hippo --init")
            os.Exit(1)
        } else if errors.Is(err, ErrConfigIncompatible) {
            fmt.Println("Configuration file is incompatible with this version.")
            fmt.Println("Run: hippo --init")
            os.Exit(1)
        } else {
            fmt.Printf("Configuration error: %v\n", err)
            os.Exit(1)
        }
    }
    
    // 4. Validate merged config has all required fields
    if err := ValidateConfig(config); err != nil {
        fmt.Printf("Configuration incomplete: %v\n", err)
        fmt.Println("Run: hippo --init")
        os.Exit(1)
    }
    
    // 5. Start TUI with config
    p := tea.NewProgram(initialModel(config), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v", err)
        os.Exit(1)
    }
}
```

**7. `app/model_init.go`**

Update to accept and use config:
```go
// OLD: Line 13
func initialModel() model

// NEW:
func initialModel(config *Config) model {
    // ... existing initialization ...
    
    return model{
        // Remove these lines:
        // organizationURL: os.Getenv("AZURE_DEVOPS_ORG_URL"),
        // projectName:     os.Getenv("AZURE_DEVOPS_PROJECT"),
        
        // Add:
        config: config,
        
        // ... rest unchanged
    }
}
```

Update Init() function:
```go
// OLD: Line 90
client, err := NewAzureDevOpsClient()

// NEW:
client, err := NewAzureDevOpsClient(m.config)
```

**8. `app/model.go`**

Update model struct:
```go
type model struct {
    // ADD:
    config *Config
    
    // REMOVE (replaced by config):
    // organizationURL string
    // projectName     string
    
    // ... rest unchanged
}
```

Update any references to `m.organizationURL` and `m.projectName` to use `m.config.OrganizationURL` and `m.config.Project` instead.

### Phase 2: Command-Line Interface

#### New Files:

**9. `app/flags.go`** - CLI flag parsing
```go
type FlagConfig struct {
    OrganizationURL *string  // Use pointers to detect if flag was explicitly set
    Project         *string
    Team            *string
    ConfigPath      *string  // custom config file location
    ShowVersion     bool
    RunWizard       bool
    ShowHelp        bool
}

func parseFlags() *FlagConfig {
    flags := &FlagConfig{}
    
    // Use helper variables for string flags
    var org, project, team, configPath string
    
    flag.StringVar(&org, "org", "", "Azure DevOps organization URL")
    flag.StringVar(&project, "project", "", "Azure DevOps project name")
    flag.StringVar(&team, "team", "", "Azure DevOps team name")
    flag.StringVar(&configPath, "config", "", "Path to config file")
    flag.BoolVar(&flags.ShowVersion, "version", false, "Show version")
    flag.BoolVar(&flags.RunWizard, "init", false, "Run configuration wizard")
    flag.BoolVar(&flags.ShowHelp, "help", false, "Show help")
    
    flag.Parse()
    
    // Only set pointers if flags were actually provided
    // (Check if they differ from flag default value AND flag was visited)
    flag.Visit(func(f *flag.Flag) {
        switch f.Name {
        case "org":
            flags.OrganizationURL = &org
        case "project":
            flags.Project = &project
        case "team":
            flags.Team = &team
        case "config":
            flags.ConfigPath = &configPath
        }
    })
    
    return flags
}

func printHelp() {
    fmt.Println("Hippo - Azure DevOps Work Item TUI")
    fmt.Println()
    fmt.Println("Usage:")
    fmt.Println("  hippo [flags]")
    fmt.Println()
    fmt.Println("Flags:")
    flag.PrintDefaults()
    fmt.Println()
    fmt.Println("Examples:")
    fmt.Println("  hippo                    # Start with config file")
    fmt.Println("  hippo --init             # Run setup wizard")
    fmt.Println("  hippo --project MyProj   # Override project for this run")
    fmt.Println()
    fmt.Println("Configuration:")
    fmt.Println("  Config file: ~/.config/hippo/config.yaml")
    fmt.Println("  Precedence: Flags > Environment Variables > Config File")
}
```

### Phase 3: Documentation & Examples

#### New Files:

**10. `app/config.example.yaml`** - Example configuration
```yaml
# Hippo Configuration File
# Location: ~/.config/hippo/config.yaml (macOS/Linux)
#           %APPDATA%\hippo\config.yaml (Windows)

# Configuration version (do not modify manually)
config_version: 1

# Azure DevOps Organization URL (required)
# Format: https://dev.azure.com/your-organization
organization_url: "https://dev.azure.com/example-org"

# Project name (required)
project: "MyProject"

# Team name (optional, defaults to project name)
team: "MyTeam"

# Future settings (not yet implemented)
# default_sprint: "current"
# cache_duration: 300
```

#### Files to Modify:

**11. `README.md`**

Major updates needed:

**Remove:**
- `.env` file setup instructions (but keep mention that `.env` files work via godotenv)
- Manual configuration setup steps
- Old environment variable-only workflows

**Add:**
```markdown
## Quick Start

1. **Install Azure CLI and login:**
   ```bash
   az login
   ```

2. **Run setup wizard:**
   ```bash
   ./hippo --init
   ```
   
   The wizard will prompt you for:
   - Azure DevOps organization URL (e.g., `https://dev.azure.com/your-org`)
   - Project name
   - Team name (optional)

3. **Start Hippo:**
   ```bash
   ./hippo
   ```

That's it! Your configuration is saved to `~/.config/hippo/config.yaml`.

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

### Configuration Sources & Precedence

Hippo supports multiple configuration sources with the following precedence (highest to lowest):

1. **Command-line flags** - Temporary overrides for single runs
2. **Environment variables** - For CI/CD, development, containers
3. **Config file** - Persistent user configuration

This means you can have a config file for daily use, but override values with environment variables in CI/CD or flags for quick tests.

### First-Time Setup

Run the interactive setup wizard:
```bash
hippo --init
```

This creates (or overwrites) your configuration file and exits. Then run `hippo` to start the application.

### Reconfiguring

To update your configuration, run the wizard again:
```bash
hippo --init
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

### Configuration Examples

**Example 1: Single project user**
```bash
# One-time setup
hippo --init

# Daily use
hippo
```

**Example 2: Multiple projects**
```bash
# Create config for main project
hippo --init

# Switch projects temporarily
hippo --project "OtherProject"

# Or use environment variables
export AZURE_DEVOPS_PROJECT="OtherProject"
hippo
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

# Note: The wizard requires interactive input and won't work in Docker CMD
# Use one of the above options instead
```
```

**Update Troubleshooting section:**
```markdown
## Troubleshooting

### "No configuration found"

Run the setup wizard:
```bash
hippo --init
```

This will create your configuration file at `~/.config/hippo/config.yaml`.

### "Configuration incomplete"

Your configuration is missing required fields (organization_url or project). Run:
```bash
hippo --init
```

Or manually edit your config file to add missing fields.

### "Configuration file is incompatible"

Your config file is from an older or newer version of Hippo. Run the wizard to update it:
```bash
hippo --init
```

### Configuration file location

Find your config file at:
- **macOS/Linux:** `~/.config/hippo/config.yaml`
- **Windows:** `%APPDATA%\hippo\config.yaml`

To use a custom location:
```bash
hippo --config /path/to/custom-config.yaml
```

### Authentication issues

Hippo uses Azure CLI for authentication. Ensure you're logged in:
```bash
az login
az account show  # Verify you're logged in
```

If you see "No subscriptions found", you may need to request access to your organization.

### Testing configuration

Test your configuration with flag overrides:
```bash
# Test with different project
hippo --project "TestProject"

# Test with different organization
hippo --org "https://dev.azure.com/test-org" --project "TestProject"
```
```

**12. `AGENTS.md`**

Updates:
```markdown
## Configuration

Hippo uses YAML configuration files stored in:
- macOS/Linux: `~/.config/hippo/config.yaml`
- Windows: `%APPDATA%\hippo\config.yaml`

Configuration loading and merging precedence (highest to lowest):
1. Command-line flags (temporary overrides)
2. Environment variables (CI/CD, development)
3. Config file (persistent configuration)

Setup wizard (`--init`):
- Creates or overwrites config file
- Exits after saving (doesn't start TUI)
- User runs `hippo` again to start with new config

Key files:
- `config.go` - Configuration loading, merging, and validation
- `config_wizard.go` - Interactive setup wizard (creates config and exits)
- `flags.go` - CLI flag parsing
- `client_auth.go` - Azure CLI token authentication
```

### Phase 4: Dependencies

**13. Update `app/go.mod`**

Add:
```go
require (
    gopkg.in/yaml.v3 v3.0.1
)
```

Keep:
```go
require (
    github.com/joho/godotenv v1.5.1  // For .env file support in development
)
```

**Note:** godotenv makes `.env` files convenient for local development. It's a small dependency with no security concerns for this use case.

**14. Run dependency update**
```bash
cd app
go get gopkg.in/yaml.v3
go mod tidy
```

### Phase 5: Testing

**Manual Testing Checklist:**
- [ ] First run with no config shows error and suggests `--init`
- [ ] `--init` flag runs wizard and creates config file
- [ ] Wizard shows warning if config file already exists
- [ ] Wizard validates organization URL format (retry on invalid)
- [ ] Wizard requires non-empty project name
- [ ] Wizard allows empty team name
- [ ] Wizard shows configuration summary before saving
- [ ] Wizard exits after creating config (doesn't start TUI)
- [ ] Config file has 0600 permissions (macOS/Linux)
- [ ] Config file includes config_version field
- [ ] Warning shown if config permissions too permissive (Unix)
- [ ] Running `hippo` after wizard starts normally with saved config
- [ ] Incompatible config version shows error and suggests `--init`
- [ ] Azure CLI authentication works
- [ ] Environment variables work and override config file values
- [ ] Empty environment variables are ignored (don't override config)
- [ ] Flag overrides work and override both config file and env vars (`--org`, `--project`)
- [ ] Empty flag values clear config values
- [ ] Partial config merging works (config file + env var + flag)
- [ ] Team field defaults to project name when not specified
- [ ] `--version` shows version
- [ ] `--help` shows help with examples
- [ ] Missing organization_url shows clear error and suggests `--init`
- [ ] Missing project shows clear error and suggests `--init`
- [ ] Config file only (no env/flags) works
- [ ] Env vars only (no config file) works
- [ ] Flags only (no config file) works
- [ ] Mixed config sources work correctly with precedence
- [ ] Ctrl+C during wizard cleans up properly
- [ ] Invalid YAML in config shows clear error
- [ ] Unknown fields in config are ignored
- [ ] `.env` file support still works via godotenv

**Unit Tests to Write:**
```bash
# Run all config tests
cd app
go test -run TestConfig -v

# Run specific tests
go test -run TestLoadConfig -v
go test -run TestValidateConfig -v
go test -run TestConfigPrecedence -v

# Check coverage
go test -cover
```

**Test cases:**
- Config loading from file
- Config loading with env var fallback
- Config loading with partial config from multiple sources
- Flag overrides (with pointer detection)
- Precedence order (flags > env > file)
- Merging configs from multiple sources
- Empty env var ignored (doesn't override)
- Empty flag clears value (explicit override)
- Validation of required fields
- Team field defaults to project name
- Config version compatibility checking
- Incompatible version detection
- Unversioned config detection (version 0 or missing)
- Unknown fields ignored (forward compatibility)
- Cross-platform path handling
- File permission setting (Unix-like systems)
- File permission warning (Unix-like systems)
- Missing config file handling
- Invalid YAML handling
- Corrupted config file handling
- Truncated config file handling
- Atomic write during save
- Wizard cleanup on interrupt

### Phase 6: Migration Guide

**15. Update `README.md` with migration section**

```markdown
## Migrating from .env Files

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
```

## Implementation Order

### Step-by-Step Sequence:

1. **Create config infrastructure** (config.go, config_test.go)
2. **Create wizard** (config_wizard.go)
3. **Add CLI flags** (flags.go)
4. **Update client** (client.go, client_auth.go)
5. **Update main** (main.go, model_init.go, model.go)
6. **Add dependencies** (go.mod)
7. **Write tests** (config_test.go, update existing tests)
8. **Update documentation** (README.md, AGENTS.md)
9. **Create examples** (config.example.yaml)
10. **Manual testing** (full walkthrough)

## Success Criteria

- [ ] Config file created on first run with wizard
- [ ] Config file includes version field for future compatibility
- [ ] Config file has secure permissions (0600 on Unix)
- [ ] Incompatible config versions trigger wizard automatically
- [ ] Azure CLI auth works without changes
- [ ] Environment variables work as permanent feature (not deprecated)
- [ ] Configuration precedence works correctly (flags > env > config file)
- [ ] Partial config merging works (e.g., config file + env var override)
- [ ] CLI flags override all other config sources
- [ ] Clear error messages for all failure modes
- [ ] All tests pass
- [ ] Documentation fully updated
- [ ] Example config file provided

## Timeline Estimate

- **Phase 1** (Core Config): 4-5 hours
  - config.go: 2 hours (including merging logic and edge cases)
  - config_wizard.go: 2 hours (including validation and atomic writes)
  - Tests: 1 hour

- **Phase 2** (CLI Flags): 1.5 hours
  - flags.go with pointer detection: 1 hour
  - Integration: 30 min

- **Phase 3** (Update Existing Code): 2-3 hours
  - client.go, client_auth.go: 1 hour
  - main.go, model*.go: 1-1.5 hours
  - Testing changes: 30 min

- **Phase 4** (Documentation): 1.5-2 hours
  - README.md updates: 1 hour
  - AGENTS.md updates: 30 min
  - Example config: 30 min

- **Phase 5** (Testing): 2.5-3.5 hours
  - Unit tests: 1.5 hours (more comprehensive coverage)
  - Integration testing: 1-2 hours

**Total: 12-15 hours**

## Future Enhancements (Post-MVP)

### v0.3.0+
- [ ] Multiple profiles support
- [ ] OS keychain integration for PAT storage
- [ ] Config commands (`hippo config show`, `hippo config edit`)
- [ ] Auto-update checking
- [ ] Custom theme support

### Nice-to-Have
- [ ] Config validation command
- [ ] Config export/import
- [ ] Team/organization templates

## Risks & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Windows path handling differs | Medium | Medium | Use `os.UserConfigDir()`, test on Windows |
| Windows file permissions differ | Low | Medium | Document Windows security, skip chmod on Windows |
| Breaking changes for existing users | High | Low | Keep env var support, clear migration docs |
| Wizard UX issues | Medium | Low | Keep simple, test with users |

## Questions & Decisions

### Resolved:
- ✅ Config format: YAML (vs JSON, TOML)
- ✅ Config location: XDG Base Directory (~/.config/hippo/)
- ✅ Migration support: NO automatic migration (manual via wizard)
- ✅ Wizard style: Plain prompts (vs full TUI)
- ✅ Environment variables: Permanent feature (not deprecated)
- ✅ Configuration precedence: Flags > Env Vars > Config File (standard convention)
- ✅ URL validation: Basic validation (https:// prefix check)
- ✅ Wizard overwrite: Show warning + require confirmation
- ✅ Empty flag behavior: Clears value (explicit override)

### Open:
- ⏳ When to remove godotenv dependency? (Suggestion: Keep it - useful for development with .env files)
- ⏳ Should we add `--migrate` command later? (Suggestion: Only if users request it)

### Recently Resolved:

**✅ RESOLVED: URL Validation in Wizard**
- **Decision:** Basic validation (Option A)
- Check URL starts with `https://` 
- Allow any organization name
- Simple regex: `^https://`
- Prevents typos but allows flexibility for server installations

**✅ RESOLVED: Wizard Overwrite Confirmation**
- **Decision:** Show warning + confirmation (Option A)
- Display current configuration values
- Require explicit confirmation before overwriting
- User can abort with Ctrl+C or 'n'

**✅ RESOLVED: Empty Flag Behavior**
- **Decision:** Clear the value (Option A)
- Empty flag explicitly clears/overrides the config value
- Config has `team: "TeamA"`, flag `--team ""` results in empty team
- Explicit empty means "I want no value for this field"

## Notes

- Keep implementation simple and focused
- Prioritize security (file permissions, no token logging)
- Maintain great first-run UX (wizard is key)
- Clear error messages at every step
- Test cross-platform thoroughly

### Config Version Compatibility Logic

**Version checking should:**
1. Missing or 0 version → Incompatible (trigger wizard)
2. Version > CurrentConfigVersion → Incompatible (user downgraded Hippo, trigger wizard)
3. Version < CurrentConfigVersion → Check if migrations available, otherwise incompatible
4. Version == CurrentConfigVersion → Compatible, proceed normally

**For initial release (v1):**
- Only version 1 is compatible
- Any other version (missing, 0, 2+) triggers wizard
- Future versions can add migration logic from v1 → v2, etc.

### Config Version Migrations (Future Enhancements)

**Not implemented in v1, but architecture should support:**

For v1 → v2+ migrations, we'll eventually need:
1. Migration function registry: `map[int]func(*Config) error`
2. Sequential migration: v1→v2→v3 (not v1→v3 directly)
3. Backup before migration
4. Rollback on failure
5. Migration functions in separate file: `config_migrations.go`

**v1 Implementation:** No migrations, just version checking. Force users to run `--init` for incompatible versions.

---

**Plan Version:** 1.0  
**Last Updated:** 2025-11-18  
**Next Review:** After Phase 1 completion
