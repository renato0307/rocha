package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/renato0307/rocha/internal/domain"
)

// SessionsAddCmd adds a new session
type SessionsAddCmd struct {
	AllowDangerouslySkipPermissions bool   `help:"Skip permission prompts in Claude (DANGEROUS)"`
	BranchName                      string `help:"Branch name" default:""`
	DisplayName                     string `help:"Display name for the session" default:""`
	Name                            string `arg:"" help:"Name of the session to add"`
	RepoInfo                        string `help:"Repository info" default:""`
	RepoPath                        string `help:"Repository path" default:""`
	State                           string `help:"Initial state" enum:"idle,working,waiting,exited" default:"idle"`
	WorktreePath                    string `help:"Worktree path" default:""`
}

// Run executes the add command
func (s *SessionsAddCmd) Run(cli *CLI) error {
	displayName := s.DisplayName
	if displayName == "" {
		displayName = s.Name
	}

	session := domain.Session{
		AllowDangerouslySkipPermissions: s.AllowDangerouslySkipPermissions,
		BranchName:                      s.BranchName,
		DisplayName:                     displayName,
		ExecutionID:                     uuid.New().String(),
		LastUpdated:                     time.Now().UTC(),
		Name:                            s.Name,
		RepoInfo:                        s.RepoInfo,
		RepoPath:                        s.RepoPath,
		State:                           domain.SessionState(s.State),
		WorktreePath:                    s.WorktreePath,
	}

	if err := cli.Container.SessionService.AddSession(context.Background(), session); err != nil {
		return fmt.Errorf("failed to add session: %w", err)
	}

	fmt.Printf("Session '%s' added successfully\n", s.Name)
	return nil
}
