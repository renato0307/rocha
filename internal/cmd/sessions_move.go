package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"rocha/internal/config"
	"rocha/internal/logging"
	"rocha/internal/services"
)

// SessionsMoveCmd moves sessions between ROCHA_HOME directories
type SessionsMoveCmd struct {
	Force bool   `help:"Skip confirmation prompt" short:"f"`
	From  string `help:"Source ROCHA_HOME path" required:"true"`
	Repo  string `help:"Repository identifier (owner/repo format)" short:"r" required:"true"`
	To    string `help:"Destination ROCHA_HOME path" required:"true"`
}

// Run executes the move command
func (s *SessionsMoveCmd) Run(container *Container, cli *CLI) error {
	logging.Logger.Info("Executing sessions move command", "repo", s.Repo, "from", s.From, "to", s.To, "force", s.Force)

	if err := s.validateRepoFormat(); err != nil {
		return err
	}

	sourceHome, destHome := s.expandPaths()

	if err := s.validateSourcePath(sourceHome); err != nil {
		return err
	}

	if err := s.createDestPath(destHome); err != nil {
		return err
	}

	ctx := context.Background()

	// Get session count for confirmation
	sourceSessions, err := container.MigrationService.GetSessionsForRepo(
		ctx,
		sourceHome,
		s.Repo,
	)
	if err != nil {
		logging.Logger.Error("Failed to get sessions for repo", "repo", s.Repo, "error", err)
		return fmt.Errorf("failed to get sessions for repository %s: %w", s.Repo, err)
	}

	if len(sourceSessions) == 0 {
		return fmt.Errorf("no sessions found for repository: %s", s.Repo)
	}

	if !s.Force {
		if !s.confirmMove(sourceHome, destHome, len(sourceSessions)) {
			return nil
		}
	}

	fmt.Printf("\nMoving repository: %s\n", s.Repo)
	logging.Logger.Info("Starting repository move", "repo", s.Repo)

	result, err := container.MigrationService.MoveRepositoryBetweenHomes(ctx, services.MoveRepositoryBetweenHomesParams{
		DestRochaHome:   destHome,
		RepoInfo:        s.Repo,
		SourceRochaHome: sourceHome,
	})
	if err != nil {
		logging.Logger.Error("Failed to move repository", "repo", s.Repo, "error", err)
		return fmt.Errorf("failed to move repository %s: %w", s.Repo, err)
	}

	fmt.Printf("Moved repository '%s' (%d session(s))\n", s.Repo, result.MovedSessionCount)
	logging.Logger.Info("Sessions move command completed successfully", "movedCount", result.MovedSessionCount, "repo", s.Repo)
	return nil
}

func (s *SessionsMoveCmd) validateRepoFormat() error {
	if !strings.Contains(s.Repo, "/") {
		return fmt.Errorf("invalid repo format '%s': must be in owner/repo format", s.Repo)
	}
	return nil
}

func (s *SessionsMoveCmd) expandPaths() (string, string) {
	sourceHome := config.ExpandPath(s.From)
	destHome := config.ExpandPath(s.To)
	logging.Logger.Debug("Paths expanded", "sourceHome", sourceHome, "destHome", destHome)
	return sourceHome, destHome
}

func (s *SessionsMoveCmd) validateSourcePath(sourceHome string) error {
	logging.Logger.Debug("Validating source path", "path", sourceHome)
	if _, err := os.Stat(sourceHome); os.IsNotExist(err) {
		logging.Logger.Error("Source ROCHA_HOME does not exist", "path", sourceHome)
		return fmt.Errorf("source ROCHA_HOME does not exist: %s", sourceHome)
	}
	return nil
}

func (s *SessionsMoveCmd) createDestPath(destHome string) error {
	logging.Logger.Debug("Creating destination directory if needed", "path", destHome)
	if err := os.MkdirAll(destHome, 0755); err != nil {
		logging.Logger.Error("Failed to create destination ROCHA_HOME", "path", destHome, "error", err)
		return fmt.Errorf("failed to create destination ROCHA_HOME: %w", err)
	}
	return nil
}

func (s *SessionsMoveCmd) confirmMove(sourceHome, destHome string, sessionCount int) bool {
	logging.Logger.Debug("Prompting user for confirmation", "repo", s.Repo)
	fmt.Println("WARNING: This operation will:")
	fmt.Println("  - Kill tmux sessions for all sessions in the specified repository")
	fmt.Println("  - Move .main directory and all worktrees to the new ROCHA_HOME location")
	fmt.Println("  - Repair git worktree references")
	fmt.Printf("  - Move sessions from %s to %s\n", sourceHome, destHome)
	fmt.Printf("\nRepository to move: %s (%d session(s))\n", s.Repo, sessionCount)
	fmt.Print("\nContinue? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		logging.Logger.Info("User cancelled session move", "repo", s.Repo)
		fmt.Println("Cancelled")
		return false
	}
	logging.Logger.Info("User confirmed session move", "repo", s.Repo)
	return true
}
