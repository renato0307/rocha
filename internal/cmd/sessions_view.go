package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"rocha/internal/domain"
	"rocha/internal/ports"
)

// SessionsViewCmd views a specific session
type SessionsViewCmd struct {
	Format string `help:"Output format: table or json" enum:"table,json" default:"table"`
	Name   string `arg:"" help:"Name of the session to view"`
}

// Run executes the view command
func (s *SessionsViewCmd) Run(tmuxClient ports.TmuxClient, cli *CLI) error {
	container, err := NewContainer(tmuxClient)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	session, err := container.SessionRepository.Get(context.Background(), s.Name)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if s.Format == "json" {
		return s.printJSON(session)
	}
	return s.printTable(session)
}

func (s *SessionsViewCmd) printJSON(session *domain.Session) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func (s *SessionsViewCmd) printTable(session *domain.Session) error {
	fmt.Printf("Session: %s\n", session.Name)
	fmt.Printf("Display Name: %s\n", session.DisplayName)
	fmt.Printf("State: %s\n", session.State)
	fmt.Printf("Execution ID: %s\n", session.ExecutionID)
	fmt.Printf("Archived: %t\n", session.IsArchived)
	fmt.Printf("Flagged: %t\n", session.IsFlagged)
	fmt.Printf("Last Updated: %s\n", session.LastUpdated.Format("2006-01-02 15:04:05"))
	fmt.Printf("Repo Path: %s\n", session.RepoPath)
	fmt.Printf("Repo Info: %s\n", session.RepoInfo)
	fmt.Printf("Branch Name: %s\n", session.BranchName)
	fmt.Printf("Worktree Path: %s\n", session.WorktreePath)
	if session.ClaudeDir != "" {
		fmt.Printf("Claude Dir: %s\n", session.ClaudeDir)
	} else {
		fmt.Printf("Claude Dir: <default>\n")
	}
	fmt.Printf("Allow Dangerously Skip Permissions: %t\n", session.AllowDangerouslySkipPermissions)

	if session.ShellSession != nil {
		fmt.Printf("\nShell Session:\n")
		fmt.Printf("  Name: %s\n", session.ShellSession.Name)
		fmt.Printf("  Display Name: %s\n", session.ShellSession.DisplayName)
		fmt.Printf("  State: %s\n", session.ShellSession.State)
		fmt.Printf("  Last Updated: %s\n", session.ShellSession.LastUpdated.Format("2006-01-02 15:04:05"))
	}

	return nil
}
