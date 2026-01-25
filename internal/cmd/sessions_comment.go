package cmd

import (
	"context"
	"fmt"

	"github.com/renato0307/rocha/internal/logging"
)

// SessionsCommentCmd adds, edits, or clears a session comment
type SessionsCommentCmd struct {
	Comment string `help:"Comment text (empty clears)" required:""`
	Name    string `arg:"" help:"Session name"`
}

// Run executes the comment command
func (s *SessionsCommentCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Executing sessions comment command", "name", s.Name, "comment", s.Comment)

	ctx := context.Background()

	// Validate session exists
	if _, err := cli.Container.SessionService.GetSession(ctx, s.Name); err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if err := cli.Container.SessionService.UpdateComment(ctx, s.Name, s.Comment); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	if s.Comment == "" {
		fmt.Printf("Comment cleared for session '%s'\n", s.Name)
	} else {
		fmt.Printf("Comment updated for session '%s'\n", s.Name)
	}
	return nil
}
