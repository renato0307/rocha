package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/renato0307/rocha/internal/logging"
)

// SessionsViewAgentSettingsCmd displays actual agent settings from running process
type SessionsViewAgentSettingsCmd struct {
	Name string `arg:"" help:"Name of the session"`
}

// Run executes the view-agent-settings command
func (s *SessionsViewAgentSettingsCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Viewing agent settings", "session", s.Name)

	ctx := context.Background()

	// Call service (service uses ProcessInspector port)
	settingsJSON, err := cli.Container.SessionService.GetAgentSettings(ctx, s.Name)
	if err != nil {
		return fmt.Errorf("failed to get agent settings: %w", err)
	}

	// Parse and pretty-print JSON
	var settings map[string]any
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return fmt.Errorf("failed to parse settings JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	fmt.Println(string(prettyJSON))

	return nil
}
