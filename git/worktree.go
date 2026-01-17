package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"rocha/logging"
	"strings"
)

// IsGitRepo checks if the given path is within a git repository
// Returns true and the repository root path if it is, false and empty string otherwise
func IsGitRepo(path string) (bool, string) {
	logging.Logger.Debug("Checking if directory is git repo", "path", path)

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Debug("Not a git repository", "path", path)
		return false, ""
	}

	repoRoot := strings.TrimSpace(string(output))
	logging.Logger.Info("Found git repository", "repo_root", repoRoot)
	return true, repoRoot
}

// CreateWorktree creates a new git worktree at the specified path with a new branch
// It first runs git pull --rebase to sync with remote
func CreateWorktree(repoPath, worktreePath, branchName string) error {
	logging.Logger.Info("Creating worktree", "repo_path", repoPath, "worktree_path", worktreePath, "branch_name", branchName)

	// Ensure the base worktree directory exists
	worktreeBase := filepath.Dir(worktreePath)
	if err := os.MkdirAll(worktreeBase, 0755); err != nil {
		logging.Logger.Error("Failed to create worktree base directory", "error", err, "path", worktreeBase)
		return fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	// Run git pull --rebase to sync with remote
	logging.Logger.Info("Running git pull --rebase", "repo_path", repoPath)
	pullCmd := exec.Command("git", "pull", "--rebase")
	pullCmd.Dir = repoPath

	if output, err := pullCmd.CombinedOutput(); err != nil {
		logging.Logger.Warn("Git pull --rebase failed (continuing anyway)", "error", err, "output", string(output))
		// Don't fail the entire operation if rebase fails - user might be offline or have uncommitted changes
		// We'll log the warning and continue with worktree creation
	} else {
		logging.Logger.Debug("Git pull --rebase succeeded")
	}

	// Create the worktree with new branch
	logging.Logger.Info("Running git worktree add", "path", worktreePath, "branch", branchName)
	worktreeCmd := exec.Command("git", "worktree", "add", worktreePath, "-b", branchName)
	worktreeCmd.Dir = repoPath

	if output, err := worktreeCmd.CombinedOutput(); err != nil {
		logging.Logger.Error("Git worktree add failed", "error", err, "output", string(output))
		return fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
	}

	logging.Logger.Info("Git worktree created successfully", "path", worktreePath, "branch", branchName)
	return nil
}

// RemoveWorktree removes a git worktree at the specified path
// repoPath is the main repository path where the git command should be run from
func RemoveWorktree(repoPath, worktreePath string) error {
	logging.Logger.Info("Removing worktree", "repo_path", repoPath, "worktree_path", worktreePath)

	// First check if the worktree path exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		logging.Logger.Warn("Worktree path does not exist", "path", worktreePath)
		return nil // Already removed, not an error
	}

	// Run git worktree remove with --force flag to allow removal even with uncommitted changes
	// This is appropriate for rocha's use case where worktrees are temporary development environments
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	cmd.Dir = repoPath

	if output, err := cmd.CombinedOutput(); err != nil {
		logging.Logger.Error("Git worktree remove failed", "error", err, "output", string(output))
		return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
	}

	logging.Logger.Info("Git worktree removed successfully", "path", worktreePath)
	return nil
}

// ListWorktrees lists all worktrees for the given repository
func ListWorktrees(repoPath string) ([]string, error) {
	logging.Logger.Debug("Listing worktrees", "repo_path", repoPath)

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Error("Failed to list worktrees", "error", err)
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Parse the porcelain output
	var worktrees []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			worktrees = append(worktrees, path)
		}
	}

	logging.Logger.Debug("Found worktrees", "count", len(worktrees))
	return worktrees, nil
}

// GetBranchName returns the current branch name for the given path
// Returns empty string if not in a git repository or cannot determine branch
func GetBranchName(path string) string {
	logging.Logger.Debug("Getting branch name", "path", path)

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Debug("Failed to get branch name", "error", err)
		return ""
	}

	branchName := strings.TrimSpace(string(output))
	logging.Logger.Debug("Found branch name", "branch", branchName)
	return branchName
}

// GetRepoInfo extracts owner/repo from git remote origin
// Returns in format "owner/repo" or empty string if not found
func GetRepoInfo(repoPath string) string {
	logging.Logger.Debug("Getting repo info", "repo_path", repoPath)

	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Debug("Failed to get remote URL", "error", err)
		return ""
	}

	remoteURL := strings.TrimSpace(string(output))
	logging.Logger.Debug("Remote URL", "url", remoteURL)

	// Parse different URL formats:
	// - https://github.com/owner/repo.git
	// - git@github.com:owner/repo.git
	// - ssh://git@github.com/owner/repo.git

	var ownerRepo string

	if strings.Contains(remoteURL, "github.com") {
		// Remove .git suffix if present
		remoteURL = strings.TrimSuffix(remoteURL, ".git")

		if strings.HasPrefix(remoteURL, "https://") {
			// https://github.com/owner/repo
			parts := strings.Split(remoteURL, "github.com/")
			if len(parts) == 2 {
				ownerRepo = parts[1]
			}
		} else if strings.HasPrefix(remoteURL, "git@") {
			// git@github.com:owner/repo
			parts := strings.Split(remoteURL, ":")
			if len(parts) == 2 {
				ownerRepo = parts[1]
			}
		} else if strings.HasPrefix(remoteURL, "ssh://") {
			// ssh://git@github.com/owner/repo
			parts := strings.Split(remoteURL, "github.com/")
			if len(parts) == 2 {
				ownerRepo = parts[1]
			}
		}
	}

	logging.Logger.Debug("Extracted repo info", "owner_repo", ownerRepo)
	return ownerRepo
}
