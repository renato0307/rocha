package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/logging"
)

// isGitRepo checks if the given path is within a git repository
// Returns true and the repository root path if it is, false and empty string otherwise
// NOTE: For worktrees, this returns the worktree path, not the main repo path
// Use getMainRepoPath to get the main repository path for worktrees
func isGitRepo(path string) (bool, string) {
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

// getMainRepoPath gets the main repository path, even for worktrees
// For regular repos, returns the same as isGitRepo
// For worktrees, returns the path to the main repository
func getMainRepoPath(path string) (string, error) {
	logging.Logger.Debug("Getting main repo path", "path", path)

	// Get the git common directory (points to main repo for worktrees)
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Debug("Failed to get git common dir", "error", err)
		return "", fmt.Errorf("failed to get git common dir: %w", err)
	}

	gitCommonDir := strings.TrimSpace(string(output))

	// If it's a relative path (like ".git"), resolve it relative to path
	if !filepath.IsAbs(gitCommonDir) {
		gitCommonDir = filepath.Join(path, gitCommonDir)
	}

	// The main repo path is the parent of the .git directory
	mainRepoPath := filepath.Dir(gitCommonDir)

	// Clean the path to resolve any .. or .
	mainRepoPath = filepath.Clean(mainRepoPath)

	logging.Logger.Info("Found main repo path", "main_repo_path", mainRepoPath)
	return mainRepoPath, nil
}

// branchExists checks if a branch exists locally or remotely
func branchExists(repoPath, branchName string) bool {
	logging.Logger.Debug("Checking if branch exists", "repo_path", repoPath, "branch", branchName)

	// First check if branch exists locally
	cmd := exec.Command("git", "show-ref", "--verify", fmt.Sprintf("refs/heads/%s", branchName))
	cmd.Dir = repoPath
	if output, err := cmd.Output(); err == nil && len(output) > 0 {
		logging.Logger.Debug("Branch exists locally", "branch", branchName)
		return true
	}

	// Check if branch exists remotely
	cmd = exec.Command("git", "ls-remote", "--heads", "origin", branchName)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		logging.Logger.Debug("Branch exists remotely", "branch", branchName)
		return true
	}

	logging.Logger.Debug("Branch does not exist", "branch", branchName)
	return false
}

// createWorktree creates a new git worktree at the specified path
// If the branch exists, it checks it out; if not, it creates a new branch
// It ensures the worktree is created from the latest origin/main by fetching,
// checking out main, and resetting to origin/main before creating the worktree
func createWorktree(repoPath, worktreePath, branchName string) error {
	logging.Logger.Info("Creating worktree", "repo_path", repoPath, "worktree_path", worktreePath, "branch_name", branchName)

	// Ensure the base worktree directory exists
	worktreeBase := filepath.Dir(worktreePath)
	if err := os.MkdirAll(worktreeBase, 0755); err != nil {
		logging.Logger.Error("Failed to create worktree base directory", "error", err, "path", worktreeBase)
		return fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	// Fetch from origin to get latest remote state
	logging.Logger.Info("Fetching from origin", "repo_path", repoPath)
	fetchCmd := exec.Command("git", "fetch", "origin")
	fetchCmd.Dir = repoPath

	if output, err := fetchCmd.CombinedOutput(); err != nil {
		logging.Logger.Warn("Git fetch origin failed (continuing anyway)", "error", err, "output", string(output))
		// Don't fail - user might be offline
	} else {
		logging.Logger.Debug("Git fetch origin succeeded")
	}

	// Checkout main branch to ensure worktree is created from main
	logging.Logger.Info("Checking out main branch", "repo_path", repoPath)
	checkoutCmd := exec.Command("git", "checkout", "main")
	checkoutCmd.Dir = repoPath

	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		logging.Logger.Warn("Git checkout main failed (continuing anyway)", "error", err, "output", string(output))
	} else {
		logging.Logger.Debug("Git checkout main succeeded")
	}

	// Reset to origin/main to get latest state
	logging.Logger.Info("Resetting to origin/main", "repo_path", repoPath)
	resetCmd := exec.Command("git", "reset", "--hard", "origin/main")
	resetCmd.Dir = repoPath

	if output, err := resetCmd.CombinedOutput(); err != nil {
		logging.Logger.Warn("Git reset to origin/main failed (continuing anyway)", "error", err, "output", string(output))
	} else {
		logging.Logger.Debug("Git reset to origin/main succeeded")
	}

	// Validate branch name before creating worktree
	if err := validateBranchName(branchName); err != nil {
		logging.Logger.Error("Invalid branch name", "branch", branchName, "error", err)
		return fmt.Errorf("invalid branch name: %w", err)
	}

	// Check if branch exists (locally or remotely)
	exists := branchExists(repoPath, branchName)
	logging.Logger.Debug("Branch existence check", "branch", branchName, "exists", exists)

	// Create the worktree
	var worktreeCmd *exec.Cmd
	if exists {
		// Branch exists - check it out in the worktree
		logging.Logger.Info("Checking out existing branch in worktree", "path", worktreePath, "branch", branchName)
		worktreeCmd = exec.Command("git", "worktree", "add", worktreePath, branchName)
	} else {
		// Branch doesn't exist - create new branch in worktree
		logging.Logger.Info("Creating new branch in worktree", "path", worktreePath, "branch", branchName)
		worktreeCmd = exec.Command("git", "worktree", "add", worktreePath, "-b", branchName)
	}
	worktreeCmd.Dir = repoPath

	if output, err := worktreeCmd.CombinedOutput(); err != nil {
		logging.Logger.Error("Git worktree add failed", "error", err, "output", string(output))
		return fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
	}

	logging.Logger.Info("Git worktree created successfully", "path", worktreePath, "branch", branchName)
	return nil
}

// removeWorktree removes a git worktree at the specified path
// repoPath is the main repository path where the git command should be run from
func removeWorktree(repoPath, worktreePath string) error {
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

// worktreeInfo holds parsed information about a git worktree
type worktreeInfo struct {
	branch string
	path   string
}

// parseWorktreeList parses git worktree list --porcelain output
// Returns a slice of worktreeInfo with path and branch for each worktree
func parseWorktreeList(output string) []worktreeInfo {
	var worktrees []worktreeInfo
	var current worktreeInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "worktree "):
			// Start of a new worktree entry
			if current.path != "" {
				worktrees = append(worktrees, current)
			}
			current = worktreeInfo{path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch "):
			current.branch = strings.TrimPrefix(line, "branch ")
		}
	}

	// Don't forget the last entry
	if current.path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// fetchWorktreeList executes git worktree list and returns parsed results
func fetchWorktreeList(repoPath string) ([]worktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeList(string(output)), nil
}

// listWorktrees lists all worktrees for the given repository
func listWorktrees(repoPath string) ([]string, error) {
	logging.Logger.Debug("Listing worktrees", "repo_path", repoPath)

	worktrees, err := fetchWorktreeList(repoPath)
	if err != nil {
		logging.Logger.Error("Failed to list worktrees", "error", err)
		return nil, err
	}

	paths := make([]string, len(worktrees))
	for i, wt := range worktrees {
		paths[i] = wt.path
	}

	logging.Logger.Debug("Found worktrees", "count", len(paths))
	return paths, nil
}

// getBranchName returns the current branch name for the given path
// Returns empty string if not in a git repository or cannot determine branch
func getBranchName(path string) string {
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

// getRepoInfo extracts owner/repo from git remote origin
// Returns in format "owner/repo" or empty string if not found
func getRepoInfo(repoPath string) string {
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

// sanitizePathComponent sanitizes a string for safe use as a path component
// Similar to SanitizeBranchName but for filesystem paths
// NOTE: We preserve original casing to avoid issues with gopls on macOS,
// which does exact string matching on paths even though macOS filesystem
// is case-insensitive.
func sanitizePathComponent(component string) string {
	if component == "" {
		return ""
	}

	// Remove control characters and problematic filesystem chars
	var builder strings.Builder
	for _, r := range component {
		if !unicode.IsControl(r) && r != '/' && r != '\\' && r != ':' {
			builder.WriteRune(r)
		}
	}

	result := builder.String()
	result = strings.TrimSpace(result)

	// Replace problematic sequences
	result = strings.ReplaceAll(result, "..", ".")

	return result
}

// buildWorktreePath constructs a worktree path with repository organization
// If repoInfo is available (format "owner/repo"), creates: base/owner/repo/sessionName
// If repoInfo is empty or invalid, falls back to: base/sessionName
func buildWorktreePath(base, repoInfo, sessionName string) string {
	logging.Logger.Debug("Building worktree path",
		"base", base, "repo_info", repoInfo, "session_name", sessionName)

	// Sanitize session name for filesystem
	sanitizedSession := sanitizePathComponent(sessionName)
	if sanitizedSession == "" {
		logging.Logger.Warn("Session name sanitization resulted in empty string, using fallback")
		sanitizedSession = "session"
	}

	// Check if we have valid repo info (format: "owner/repo")
	if repoInfo != "" && strings.Contains(repoInfo, "/") {
		parts := strings.Split(repoInfo, "/")
		if len(parts) == 2 {
			owner := sanitizePathComponent(parts[0])
			repo := sanitizePathComponent(parts[1])

			if owner != "" && repo != "" {
				path := filepath.Join(base, owner, repo, sanitizedSession)
				logging.Logger.Info("Built repository-organized worktree path",
					"path", path, "owner", owner, "repo", repo)
				return path
			}
		}
	}

	// Fallback to flat structure if repo info unavailable
	logging.Logger.Warn("RepoInfo not available or invalid, using flat structure",
		"repo_info", repoInfo)
	return filepath.Join(base, sanitizedSession)
}

// getWorktreeForBranch finds an existing worktree for a branch
// Returns the worktree path if found, empty string if not
// Excludes the main repository directory (.main) from search results
func getWorktreeForBranch(repoPath, branchName string) (string, error) {
	logging.Logger.Debug("Looking for existing worktree for branch",
		"repo_path", repoPath, "branch", branchName)

	worktrees, err := fetchWorktreeList(repoPath)
	if err != nil {
		logging.Logger.Error("Failed to list worktrees", "error", err)
		return "", err
	}

	expectedBranchRef := fmt.Sprintf("refs/heads/%s", branchName)

	for _, wt := range worktrees {
		// Skip the main repository directory - we only want actual worktrees
		if strings.HasSuffix(wt.path, "/"+config.MainRepoDir) {
			logging.Logger.Debug("Skipping main repository directory", "path", wt.path)
			continue
		}

		if wt.branch == expectedBranchRef {
			logging.Logger.Info("Found existing worktree for branch",
				"branch", branchName, "worktree", wt.path)
			return wt.path, nil
		}
	}

	logging.Logger.Debug("No existing worktree found for branch", "branch", branchName)
	return "", nil
}

// repairWorktrees repairs git worktree references after moving
// Must be run from the main repository directory (config.MainRepoDir) with paths to all moved worktrees
// This is necessary when both the main repository and worktrees have been moved
func repairWorktrees(mainRepoPath string, worktreePaths []string) error {
	logging.Logger.Info("Repairing worktree references",
		"mainRepo", mainRepoPath,
		"worktreeCount", len(worktreePaths))

	if len(worktreePaths) == 0 {
		logging.Logger.Debug("No worktrees to repair")
		return nil
	}

	// Build args: git worktree repair <path1> <path2> ...
	args := []string{"worktree", "repair"}
	args = append(args, worktreePaths...)

	logging.Logger.Debug("Running git worktree repair", "args", args)

	cmd := exec.Command("git", args...)
	cmd.Dir = mainRepoPath // Run from main repository directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		logging.Logger.Error("Git worktree repair failed",
			"error", err,
			"output", string(output))
		return fmt.Errorf("failed to repair worktrees: %w\nOutput: %s",
			err, string(output))
	}

	logging.Logger.Info("Worktree references repaired successfully",
		"output", string(output))
	return nil
}
