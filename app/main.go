package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (ignore error if it doesn't)
	// Try current directory first, then parent directory
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	// 1. Parse CLI flags
	flags := parseFlags()

	// 2. Handle special flags
	if flags.ShowVersion {
		fmt.Printf("Hippo %s\n", Version)
		return
	}

	if flags.ShowHelp {
		printHelp()
		return
	}

	// 3. Check for dummy mode (flag or environment variable)
	dummyMode := flags.DummyMode || os.Getenv("HIPPO_DUMMY_MODE") == "true"

	// 4. If dummy mode, skip config and use dummy backend
	if dummyMode {
		m := initialModelWithDummyBackend()
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
		return
	}

	// 5. Load and merge configuration from all sources
	config, configSource, err := LoadConfig(flags)

	// 6. Determine if we need to run wizard
	needsWizard := false
	var existingConfig *Config
	var existingConfigSource *ConfigSource

	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			needsWizard = true
			existingConfig = nil
			existingConfigSource = nil
		} else if errors.Is(err, ErrConfigIncompatible) {
			needsWizard = true
			// Try to load existing config for pre-filling
			configPath, _ := getConfigPath(flags)
			existingConfig, _ = loadConfigFile(configPath)
			existingConfigSource = nil // Don't track source for incompatible config
		} else {
			fmt.Printf("Configuration error: %v\n", err)
			os.Exit(1)
		}
	} else if err := ValidateConfig(config); err != nil {
		needsWizard = true
		existingConfig = config
		existingConfigSource = configSource
	}

	// 7. If --init flag is set, force wizard mode
	if flags.RunWizard {
		needsWizard = true
		existingConfig = config
		existingConfigSource = configSource
	}

	// 8. Start TUI (either with wizard or normal mode)
	var m model
	if needsWizard {
		m = initialModelWithWizard(existingConfig, existingConfigSource)
	} else {
		m = initialModel(config, configSource)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
