package cmd

import (
	"context"
	"fmt"

	"rocha/internal/domain"
	"rocha/internal/ports"
)

// SessionsArchiveCmd archives or unarchives a session
type SessionsArchiveCmd struct {
	Force              bool   `help:"Skip confirmation prompt" short:"f"`
	Name               string `arg:"" help:"Name of the session to archive/unarchive"`
	RemoveWorktree     bool   `help:"Remove associated git worktree" short:"w"`
	SkipWorktreePrompt bool   `help:"Don't prompt about worktree removal" short:"s"`
}

// Run executes the archive command
func (s *SessionsArchiveCmd) Run(tmuxClient ports.TmuxClient, cli *CLI) error {
	container, err := NewContainer(tmuxClient)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	session, err := container.SessionRepository.Get(context.Background(), s.Name)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	isArchiving := !session.IsArchived

	if isArchiving {
		return s.archiveSession(container, session)
	}
	return s.unarchiveSession(container)
}

func (s *SessionsArchiveCmd) archiveSession(container *Container, session *domain.Session) error {
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

	ctx := context.Background()
	if err := container.SessionService.ArchiveSession(ctx, s.Name, removeWorktree); err != nil {
		return fmt.Errorf("failed to archive session: %w", err)
	}

	if removeWorktree && session.WorktreePath != "" {
		fmt.Printf("Removed worktree at '%s'\n", session.WorktreePath)
	}

	fmt.Printf("Session '%s' archived successfully\n", s.Name)
	return nil
}

func (s *SessionsArchiveCmd) unarchiveSession(container *Container) error {
	if err := container.SessionRepository.ToggleArchive(context.Background(), s.Name); err != nil {
		return fmt.Errorf("failed to unarchive session: %w", err)
	}

	fmt.Printf("Session '%s' unarchived successfully\n", s.Name)
	return nil
}
