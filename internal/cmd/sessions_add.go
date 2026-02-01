package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/services"
)

// SessionsAddCmd adds a new session
type SessionsAddCmd struct {
	AllowDangerouslySkipPermissions bool   `help:"Skip permission prompts in Claude (DANGEROUS)"`
	BranchName                      string `help:"Branch name" default:""`
	DebugClaude                     bool   `help:"Enable debug logging in Claude Code"`
	DisplayName                     string `help:"Display name for the session" default:""`
	InitialPrompt                   string `help:"Initial prompt to send to Claude on session start" name:"prompt" short:"p" default:""`
	Name                            string `arg:"" help:"Name of the session to add"`
	RepoInfo                        string `help:"Repository info" default:""`
	RepoPath                        string `help:"Repository path" default:""`
	RepoSource                      string `help:"Repository source URL (creates worktree)" default:""`
	StartClaude                     bool   `help:"Create tmux session and start Claude" name:"start-claude"`
	State                           string `help:"Initial state" enum:"idle,working,waiting,exited" default:"idle"`
	WorktreePath                    string `help:"Worktree path" default:""`
}

// Run executes the add command
func (s *SessionsAddCmd) Run(cli *CLI) error {
	ctx := context.Background()

	// If --start-claude is provided, use SessionService.CreateSession()
	// which creates the tmux session and starts Claude with the prompt
	if s.StartClaude {
		return s.runWithStartClaude(ctx, cli)
	}

	// Otherwise, just add metadata to the database (existing behavior)
	return s.runMetadataOnly(ctx, cli)
}

// runWithStartClaude creates a tmux session and starts Claude
func (s *SessionsAddCmd) runWithStartClaude(ctx context.Context, cli *CLI) error {
	logging.Logger.Info("Creating session with tmux and Claude",
		"name", s.Name,
		"has_prompt", s.InitialPrompt != "",
		"repo_source", s.RepoSource)

	params := services.CreateSessionParams{
		AllowDangerouslySkipPermissions: s.AllowDangerouslySkipPermissions,
		BranchNameOverride:              s.BranchName,
		DebugClaude:                     s.DebugClaude,
		InitialPrompt:                   s.InitialPrompt,
		RepoSource:                      s.RepoSource,
		SessionName:                     s.Name,
		TmuxStatusPosition:              cli.Container.SettingsService.GetTmuxStatusPosition(),
	}

	result, err := cli.Container.SessionService.CreateSession(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	fmt.Printf("Session '%s' created successfully\n", result.Session.Name)
	if result.WorktreePath != "" {
		fmt.Printf("Worktree: %s\n", result.WorktreePath)
	}
	if s.InitialPrompt != "" {
		fmt.Printf("Initial prompt sent to Claude\n")
	}
	return nil
}

// runMetadataOnly adds session metadata to the database without creating tmux session
func (s *SessionsAddCmd) runMetadataOnly(ctx context.Context, cli *CLI) error {
	displayName := s.DisplayName
	if displayName == "" {
		displayName = s.Name
	}

	session := domain.Session{
		AllowDangerouslySkipPermissions: s.AllowDangerouslySkipPermissions,
		BranchName:                      s.BranchName,
		DebugClaude:                     s.DebugClaude,
		DisplayName:                     displayName,
		ExecutionID:                     uuid.New().String(),
		InitialPrompt:                   s.InitialPrompt,
		LastUpdated:                     time.Now().UTC(),
		Name:                            s.Name,
		RepoInfo:                        s.RepoInfo,
		RepoPath:                        s.RepoPath,
		RepoSource:                      s.RepoSource,
		State:                           domain.SessionState(s.State),
		WorktreePath:                    s.WorktreePath,
	}

	if err := cli.Container.SessionService.AddSession(ctx, session); err != nil {
		return fmt.Errorf("failed to add session: %w", err)
	}

	fmt.Printf("Session '%s' added successfully\n", s.Name)
	if s.InitialPrompt != "" {
		fmt.Printf("Initial prompt stored (will be sent when session starts via UI)\n")
	}
	return nil
}
