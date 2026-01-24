package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rocha/logging"
	"rocha/operations"
	"rocha/paths"
	"rocha/storage"
	"rocha/tmux"
)

// SessionsMoveCmd moves sessions between ROCHA_HOME directories
type SessionsMoveCmd struct {
	Force bool   `help:"Skip confirmation prompt" short:"f"`
	From  string `help:"Source ROCHA_HOME path" required:"true"`
	Repo  string `help:"Repository identifier (owner/repo format)" short:"r" required:"true"`
	To    string `help:"Destination ROCHA_HOME path" required:"true"`
}

// Run executes the move command
func (s *SessionsMoveCmd) Run(cli *CLI) error {
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

	sourceStore, destStore, err := s.openStores(sourceHome, destHome)
	if err != nil {
		return err
	}
	defer sourceStore.Close()
	defer destStore.Close()

	sessionCount, err := s.countSessionsToMove(sourceStore)
	if err != nil {
		return err
	}

	if !s.Force {
		if !s.confirmMove(sourceHome, destHome, sessionCount) {
			return nil
		}
	}

	return s.moveRepository(sourceStore, destStore, sourceHome, destHome)
}

func (s *SessionsMoveCmd) validateRepoFormat() error {
	if !strings.Contains(s.Repo, "/") {
		return fmt.Errorf("invalid repo format '%s': must be in owner/repo format", s.Repo)
	}
	return nil
}

func (s *SessionsMoveCmd) expandPaths() (string, string) {
	sourceHome := paths.ExpandPath(s.From)
	destHome := paths.ExpandPath(s.To)
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

func (s *SessionsMoveCmd) openStores(sourceHome, destHome string) (*storage.Store, *storage.Store, error) {
	sourceDBPath := filepath.Join(sourceHome, "state.db")
	destDBPath := filepath.Join(destHome, "state.db")
	logging.Logger.Debug("Opening databases", "source", sourceDBPath, "dest", destDBPath)

	sourceStore, err := storage.NewStore(sourceDBPath)
	if err != nil {
		logging.Logger.Error("Failed to open source database", "path", sourceDBPath, "error", err)
		return nil, nil, fmt.Errorf("failed to open source database: %w", err)
	}

	destStore, err := storage.NewStore(destDBPath)
	if err != nil {
		sourceStore.Close()
		logging.Logger.Error("Failed to open destination database", "path", destDBPath, "error", err)
		return nil, nil, fmt.Errorf("failed to open destination database: %w", err)
	}

	return sourceStore, destStore, nil
}

func (s *SessionsMoveCmd) countSessionsToMove(sourceStore *storage.Store) (int, error) {
	sessions, err := sourceStore.ListSessions(context.Background(), false)
	if err != nil {
		logging.Logger.Error("Failed to list sessions", "error", err)
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	count := 0
	for _, sess := range sessions {
		if sess.RepoInfo == s.Repo {
			count++
		}
	}
	return count, nil
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

func (s *SessionsMoveCmd) moveRepository(sourceStore, destStore *storage.Store, sourceHome, destHome string) error {
	ctx := context.Background()
	tmuxClient := tmux.NewClient()

	fmt.Printf("\nMoving repository: %s\n", s.Repo)
	logging.Logger.Info("Starting repository move", "repo", s.Repo)

	movedSessions, err := operations.MoveRepository(ctx, s.Repo, sourceStore, destStore, sourceHome, destHome, tmuxClient)
	if err != nil {
		logging.Logger.Error("Failed to move repository", "repo", s.Repo, "error", err)
		return fmt.Errorf("failed to move repository %s: %w", s.Repo, err)
	}

	fmt.Printf("Moved repository '%s' (%d session(s))\n", s.Repo, len(movedSessions))
	logging.Logger.Info("Sessions move command completed successfully", "movedCount", len(movedSessions), "repo", s.Repo)
	return nil
}
