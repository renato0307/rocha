package cmd

import (
	"context"
	"fmt"

	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/theme"
)

type SessionsOpenPRCmd struct {
	Name string `arg:"" help:"Session name"`
}

func (s *SessionsOpenPRCmd) Run(cli *CLI) error {
	ctx := context.Background()

	// Get session info
	session, err := cli.Container.SessionService.GetSession(ctx, s.Name)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Validate worktree/branch exists
	if session.WorktreePath == "" {
		return fmt.Errorf("session '%s' has no worktree", s.Name)
	}
	if session.BranchName == "" {
		return fmt.Errorf("session '%s' has no branch", s.Name)
	}

	logging.Logger.Debug("Opening PR for session", "name", s.Name, "path", session.WorktreePath)

	// Fetch PR info to get the number
	prInfo, err := cli.Container.GitService.FetchPRInfo(ctx, session.WorktreePath, session.BranchName)
	if err != nil {
		return fmt.Errorf("failed to get PR info: %w", err)
	}
	if prInfo == nil || prInfo.Number == 0 {
		return fmt.Errorf("no PR found for branch '%s'", session.BranchName)
	}

	// Open PR in browser
	if err := cli.Container.GitService.OpenPRInBrowser(session.WorktreePath); err != nil {
		return fmt.Errorf("failed to open PR: %w", err)
	}

	// Print styled output: "PR" in yellow, "#number" in default
	fmt.Printf("%s #%d\n", theme.PRLabelStyle.Render("PR"), prInfo.Number)

	return nil
}
