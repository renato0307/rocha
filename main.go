package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"rocha/cmd"
	"rocha/config"
	"rocha/ports"
	"rocha/adapters/tmux"
	"rocha/version"
)

func main() {
	// Load settings from ~/.rocha/settings.json
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
		kong.BindTo(tmuxClient, (*ports.TmuxClient)(nil)),
		kong.BindTo(&cli, (*cmd.CLI)(nil)),
	)

	// Execute the selected command
	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
