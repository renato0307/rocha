package cmd

import (
	"context"
	"fmt"

	"rocha/git"
	"rocha/paths"
	"rocha/storage"
)

// SessionsArchiveCmd archives or unarchives a session
type SessionsArchiveCmd struct {
	Force              bool   `help:"Skip confirmation prompt" short:"f"`
	Name               string `arg:"" help:"Name of the session to archive/unarchive"`
	RemoveWorktree     bool   `help:"Remove associated git worktree" short:"w"`
	SkipWorktreePrompt bool   `help:"Don't prompt about worktree removal" short:"s"`
}

// Run executes the archive command
func (s *SessionsArchiveCmd) Run(cli *CLI) error {
	store, err := storage.NewStore(paths.GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	session, err := store.GetSession(context.Background(), s.Name)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	isArchiving := !session.IsArchived

	if isArchiving {
		return s.archiveSession(store, session)
	}
	return s.unarchiveSession(store)
}

func (s *SessionsArchiveCmd) archiveSession(store *storage.Store, session *storage.SessionInfo) error {
	if !s.Force {
		fmt.Printf("Are you sure you want to archive session '%s'? (y/N): ", s.Name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	removeWorktree := s.RemoveWorktree
	if session.WorktreePath != "" && !s.SkipWorktreePrompt && !s.RemoveWorktree {
		fmt.Printf("Remove associated worktree at '%s'? (y/N): ", session.WorktreePath)
		var response string
		fmt.Scanln(&response)
		removeWorktree = (response == "y" || response == "Y")
	}

	if removeWorktree && session.WorktreePath != "" {
		if err := git.RemoveWorktree(session.RepoPath, session.WorktreePath); err != nil {
			fmt.Printf("Warning: Failed to remove worktree: %v\n", err)
			fmt.Println("Continuing with archiving...")
		} else {
			fmt.Printf("Removed worktree at '%s'\n", session.WorktreePath)
		}
	}

	if err := store.ToggleArchive(context.Background(), s.Name); err != nil {
		return fmt.Errorf("failed to archive session: %w", err)
	}

	fmt.Printf("Session '%s' archived successfully\n", s.Name)
	return nil
}

func (s *SessionsArchiveCmd) unarchiveSession(store *storage.Store) error {
	if err := store.ToggleArchive(context.Background(), s.Name); err != nil {
		return fmt.Errorf("failed to unarchive session: %w", err)
	}

	fmt.Printf("Session '%s' unarchived successfully\n", s.Name)
	return nil
}
