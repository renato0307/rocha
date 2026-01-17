package main

import (
	"fmt"
	"os"
	"rocha/cmd"
	"rocha/version"

	"github.com/alecthomas/kong"
)

func main() {
	// Parse CLI arguments with Kong
	var cli cmd.CLI
	ctx := kong.Parse(&cli,
		kong.Name("rocha"),
		kong.Description(version.Tagline),
		kong.UsageOnError(),
	)

	// Execute the selected command
	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
