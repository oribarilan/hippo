package main

import (
	"flag"
	"fmt"
)

// FlagConfig holds command-line flag values
type FlagConfig struct {
	OrganizationURL *string // Use pointers to detect if flag was explicitly set
	Project         *string
	Team            *string
	ConfigPath      *string // custom config file location
	ShowVersion     bool
	RunWizard       bool
	ShowHelp        bool
	DummyMode       bool // Internal: enable dummy backend for development
}

// parseFlags parses command-line flags and returns a FlagConfig
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
	// Undocumented flag for development - not shown in help
	flag.BoolVar(&flags.DummyMode, "dummy", false, "")

	flag.Parse()

	// Only set pointers if flags were actually provided
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

// printHelp prints usage information
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
