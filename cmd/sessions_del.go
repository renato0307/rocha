package cmd

import (
	"context"
	"fmt"

	"rocha/logging"
	"rocha/operations"
	"rocha/paths"
	"rocha/storage"
	"rocha/tmux"
)

// SessionsDelCmd deletes a session
type SessionsDelCmd struct {
	Force              bool   `help:"Force deletion without confirmation" short:"f"`
	Name               string `arg:"" help:"Name of the session to delete"`
	SkipKillTmux       bool   `help:"Skip killing tmux session" short:"k"`
	SkipRemoveWorktree bool   `help:"Skip removing associated git worktree" short:"w"`
}

// Run executes the del command
func (s *SessionsDelCmd) Run(cli *CLI) error {
	killTmux := !s.SkipKillTmux
	removeWorktree := !s.SkipRemoveWorktree

	logging.Logger.Info("Executing sessions del command", "session", s.Name, "killTmux", killTmux, "removeWorktree", removeWorktree, "force", s.Force)

	ctx := context.Background()
	store, err := storage.NewStore(paths.GetDBPath())
	if err != nil {
		logging.Logger.Error("Failed to open database", "error", err)
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	sessInfo, err := s.validateSession(ctx, store)
	if err != nil {
		return err
	}

	if !s.Force {
		if !s.confirmDeletion(sessInfo, killTmux, removeWorktree) {
			return nil
		}
	}

	return s.deleteSession(ctx, store, killTmux, removeWorktree)
}

func (s *SessionsDelCmd) validateSession(ctx context.Context, store *storage.Store) (*storage.SessionInfo, error) {
	logging.Logger.Debug("Checking if session exists", "session", s.Name)
	sessInfo, err := store.GetSession(ctx, s.Name)
	if err != nil {
		logging.Logger.Error("Session not found", "session", s.Name, "error", err)
		return nil, fmt.Errorf("session not found: %w", err)
	}
	logging.Logger.Debug("Session found", "session", s.Name, "worktreePath", sessInfo.WorktreePath)
	return sessInfo, nil
}

func (s *SessionsDelCmd) confirmDeletion(sessInfo *storage.SessionInfo, killTmux, removeWorktree bool) bool {
	logging.Logger.Debug("Prompting user for confirmation", "session", s.Name)
	fmt.Printf("WARNING: This will delete session '%s'\n", s.Name)
	if killTmux {
		fmt.Println("  - Kill tmux session")
	}
	if removeWorktree && sessInfo.WorktreePath != "" {
		fmt.Printf("  - Remove worktree at '%s'\n", sessInfo.WorktreePath)
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

func (s *SessionsDelCmd) deleteSession(ctx context.Context, store *storage.Store, killTmux, removeWorktree bool) error {
	tmuxClient := tmux.NewClient()

	logging.Logger.Info("Deleting session", "session", s.Name)
	err := operations.DeleteSession(ctx, s.Name, store, operations.DeleteSessionOptions{
		KillTmux:       killTmux,
		RemoveWorktree: removeWorktree,
	}, tmuxClient)
	if err != nil {
		logging.Logger.Error("Failed to delete session", "session", s.Name, "error", err)
		return fmt.Errorf("failed to delete session: %w", err)
	}

	logging.Logger.Info("Session deleted successfully via CLI", "session", s.Name)
	fmt.Printf("Session '%s' deleted successfully\n", s.Name)
	return nil
}
