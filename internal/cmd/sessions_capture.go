package cmd

import (
	"context"
	"fmt"

	"rocha/internal/logging"
)

// SessionsCaptureCmd captures the content of a session's tmux pane
type SessionsCaptureCmd struct {
	Lines int    `help:"Number of lines to capture" default:"50" short:"n"`
	Name  string `arg:"" help:"Session name"`
}

// Run executes the capture command
func (s *SessionsCaptureCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Executing sessions capture command", "name", s.Name, "lines", s.Lines)

	container, err := NewContainer(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	ctx := context.Background()

	// Validate session exists in database
	if _, err := container.SessionService.GetSession(ctx, s.Name); err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Check if tmux session is running
	if !container.SessionService.SessionExists(s.Name) {
		return fmt.Errorf("tmux session '%s' is not running", s.Name)
	}

	content, err := container.ShellService.CapturePane(s.Name, s.Lines)
	if err != nil {
		return fmt.Errorf("failed to capture pane content: %w", err)
	}

	fmt.Print(content)
	return nil
}
