package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	ConfigVersion   int    `yaml:"config_version"`
	OrganizationURL string `yaml:"organization_url"`
	Project         string `yaml:"project"`
	Team            string `yaml:"team"`
}

// ConfigSource tracks the source of each configuration value
type ConfigSource struct {
	OrganizationURL string // "file", "env", "flag", or ""
	Project         string
	Team            string
	ConfigPath      string // The actual path to the config file (for display)
}

const CurrentConfigVersion = 1

// Error types
var (
	ErrConfigNotFound     = errors.New("config file not found")
	ErrConfigIncompatible = errors.New("config version incompatible")
	ErrConfigInvalid      = errors.New("config file invalid")
)

// LoadConfig loads configuration from file, environment variables, and flags
// Precedence: Flags > Environment Variables > Config File
// Returns the loaded config and source tracking information
func LoadConfig(flags *FlagConfig) (*Config, *ConfigSource, error) {
	config := &Config{}
	source := &ConfigSource{}

	// 1. Load config file (if exists)
	configPath, err := getConfigPath(flags)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get config path: %w", err)
	}

	fileConfig, err := loadConfigFile(configPath)
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			// No config file - check if we can proceed with env/flags
			fileConfig = &Config{}
		} else {
			return nil, nil, err
		}
	}

	// Check version compatibility if config file exists
	if fileConfig.ConfigVersion != 0 {
		if !isConfigVersionCompatible(fileConfig) {
			return nil, nil, ErrConfigIncompatible
		}
		*config = *fileConfig

		// Track that values came from file
		if config.OrganizationURL != "" {
			source.OrganizationURL = "file"
		}
		if config.Project != "" {
			source.Project = "file"
		}
		if config.Team != "" {
			source.Team = "file"
		}
		source.ConfigPath = configPath
	}

	// 2. Merge with environment variables (non-empty overrides)
	if orgURL := os.Getenv("HIPPO_ADO_ORG_URL"); orgURL != "" {
		config.OrganizationURL = orgURL
		source.OrganizationURL = "env"
	}
	if project := os.Getenv("HIPPO_ADO_PROJECT"); project != "" {
		config.Project = project
		source.Project = "env"
	}
	if team := os.Getenv("HIPPO_ADO_TEAM"); team != "" {
		config.Team = team
		source.Team = "env"
	}

	// 3. Merge with CLI flags (explicit flags override everything)
	if flags.OrganizationURL != nil {
		config.OrganizationURL = *flags.OrganizationURL
		source.OrganizationURL = "flag"
	}
	if flags.Project != nil {
		config.Project = *flags.Project
		source.Project = "flag"
	}
	if flags.Team != nil {
		config.Team = *flags.Team
		source.Team = "flag"
	}

	// If no config was loaded from any source, return not found error
	if config.OrganizationURL == "" && config.Project == "" && fileConfig.ConfigVersion == 0 {
		return nil, nil, ErrConfigNotFound
	}

	return config, source, nil
}

// loadConfigFile loads configuration from a YAML file
func loadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigInvalid, err)
	}

	// Check permissions on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := checkConfigPermissions(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	return &config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to temporary file first (atomic write)
	tempPath := configPath + ".tmp"
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Set permissions on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := setConfigPermissions(tempPath); err != nil {
			os.Remove(tempPath)
			return err
		}
	}

	// Atomic rename
	if err := os.Rename(tempPath, configPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	return filepath.Join(configDir, "hippo", "config.yaml"), nil
}

// getConfigPath returns the config path, using custom path from flags if provided
func getConfigPath(flags *FlagConfig) (string, error) {
	if flags.ConfigPath != nil && *flags.ConfigPath != "" {
		return *flags.ConfigPath, nil
	}
	return GetConfigPath()
}

// ValidateConfig validates that all required fields are present
func ValidateConfig(config *Config) error {
	if config.OrganizationURL == "" {
		return fmt.Errorf("organization_url is required")
	}
	if config.Project == "" {
		return fmt.Errorf("project is required")
	}
	if config.Team == "" {
		return fmt.Errorf("team is required")
	}

	return nil
}

// isConfigVersionCompatible checks if the config version is compatible
func isConfigVersionCompatible(config *Config) bool {
	// Version must be non-zero and match current version
	if config.ConfigVersion == 0 {
		return false
	}
	if config.ConfigVersion != CurrentConfigVersion {
		return false
	}
	return true
}

// setConfigPermissions sets the config file to user read/write only (0600)
func setConfigPermissions(path string) error {
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}
	return nil
}

// checkConfigPermissions checks if config file permissions are secure
func checkConfigPermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil // File doesn't exist or can't be checked
	}

	mode := info.Mode().Perm()
	// Check if group or others have any permissions
	if mode&0077 != 0 {
		return fmt.Errorf("config file has insecure permissions %04o, should be 0600. Run: chmod 600 %s", mode, path)
	}

	return nil
}
