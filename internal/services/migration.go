package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// SessionRepositoryFactory creates session repositories for a given path
type SessionRepositoryFactory func(rochaHomePath string) (ports.SessionRepository, error)

// MigrationService handles session migration between ROCHA_HOME directories
type MigrationService struct {
	gitRepo    ports.GitRepository
	repoFactory SessionRepositoryFactory
	tmuxClient ports.TmuxSessionLifecycle
}

// NewMigrationService creates a new MigrationService
func NewMigrationService(
	gitRepo ports.GitRepository,
	tmuxClient ports.TmuxSessionLifecycle,
	repoFactory SessionRepositoryFactory,
) *MigrationService {
	return &MigrationService{
		gitRepo:     gitRepo,
		repoFactory: repoFactory,
		tmuxClient:  tmuxClient,
	}
}

// MoveRepositoryBetweenHomesParams contains parameters for moving a repository between ROCHA_HOME directories
type MoveRepositoryBetweenHomesParams struct {
	DestRochaHome   string
	RepoInfo        string
	SourceRochaHome string
}

// MoveRepositoryBetweenHomesResult contains the result of a repository move operation
type MoveRepositoryBetweenHomesResult struct {
	MovedSessionCount int
	SourceSessions    []domain.Session
}

// MoveRepositoryBetweenHomes moves all sessions for a repository from one ROCHA_HOME to another
// This is the high-level API that manages repositories internally
func (s *MigrationService) MoveRepositoryBetweenHomes(
	ctx context.Context,
	params MoveRepositoryBetweenHomesParams,
) (*MoveRepositoryBetweenHomesResult, error) {
	logging.Logger.Info("Moving repository between ROCHA_HOME directories",
		"repo", params.RepoInfo,
		"from", params.SourceRochaHome,
		"to", params.DestRochaHome)

	// Open source repository
	sourceRepo, err := s.repoFactory(params.SourceRochaHome)
	if err != nil {
		logging.Logger.Error("Failed to open source repository", "path", params.SourceRochaHome, "error", err)
		return nil, fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceRepo.Close()

	// List sessions and filter by repo
	sessions, err := sourceRepo.List(ctx, false)
	if err != nil {
		logging.Logger.Error("Failed to list sessions", "error", err)
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var sourceSessions []domain.Session
	for _, sess := range sessions {
		if sess.RepoInfo == params.RepoInfo {
			sourceSessions = append(sourceSessions, sess)
		}
	}

	if len(sourceSessions) == 0 {
		logging.Logger.Error("No sessions found for repository", "repo", params.RepoInfo)
		return nil, fmt.Errorf("no sessions found for repository: %s", params.RepoInfo)
	}

	// Open destination repository
	destRepo, err := s.repoFactory(params.DestRochaHome)
	if err != nil {
		logging.Logger.Error("Failed to open destination repository", "path", params.DestRochaHome, "error", err)
		return nil, fmt.Errorf("failed to open destination database: %w", err)
	}
	defer destRepo.Close()

	// Move repository using internal method
	movedNames, err := s.moveRepository(ctx, moveRepositoryInternalParams{
		DestRochaHome:   params.DestRochaHome,
		DestSessionRepo: destRepo,
		RepoInfo:        params.RepoInfo,
		SourceRochaHome: params.SourceRochaHome,
		SourceSessions:  sourceSessions,
	})
	if err != nil {
		return nil, err
	}

	// Delete sessions from source
	fmt.Printf("Cleaning up source database...\n")
	for _, sessName := range movedNames {
		logging.Logger.Debug("Deleting session from source", "session", sessName)
		if err := sourceRepo.Delete(ctx, sessName); err != nil {
			logging.Logger.Warn("Failed to delete session from source", "session", sessName, "error", err)
			fmt.Printf("⚠ Warning: Failed to delete session %s from source: %v\n", sessName, err)
		}
	}

	return &MoveRepositoryBetweenHomesResult{
		MovedSessionCount: len(movedNames),
		SourceSessions:    sourceSessions,
	}, nil
}

// GetSessionsForRepo returns sessions for a specific repository from a ROCHA_HOME directory
func (s *MigrationService) GetSessionsForRepo(
	ctx context.Context,
	rochaHome string,
	repoInfo string,
) ([]domain.Session, error) {
	repo, err := s.repoFactory(rochaHome)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}
	defer repo.Close()

	sessions, err := repo.List(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var result []domain.Session
	for _, sess := range sessions {
		if sess.RepoInfo == repoInfo {
			result = append(result, sess)
		}
	}
	return result, nil
}

// moveRepositoryInternalParams contains parameters for the internal move operation
type moveRepositoryInternalParams struct {
	DestRochaHome   string
	DestSessionRepo ports.SessionRepository
	RepoInfo        string
	SourceRochaHome string
	SourceSessions  []domain.Session
}

// MoveRepositoryParams contains parameters for moving a repository
// Deprecated: Use MoveRepositoryBetweenHomes instead for cleaner API
type MoveRepositoryParams struct {
	DestRochaHome   string
	DestSessionRepo ports.SessionRepository
	RepoInfo        string
	SourceRochaHome string
	SourceSessions  []domain.Session
}

// MoveRepository moves all sessions from a single repository
// Deprecated: Use MoveRepositoryBetweenHomes instead for cleaner API
// Returns: list of moved session names, error
func (s *MigrationService) MoveRepository(
	ctx context.Context,
	params MoveRepositoryParams,
) ([]string, error) {
	return s.moveRepository(ctx, moveRepositoryInternalParams{
		DestRochaHome:   params.DestRochaHome,
		DestSessionRepo: params.DestSessionRepo,
		RepoInfo:        params.RepoInfo,
		SourceRochaHome: params.SourceRochaHome,
		SourceSessions:  params.SourceSessions,
	})
}

// moveRepository is the internal implementation for moving a repository
func (s *MigrationService) moveRepository(
	ctx context.Context,
	params moveRepositoryInternalParams,
) ([]string, error) {
	logging.Logger.Info("Moving repository",
		"repo", params.RepoInfo,
		"from", params.SourceRochaHome,
		"to", params.DestRochaHome)

	repoSessions := params.SourceSessions
	if len(repoSessions) == 0 {
		logging.Logger.Error("No sessions found for repository", "repo", params.RepoInfo)
		return nil, fmt.Errorf("no sessions found for repository: %s", params.RepoInfo)
	}

	logging.Logger.Info("Found sessions for repository", "repo", params.RepoInfo, "count", len(repoSessions))

	// Extract shared main repository path from first session
	mainRepoPath := repoSessions[0].RepoPath
	if mainRepoPath == "" {
		logging.Logger.Warn("No RepoPath found for repository sessions", "repo", params.RepoInfo)
		fmt.Printf("⚠ Warning: No %s directory found for repository %s\n", config.MainRepoDir, params.RepoInfo)
	}

	// Validate all sessions share the same main repository path
	for _, sess := range repoSessions {
		if sess.RepoPath != mainRepoPath {
			logging.Logger.Error("Sessions have different RepoPath values",
				"repo", params.RepoInfo,
				"expected", mainRepoPath,
				"found", sess.RepoPath)
			return nil, fmt.Errorf("sessions in repository %s have different main repository paths", params.RepoInfo)
		}
	}

	// Kill all tmux sessions first
	logging.Logger.Debug("Killing tmux sessions for repository", "repo", params.RepoInfo)
	for _, sess := range repoSessions {
		fmt.Printf("Killing tmux session '%s'...\n", sess.Name)
		if err := s.tmuxClient.KillSession(sess.Name); err != nil {
			logging.Logger.Warn("Failed to kill tmux session", "session", sess.Name, "error", err)
			fmt.Printf("⚠ Warning: Failed to kill tmux session %s: %v\n", sess.Name, err)
		}

		// Kill shell session if exists
		if sess.ShellSession != nil {
			shellName := sess.ShellSession.Name
			logging.Logger.Debug("Killing shell session", "session", shellName)
			if err := s.tmuxClient.KillSession(shellName); err != nil {
				logging.Logger.Warn("Failed to kill shell session", "session", shellName, "error", err)
				fmt.Printf("⚠ Warning: Failed to kill shell session %s: %v\n", shellName, err)
			}
		}
	}

	// Move main repository directory if it exists
	if mainRepoPath != "" {
		sourceMainPath := mainRepoPath
		destMainPath := strings.Replace(mainRepoPath, params.SourceRochaHome, params.DestRochaHome, 1)

		logging.Logger.Info("Moving main repository directory", "from", sourceMainPath, "to", destMainPath)
		fmt.Printf("Moving main repository directory...\n")

		if err := s.moveMainDirectory(sourceMainPath, destMainPath); err != nil {
			logging.Logger.Error("Failed to move main repository directory", "error", err)
			return nil, fmt.Errorf("failed to move main repository directory: %w", err)
		}
		fmt.Printf("✓ Moved main repository directory\n")

		// Update mainRepoPath to point to new location
		mainRepoPath = destMainPath
	}

	// Move all session worktrees and collect paths for repair
	var movedWorktreePaths []string
	for i := range repoSessions {
		// Update session paths
		s.updateSessionPaths(&repoSessions[i], params.SourceRochaHome, params.DestRochaHome)

		// Move worktree if exists
		if repoSessions[i].WorktreePath != "" {
			sourceWorktree := strings.Replace(repoSessions[i].WorktreePath, params.DestRochaHome, params.SourceRochaHome, 1)
			destWorktree := repoSessions[i].WorktreePath

			fmt.Printf("Moving worktree '%s'...\n", repoSessions[i].Name)
			logging.Logger.Info("Moving worktree", "session", repoSessions[i].Name, "from", sourceWorktree, "to", destWorktree)

			if err := s.moveWorktree(sourceWorktree, destWorktree); err != nil {
				logging.Logger.Warn("Failed to move worktree", "session", repoSessions[i].Name, "error", err)
				fmt.Printf("⚠ Warning: Failed to move worktree for %s: %v\n", repoSessions[i].Name, err)
			} else {
				movedWorktreePaths = append(movedWorktreePaths, destWorktree)
				fmt.Printf("✓ Moved worktree '%s'\n", repoSessions[i].Name)
			}
		}

		// Add session to destination store
		logging.Logger.Debug("Adding session to destination store", "session", repoSessions[i].Name)
		if err := params.DestSessionRepo.Add(ctx, repoSessions[i]); err != nil {
			logging.Logger.Error("Failed to add session to destination", "session", repoSessions[i].Name, "error", err)
			return nil, fmt.Errorf("failed to add session %s to destination: %w", repoSessions[i].Name, err)
		}
	}

	// Repair git worktree references if we moved .main and worktrees
	if mainRepoPath != "" && len(movedWorktreePaths) > 0 {
		fmt.Printf("Repairing git worktree references...\n")
		logging.Logger.Info("Repairing worktree references", "mainRepo", mainRepoPath, "worktreeCount", len(movedWorktreePaths))

		if err := s.gitRepo.RepairWorktrees(mainRepoPath, movedWorktreePaths); err != nil {
			logging.Logger.Error("Failed to repair worktrees", "error", err)
			return nil, fmt.Errorf("failed to repair worktrees: %w", err)
		}
		fmt.Printf("✓ Repaired worktree references\n")
	}

	// Collect moved session names
	movedSessionNames := make([]string, len(repoSessions))
	for i, sess := range repoSessions {
		movedSessionNames[i] = sess.Name
	}

	logging.Logger.Info("Repository moved successfully", "repo", params.RepoInfo, "sessionCount", len(movedSessionNames))
	return movedSessionNames, nil
}

// MoveSessionParams contains parameters for moving a single session
type MoveSessionParams struct {
	DestRochaHome   string
	DestSessionRepo ports.SessionRepository
	Session         domain.Session
	SourceRochaHome string
}

// MoveSession handles moving a single session between stores
func (s *MigrationService) MoveSession(
	ctx context.Context,
	params MoveSessionParams,
) error {
	logging.Logger.Info("Moving session",
		"session", params.Session.Name,
		"from", params.SourceRochaHome,
		"to", params.DestRochaHome)

	sess := params.Session

	// Kill tmux session (graceful failure - session might not be running)
	logging.Logger.Debug("Killing tmux session", "session", sess.Name)
	if err := s.tmuxClient.KillSession(sess.Name); err != nil {
		logging.Logger.Warn("Failed to kill tmux session", "session", sess.Name, "error", err)
		fmt.Printf("⚠ Warning: Failed to kill tmux session %s: %v\n", sess.Name, err)
	}

	// Kill shell session if exists
	if sess.ShellSession != nil {
		shellName := sess.ShellSession.Name
		logging.Logger.Debug("Killing shell session", "session", shellName)
		if err := s.tmuxClient.KillSession(shellName); err != nil {
			logging.Logger.Warn("Failed to kill shell session", "session", shellName, "error", err)
			fmt.Printf("⚠ Warning: Failed to kill shell session %s: %v\n", shellName, err)
		}
	}

	// Update paths in session
	logging.Logger.Debug("Updating session paths", "session", sess.Name)
	s.updateSessionPaths(&sess, params.SourceRochaHome, params.DestRochaHome)

	// Move worktree if it exists
	if sess.WorktreePath != "" {
		sourceWorktree := strings.Replace(sess.WorktreePath, params.DestRochaHome, params.SourceRochaHome, 1)
		destWorktree := sess.WorktreePath

		logging.Logger.Info("Moving worktree", "session", sess.Name, "from", sourceWorktree, "to", destWorktree)
		if err := s.moveWorktree(sourceWorktree, destWorktree); err != nil {
			logging.Logger.Warn("Failed to move worktree", "session", sess.Name, "error", err)
			fmt.Printf("⚠ Warning: Failed to move worktree for %s: %v\n", sess.Name, err)
		} else {
			logging.Logger.Info("Worktree moved successfully", "session", sess.Name)
		}
	}

	// Add session to destination store
	logging.Logger.Debug("Adding session to destination store", "session", sess.Name)
	if err := params.DestSessionRepo.Add(ctx, sess); err != nil {
		logging.Logger.Error("Failed to add session to destination", "session", sess.Name, "error", err)
		return fmt.Errorf("failed to add session %s to destination: %w", sess.Name, err)
	}

	logging.Logger.Info("Session moved successfully", "session", sess.Name)
	return nil
}

// VerifySession confirms session exists in destination store
func (s *MigrationService) VerifySession(
	ctx context.Context,
	sessionName string,
	destRepo ports.SessionReader,
) error {
	logging.Logger.Debug("Verifying session in destination", "session", sessionName)
	_, err := destRepo.Get(ctx, sessionName)
	if err != nil {
		logging.Logger.Error("Verification failed - session not found in destination", "session", sessionName, "error", err)
		return fmt.Errorf("verification failed - session %s not found in destination: %w", sessionName, err)
	}
	logging.Logger.Info("Session verified successfully", "session", sessionName)
	return nil
}

// updateSessionPaths updates WorktreePath, RepoPath, and ClaudeDir in session
func (s *MigrationService) updateSessionPaths(sess *domain.Session, sourceRochaHome, destRochaHome string) {
	logging.Logger.Debug("Updating session paths", "session", sess.Name, "from", sourceRochaHome, "to", destRochaHome)

	// Update WorktreePath
	if sess.WorktreePath != "" {
		sess.WorktreePath = strings.Replace(sess.WorktreePath, sourceRochaHome, destRochaHome, 1)
	}

	// Update RepoPath (main repository directory path)
	if sess.RepoPath != "" {
		sess.RepoPath = strings.Replace(sess.RepoPath, sourceRochaHome, destRochaHome, 1)
	}

	// Update ClaudeDir if it references sourceRochaHome
	if sess.ClaudeDir != "" && strings.Contains(sess.ClaudeDir, sourceRochaHome) {
		sess.ClaudeDir = strings.Replace(sess.ClaudeDir, sourceRochaHome, destRochaHome, 1)
	}

	// Update shell session paths if exists
	if sess.ShellSession != nil {
		s.updateSessionPaths(sess.ShellSession, sourceRochaHome, destRochaHome)
	}
}

// moveMainDirectory moves a main repository directory from source to destination
func (s *MigrationService) moveMainDirectory(sourcePath, destPath string) error {
	logging.Logger.Info("Moving main repository directory", "from", sourcePath, "to", destPath)

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		logging.Logger.Warn("Source main repository directory does not exist", "path", sourcePath)
		return fmt.Errorf("source main repository directory does not exist: %s", sourcePath)
	}

	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		// Destination exists - check if it's the same repo
		logging.Logger.Debug("Destination main repository directory already exists, checking if same repo", "path", destPath)

		sourceRemote := s.gitRepo.GetRemoteURL(sourcePath)
		destRemote := s.gitRepo.GetRemoteURL(destPath)

		if sourceRemote == "" || destRemote == "" {
			logging.Logger.Warn("Could not get remote URL for comparison", "sourceRemote", sourceRemote, "destRemote", destRemote)
			return fmt.Errorf("main repository directory already exists at destination: %s", destPath)
		}

		// Normalize URLs for comparison
		if !s.isSameRepo(sourceRemote, destRemote) {
			logging.Logger.Error("Destination main repository is different repository", "sourceRemote", sourceRemote, "destRemote", destRemote)
			return fmt.Errorf("main repository directory at destination is a different repository.\nSource: %s\nDestination: %s", sourceRemote, destRemote)
		}

		// Same repo - use existing main repository, don't move
		logging.Logger.Info("Destination main repository is same repository, using existing", "path", destPath)
		fmt.Printf("✓ Using existing main repository at destination (same repository)\n")
		return nil
	}

	// Create parent directories for destination
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Try atomic rename first (works if same filesystem)
	err := os.Rename(sourcePath, destPath)
	if err == nil {
		logging.Logger.Info("Main repository directory moved using atomic rename", "from", sourcePath, "to", destPath)
		return nil
	}

	// If rename fails, fall back to copy + delete (for cross-filesystem moves)
	if err := s.copyDirectory(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to copy main repository directory: %w", err)
	}

	// Remove source after successful copy
	if err := os.RemoveAll(sourcePath); err != nil {
		return fmt.Errorf("failed to remove source main repository directory after copy: %w", err)
	}

	logging.Logger.Info("Main repository directory moved using copy+delete", "from", sourcePath, "to", destPath)
	return nil
}

// isSameRepo checks if two URLs point to the same repository
func (s *MigrationService) isSameRepo(url1, url2 string) bool {
	normalize := func(url string) string {
		url = strings.TrimSuffix(url, ".git")
		url = strings.TrimSuffix(url, "/")
		url = strings.ToLower(url)

		if strings.HasPrefix(url, "https://") {
			url = strings.TrimPrefix(url, "https://")
		} else if strings.HasPrefix(url, "http://") {
			url = strings.TrimPrefix(url, "http://")
		}
		if strings.HasPrefix(url, "ssh://") {
			url = strings.TrimPrefix(url, "ssh://")
			if idx := strings.Index(url, "@"); idx >= 0 {
				url = url[idx+1:]
			}
		}
		if strings.Contains(url, "@") && strings.Contains(url, ":") {
			parts := strings.SplitN(url, "@", 2)
			if len(parts) == 2 {
				url = strings.Replace(parts[1], ":", "/", 1)
			}
		}
		return url
	}
	return normalize(url1) == normalize(url2)
}

// moveWorktree moves a worktree from source to destination
func (s *MigrationService) moveWorktree(sourcePath, destPath string) error {
	logging.Logger.Info("Moving worktree", "from", sourcePath, "to", destPath)

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source worktree does not exist: %s", sourcePath)
	}

	// Create parent directories for destination
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Try atomic rename first
	err := os.Rename(sourcePath, destPath)
	if err == nil {
		return nil
	}

	// Fall back to copy + delete
	if err := s.copyDirectory(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to copy worktree: %w", err)
	}

	if err := os.RemoveAll(sourcePath); err != nil {
		return fmt.Errorf("failed to remove source worktree after copy: %w", err)
	}

	return nil
}

// copyDirectory recursively copies a directory
func (s *MigrationService) copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := s.copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := s.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func (s *MigrationService) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
