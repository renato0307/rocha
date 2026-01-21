package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rocha/cmd"
	"rocha/config"
	"rocha/tmux"
	"rocha/version"

	"github.com/alecthomas/kong"
)

func main() {
	// PHASE 1: Bootstrap - determine profile from CLI/env only (not settings yet)
	// This is needed because settings location depends on ROCHA_HOME, which depends on profile
	bootstrapProfile := getBootstrapProfile()

	// PHASE 2: Set ROCHA_HOME if bootstrap profile exists and is not default
	if bootstrapProfile != "" && bootstrapProfile != "default" {
		if _, hasEnv := os.LookupEnv("ROCHA_HOME"); !hasEnv {
			homeDir, _ := os.UserHomeDir()
			rochaHome := filepath.Join(homeDir, fmt.Sprintf(".rocha_%s", bootstrapProfile))
			os.Setenv("ROCHA_HOME", rochaHome)
		}
	}

	// PHASE 3: Now load settings from the correct ROCHA_HOME
	settings, err := config.LoadSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load settings: %v\n", err)
		settings = &config.Settings{} // Use empty settings
	}

	// Create tmux client for dependency injection
	tmuxClient := tmux.NewClient()

	// Parse CLI arguments with Kong
	var cli cmd.CLI
	cli.SetSettings(settings) // Set settings before parsing
	ctx := kong.Parse(&cli,
		kong.Name("rocha"),
		kong.Description(version.Tagline),
		kong.Vars{
			"version": version.Info(),
		},
		kong.UsageOnError(),
		kong.BindTo(tmuxClient, (*tmux.Client)(nil)),
		kong.BindTo(&cli, (*cmd.CLI)(nil)),
	)

	// Execute the selected command
	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// getBootstrapProfile manually parses CLI args for --profile flag
// This runs before Kong parsing to determine ROCHA_HOME location
func getBootstrapProfile() string {
	// Check command line args manually
	for i, arg := range os.Args {
		if arg == "--profile" || arg == "-p" {
			if i+1 < len(os.Args) {
				return os.Args[i+1]
			}
		}
		if strings.HasPrefix(arg, "--profile=") {
			return strings.TrimPrefix(arg, "--profile=")
		}
	}

	// Check environment variable
	return os.Getenv("ROCHA_PROFILE")
}
