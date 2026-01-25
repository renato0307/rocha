package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/renato0307/rocha/internal/cmd"
	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/ui"
)

// Build information injected at build time via ldflags
// Example: -ldflags="-X main.Version=v1.0.0 -X main.Commit=abc123 ..."
var (
	Commit    = "unknown"
	Date      = "unknown"
	GoVersion = "unknown"
	Version   = "dev"
)

// Tagline is the application's tagline used in help text and documentation
const Tagline = "I'm Rocha, and I manage coding agents"

// versionInfo returns formatted version information for CLI display
func versionInfo() string {
	return fmt.Sprintf("rocha %s (commit: %s, built: %s, go: %s)",
		Version, Commit, Date, GoVersion)
}

func main() {
	// Set version info for UI components
	ui.SetVersionInfo(ui.VersionInfo{
		Commit:    Commit,
		Date:      Date,
		GoVersion: GoVersion,
		Tagline:   Tagline,
		Version:   Version,
	})

	// Load settings from ~/.rocha/settings.json
	settings, err := config.LoadSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load settings: %v\n", err)
		settings = &config.Settings{} // Use empty settings
	}

	// Parse CLI arguments with Kong
	// Container is created in CLI.AfterApply() after logging is initialized
	var cli cmd.CLI
	cli.SetSettings(settings) // Set settings before parsing
	ctx := kong.Parse(&cli,
		kong.Name("rocha"),
		kong.Description(Tagline),
		kong.Vars{
			"version": versionInfo(),
		},
		kong.UsageOnError(),
		kong.Bind(&cli),
	)
	defer cli.Close()

	// Execute the selected command
	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
