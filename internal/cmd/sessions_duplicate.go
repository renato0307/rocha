package cmd

import (
	"context"
	"fmt"

	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/services"
)

// SessionsDuplicateCmd creates a new session from an existing repository
type SessionsDuplicateCmd struct {
	Branch  string `help:"Branch for new session"`
	Name    string `arg:"" help:"Source session name"`
	NewName string `help:"New session name" required:"" name:"new-name"`
}

// Run executes the duplicate command
func (s *SessionsDuplicateCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Executing sessions duplicate command", "name", s.Name, "newName", s.NewName, "branch", s.Branch)

	ctx := context.Background()

	// Get source session
	sourceSession, err := cli.Container.SessionService.GetSession(ctx, s.Name)
	if err != nil {
		return fmt.Errorf("source session not found: %w", err)
	}

	// Validate source session has a repo source
	if sourceSession.RepoSource == "" {
		return fmt.Errorf("source session '%s' has no repository source", s.Name)
	}

	// Create new session from source repo
	params := services.CreateSessionParams{
		AllowDangerouslySkipPermissions: sourceSession.AllowDangerouslySkipPermissions,
		BranchNameOverride:              s.Branch,
		ClaudeDirOverride:               sourceSession.ClaudeDir,
		RepoSource:                      sourceSession.RepoSource,
		SessionName:                     s.NewName,
		TmuxStatusPosition:              cli.Container.SettingsService.GetTmuxStatusPosition(),
	}

	result, err := cli.Container.SessionService.CreateSession(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	fmt.Printf("Session '%s' created from '%s'\n", result.Session.Name, s.Name)
	if result.WorktreePath != "" {
		fmt.Printf("Worktree created at: %s\n", result.WorktreePath)
	}
	return nil
}
