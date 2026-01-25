package cmd

import (
	"context"
	"fmt"

	"github.com/renato0307/rocha/internal/logging"
)

// SessionsFlagCmd toggles the flag state of a session
type SessionsFlagCmd struct {
	Name string `arg:"" help:"Session name"`
}

// Run executes the flag command
func (s *SessionsFlagCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Executing sessions flag command", "name", s.Name)

	ctx := context.Background()

	// Get session to check current flag state
	session, err := cli.Container.SessionService.GetSession(ctx, s.Name)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	wasFlagged := session.IsFlagged

	if err := cli.Container.SessionService.ToggleFlag(ctx, s.Name); err != nil {
		return fmt.Errorf("failed to toggle flag: %w", err)
	}

	if wasFlagged {
		fmt.Printf("Session '%s' unflagged\n", s.Name)
	} else {
		fmt.Printf("Session '%s' flagged\n", s.Name)
	}
	return nil
}
