package main

import (
	"fmt"
	"os"
	"rocha/cmd"
	"rocha/tmux"
	"rocha/version"

	"github.com/alecthomas/kong"
)

func main() {
	// Create tmux client for dependency injection
	tmuxClient := tmux.NewClient()

	// Parse CLI arguments with Kong
	var cli cmd.CLI
	ctx := kong.Parse(&cli,
		kong.Name("rocha"),
		kong.Description(version.Tagline),
		kong.UsageOnError(),
		kong.BindTo(tmuxClient, (*tmux.Client)(nil)),
	)

	// Execute the selected command
	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
