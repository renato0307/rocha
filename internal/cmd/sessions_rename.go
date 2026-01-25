package cmd

import (
	"context"
	"fmt"

	"github.com/renato0307/rocha/internal/logging"
)

// SessionsRenameCmd updates the display name of a session
type SessionsRenameCmd struct {
	DisplayName string `help:"New display name" required:"" name:"display-name"`
	Name        string `arg:"" help:"Session name"`
}

// Run executes the rename command
func (s *SessionsRenameCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Executing sessions rename command", "name", s.Name, "displayName", s.DisplayName)

	ctx := context.Background()

	// Validate session exists
	if _, err := cli.Container.SessionService.GetSession(ctx, s.Name); err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if err := cli.Container.SessionService.UpdateDisplayName(ctx, s.Name, s.DisplayName); err != nil {
		return fmt.Errorf("failed to update display name: %w", err)
	}

	fmt.Printf("Session '%s' display name updated to '%s'\n", s.Name, s.DisplayName)
	return nil
}
