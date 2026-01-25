package cmd

import (
	"context"
	"fmt"

	"rocha/internal/logging"
)

// SessionsFlagCmd toggles the flag state of a session
type SessionsFlagCmd struct {
	Name string `arg:"" help:"Session name"`
}

// Run executes the flag command
func (s *SessionsFlagCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Executing sessions flag command", "name", s.Name)

	container, err := NewContainer(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	ctx := context.Background()

	// Get session to check current flag state
	session, err := container.SessionService.GetSession(ctx, s.Name)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	wasFlagged := session.IsFlagged

	if err := container.SessionService.ToggleFlag(ctx, s.Name); err != nil {
		return fmt.Errorf("failed to toggle flag: %w", err)
	}

	if wasFlagged {
		fmt.Printf("Session '%s' unflagged\n", s.Name)
	} else {
		fmt.Printf("Session '%s' flagged\n", s.Name)
	}
	return nil
}
