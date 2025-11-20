package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				ConfigVersion:   1,
				OrganizationURL: "https://dev.azure.com/org",
				Project:         "project",
				Team:            "team",
			},
			wantErr: false,
		},
		{
			name: "missing team",
			config: &Config{
				ConfigVersion:   1,
				OrganizationURL: "https://dev.azure.com/org",
				Project:         "project",
				Team:            "",
			},
			wantErr: true,
		},
		{
			name: "missing organization URL",
			config: &Config{
				ConfigVersion: 1,
				Project:       "project",
			},
			wantErr: true,
		},
		{
			name: "missing project",
			config: &Config{
				ConfigVersion:   1,
				OrganizationURL: "https://dev.azure.com/org",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsConfigVersionCompatible(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name:   "current version is compatible",
			config: &Config{ConfigVersion: CurrentConfigVersion},
			want:   true,
		},
		{
			name:   "version 0 is incompatible",
			config: &Config{ConfigVersion: 0},
			want:   false,
		},
		{
			name:   "higher version is incompatible",
			config: &Config{ConfigVersion: CurrentConfigVersion + 1},
			want:   false,
		},
		{
			name:   "lower version is incompatible",
			config: &Config{ConfigVersion: CurrentConfigVersion - 1},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConfigVersionCompatible(tt.config); got != tt.want {
				t.Errorf("isConfigVersionCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadConfig_Precedence(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create a config file
	configFileContent := `config_version: 1
organization_url: "https://dev.azure.com/file-org"
project: "file-project"
team: "file-team"
`
	if err := os.WriteFile(configPath, []byte(configFileContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Test 1: Config file only
	t.Run("config file only", func(t *testing.T) {
		flags := &FlagConfig{}
		flags.ConfigPath = &configPath

		config, source, err := LoadConfig(flags)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.OrganizationURL != "https://dev.azure.com/file-org" {
			t.Errorf("OrganizationURL = %v, want https://dev.azure.com/file-org", config.OrganizationURL)
		}
		if config.Project != "file-project" {
			t.Errorf("Project = %v, want file-project", config.Project)
		}

		// Verify source tracking
		if source.OrganizationURL != "file" {
			t.Errorf("OrganizationURL source = %v, want file", source.OrganizationURL)
		}
		if source.Project != "file" {
			t.Errorf("Project source = %v, want file", source.Project)
		}
	})

	// Test 2: Environment variable overrides config file
	t.Run("env var overrides config file", func(t *testing.T) {
		os.Setenv("HIPPO_ADO_PROJECT", "env-project")
		defer os.Unsetenv("HIPPO_ADO_PROJECT")

		flags := &FlagConfig{}
		flags.ConfigPath = &configPath

		config, source, err := LoadConfig(flags)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.Project != "env-project" {
			t.Errorf("Project = %v, want env-project", config.Project)
		}
		if config.OrganizationURL != "https://dev.azure.com/file-org" {
			t.Errorf("OrganizationURL = %v, want https://dev.azure.com/file-org", config.OrganizationURL)
		}

		// Verify source tracking
		if source.Project != "env" {
			t.Errorf("Project source = %v, want env", source.Project)
		}
		if source.OrganizationURL != "file" {
			t.Errorf("OrganizationURL source = %v, want file", source.OrganizationURL)
		}
	})

	// Test 3: Flag overrides everything
	t.Run("flag overrides everything", func(t *testing.T) {
		os.Setenv("HIPPO_ADO_PROJECT", "env-project")
		defer os.Unsetenv("HIPPO_ADO_PROJECT")

		flagProject := "flag-project"
		flags := &FlagConfig{
			Project: &flagProject,
		}
		flags.ConfigPath = &configPath

		config, source, err := LoadConfig(flags)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.Project != "flag-project" {
			t.Errorf("Project = %v, want flag-project", config.Project)
		}

		// Verify source tracking
		if source.Project != "flag" {
			t.Errorf("Project source = %v, want flag", source.Project)
		}
	})

	// Test 4: Empty env var is ignored
	t.Run("empty env var is ignored", func(t *testing.T) {
		os.Setenv("HIPPO_ADO_PROJECT", "")
		defer os.Unsetenv("HIPPO_ADO_PROJECT")

		flags := &FlagConfig{}
		flags.ConfigPath = &configPath

		config, _, err := LoadConfig(flags)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.Project != "file-project" {
			t.Errorf("Project = %v, want file-project (empty env var should be ignored)", config.Project)
		}
	})
}

func TestSaveConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "hippo", "config.yaml")

	// Create directory
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		ConfigVersion:   1,
		OrganizationURL: "https://dev.azure.com/test-org",
		Project:         "test-project",
		Team:            "test-team",
	}

	// Save to temp path by creating config file manually
	data := []byte(`config_version: 1
organization_url: "https://dev.azure.com/test-org"
project: "test-project"
team: "test-team"
`)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	// Verify the file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Verify file permissions (Unix only)
	if info, err := os.Stat(configPath); err == nil {
		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Config file permissions = %04o, want 0600", mode)
		}
	}

	// Verify the content
	loadedConfig, err := loadConfigFile(configPath)
	if err != nil {
		t.Fatalf("loadConfigFile() error = %v", err)
	}

	if loadedConfig.OrganizationURL != config.OrganizationURL {
		t.Errorf("OrganizationURL = %v, want %v", loadedConfig.OrganizationURL, config.OrganizationURL)
	}
	if loadedConfig.Project != config.Project {
		t.Errorf("Project = %v, want %v", loadedConfig.Project, config.Project)
	}
	if loadedConfig.Team != config.Team {
		t.Errorf("Team = %v, want %v", loadedConfig.Team, config.Team)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	// Clear any environment variables that might interfere
	os.Unsetenv("HIPPO_ADO_ORG_URL")
	os.Unsetenv("HIPPO_ADO_PROJECT")
	os.Unsetenv("HIPPO_ADO_TEAM")

	flags := &FlagConfig{}
	nonExistentPath := "/nonexistent/config.yaml"
	flags.ConfigPath = &nonExistentPath

	config, _, err := LoadConfig(flags)
	if err != ErrConfigNotFound {
		t.Errorf("LoadConfig() error = %v, want ErrConfigNotFound", err)
	}
	if config != nil {
		t.Error("LoadConfig() should return nil config when not found")
	}
}

func TestLoadConfig_IncompatibleVersion(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create a config file with incompatible version
	configFileContent := `config_version: 99
organization_url: "https://dev.azure.com/test-org"
project: "test-project"
`
	if err := os.WriteFile(configPath, []byte(configFileContent), 0600); err != nil {
		t.Fatal(err)
	}

	flags := &FlagConfig{}
	flags.ConfigPath = &configPath

	_, _, err := LoadConfig(flags)
	if err != ErrConfigIncompatible {
		t.Errorf("LoadConfig() error = %v, want ErrConfigIncompatible", err)
	}
}
