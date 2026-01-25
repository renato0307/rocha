package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"rocha/internal/application"
	"rocha/internal/config"
	"rocha/internal/domain"
	"rocha/internal/logging"
	"rocha/internal/ports"
)

// SessionsMoveCmd moves sessions between ROCHA_HOME directories
type SessionsMoveCmd struct {
	Force bool   `help:"Skip confirmation prompt" short:"f"`
	From  string `help:"Source ROCHA_HOME path" required:"true"`
	Repo  string `help:"Repository identifier (owner/repo format)" short:"r" required:"true"`
	To    string `help:"Destination ROCHA_HOME path" required:"true"`
}

// Run executes the move command
func (s *SessionsMoveCmd) Run(tmuxClient ports.TmuxClient, cli *CLI) error {
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

	sourceRepo, destRepo, err := s.openRepositories(sourceHome, destHome)
	if err != nil {
		return err
	}
	defer sourceRepo.Close()
	defer destRepo.Close()

	sourceSessions, err := s.getSourceSessions(sourceRepo)
	if err != nil {
		return err
	}

	if !s.Force {
		if !s.confirmMove(sourceHome, destHome, len(sourceSessions)) {
			return nil
		}
	}

	return s.moveRepository(tmuxClient, sourceSessions, sourceRepo, destRepo, sourceHome, destHome)
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

func (s *SessionsMoveCmd) openRepositories(sourceHome, destHome string) (ports.SessionRepository, ports.SessionRepository, error) {
	logging.Logger.Debug("Opening repositories", "sourceHome", sourceHome, "destHome", destHome)

	sourceRepo, err := NewSessionRepositoryForPath(sourceHome)
	if err != nil {
		logging.Logger.Error("Failed to open source repository", "path", sourceHome, "error", err)
		return nil, nil, fmt.Errorf("failed to open source database: %w", err)
	}

	destRepo, err := NewSessionRepositoryForPath(destHome)
	if err != nil {
		sourceRepo.Close()
		logging.Logger.Error("Failed to open destination repository", "path", destHome, "error", err)
		return nil, nil, fmt.Errorf("failed to open destination database: %w", err)
	}

	return sourceRepo, destRepo, nil
}

func (s *SessionsMoveCmd) getSourceSessions(sourceRepo ports.SessionReader) ([]domain.Session, error) {
	ctx := context.Background()
	sessions, err := sourceRepo.List(ctx, false)
	if err != nil {
		logging.Logger.Error("Failed to list sessions", "error", err)
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var repoSessions []domain.Session
	for _, sess := range sessions {
		if sess.RepoInfo == s.Repo {
			repoSessions = append(repoSessions, sess)
		}
	}
	return repoSessions, nil
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

func (s *SessionsMoveCmd) moveRepository(
	tmuxClient ports.TmuxClient,
	sourceSessions []domain.Session,
	sourceRepo ports.SessionRepository,
	destRepo ports.SessionRepository,
	sourceHome, destHome string,
) error {
	ctx := context.Background()

	// Create a temporary container for the migration service
	container, err := NewContainer(tmuxClient)
	if err != nil {
		logging.Logger.Error("Failed to create container", "error", err)
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	fmt.Printf("\nMoving repository: %s\n", s.Repo)
	logging.Logger.Info("Starting repository move", "repo", s.Repo)

	movedSessions, err := container.MigrationService.MoveRepository(ctx, application.MoveRepositoryParams{
		DestRochaHome:   destHome,
		DestSessionRepo: destRepo,
		RepoInfo:        s.Repo,
		SourceRochaHome: sourceHome,
		SourceSessions:  sourceSessions,
	})
	if err != nil {
		logging.Logger.Error("Failed to move repository", "repo", s.Repo, "error", err)
		return fmt.Errorf("failed to move repository %s: %w", s.Repo, err)
	}

	// Delete sessions from source store after successful move
	fmt.Printf("Cleaning up source database...\n")
	for _, sessName := range movedSessions {
		logging.Logger.Debug("Deleting session from source", "session", sessName)
		if err := sourceRepo.Delete(ctx, sessName); err != nil {
			logging.Logger.Warn("Failed to delete session from source", "session", sessName, "error", err)
			fmt.Printf("⚠ Warning: Failed to delete session %s from source: %v\n", sessName, err)
		}
	}

	fmt.Printf("Moved repository '%s' (%d session(s))\n", s.Repo, len(movedSessions))
	logging.Logger.Info("Sessions move command completed successfully", "movedCount", len(movedSessions), "repo", s.Repo)
	return nil
}
