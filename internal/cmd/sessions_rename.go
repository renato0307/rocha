package cmd

import (
	"context"
	"fmt"

	"rocha/internal/logging"
)

// SessionsRenameCmd updates the display name of a session
type SessionsRenameCmd struct {
	DisplayName string `help:"New display name" required:"" name:"display-name"`
	Name        string `arg:"" help:"Session name"`
}

// Run executes the rename command
func (s *SessionsRenameCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Executing sessions rename command", "name", s.Name, "displayName", s.DisplayName)

	container, err := NewContainer(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	ctx := context.Background()

	// Validate session exists
	if _, err := container.SessionService.GetSession(ctx, s.Name); err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if err := container.SessionService.UpdateDisplayName(ctx, s.Name, s.DisplayName); err != nil {
		return fmt.Errorf("failed to update display name: %w", err)
	}

	fmt.Printf("Session '%s' display name updated to '%s'\n", s.Name, s.DisplayName)
	return nil
}
