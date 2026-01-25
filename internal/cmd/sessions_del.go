package cmd

import (
	"context"
	"fmt"

	"rocha/internal/domain"
	"rocha/internal/logging"
	"rocha/internal/services"
)

// SessionsDelCmd deletes a session
type SessionsDelCmd struct {
	Force              bool   `help:"Force deletion without confirmation" short:"f"`
	Name               string `arg:"" help:"Name of the session to delete"`
	SkipKillTmux       bool   `help:"Skip killing tmux session" short:"k"`
	SkipRemoveWorktree bool   `help:"Skip removing associated git worktree" short:"w"`
}

// Run executes the del command
func (s *SessionsDelCmd) Run(container *Container, cli *CLI) error {
	killTmux := !s.SkipKillTmux
	removeWorktree := !s.SkipRemoveWorktree

	logging.Logger.Info("Executing sessions del command", "session", s.Name, "killTmux", killTmux, "removeWorktree", removeWorktree, "force", s.Force)

	ctx := context.Background()
	session, err := s.validateSession(ctx, container)
	if err != nil {
		return err
	}

	if !s.Force {
		if !s.confirmDeletion(session, killTmux, removeWorktree) {
			return nil
		}
	}

	return s.deleteSession(ctx, container, killTmux, removeWorktree)
}

func (s *SessionsDelCmd) validateSession(ctx context.Context, container *Container) (*domain.Session, error) {
	logging.Logger.Debug("Checking if session exists", "session", s.Name)
	session, err := container.SessionService.GetSession(ctx, s.Name)
	if err != nil {
		logging.Logger.Error("Session not found", "session", s.Name, "error", err)
		return nil, fmt.Errorf("session not found: %w", err)
	}
	logging.Logger.Debug("Session found", "session", s.Name, "worktreePath", session.WorktreePath)
	return session, nil
}

func (s *SessionsDelCmd) confirmDeletion(session *domain.Session, killTmux, removeWorktree bool) bool {
	logging.Logger.Debug("Prompting user for confirmation", "session", s.Name)
	fmt.Printf("WARNING: This will delete session '%s'\n", s.Name)
	if killTmux {
		fmt.Println("  - Kill tmux session")
	}
	if removeWorktree && session.WorktreePath != "" {
		fmt.Printf("  - Remove worktree at '%s'\n", session.WorktreePath)
	}
	fmt.Print("\nContinue? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		logging.Logger.Info("User cancelled session deletion", "session", s.Name)
		fmt.Println("Cancelled")
		return false
	}
	logging.Logger.Info("User confirmed session deletion", "session", s.Name)
	return true
}

func (s *SessionsDelCmd) deleteSession(ctx context.Context, container *Container, killTmux, removeWorktree bool) error {
	logging.Logger.Info("Deleting session", "session", s.Name)
	err := container.SessionService.DeleteSession(ctx, s.Name, services.DeleteSessionOptions{
		KillTmux:       killTmux,
		RemoveWorktree: removeWorktree,
	})
	if err != nil {
		logging.Logger.Error("Failed to delete session", "session", s.Name, "error", err)
		return fmt.Errorf("failed to delete session: %w", err)
	}

	logging.Logger.Info("Session deleted successfully via CLI", "session", s.Name)
	fmt.Printf("Session '%s' deleted successfully\n", s.Name)
	return nil
}
