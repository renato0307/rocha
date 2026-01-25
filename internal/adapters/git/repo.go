package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"rocha/internal/logging"
)

// repoSource represents parsed repository source (internal)
type repoSource struct {
	branch   string // Branch name from URL fragment (e.g., #branch-name)
	isRemote bool
	owner    string // From github.com/owner/repo or similar
	path     string // URL or local path (without branch fragment)
	repo     string // From github.com/owner/repo or similar
}

// isGitURL checks if string is git URL (https://, git@, ssh://)
func isGitURL(source string) bool {
	if source == "" {
		return false
	}

	// Match common git URL patterns
	patterns := []string{
		`^https?://`,           // https:// or http://
		`^git@`,                // git@github.com:owner/repo
		`^ssh://`,              // ssh://git@github.com/owner/repo
		`^git://`,              // git://github.com/owner/repo
		`^ftps?://`,            // ftp:// or ftps://
		`\.git(/|\\)?$`,        // ends with .git
		`^[a-zA-Z0-9.-]+@.*:`, // generic user@host:path format
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, source)
		if matched {
			return true
		}
	}

	return false
}

// parseRepoSource parses repository path or URL
// Returns repoSource with parsed information
// Supports branch specification via URL fragment: https://github.com/owner/repo#branch-name
func parseRepoSource(source string) (*repoSource, error) {
	logging.Logger.Debug("Parsing repo source", "source", source)

	if source == "" {
		return nil, fmt.Errorf("empty source")
	}

	// Extract branch from URL fragment (after #)
	var branch string
	if idx := strings.Index(source, "#"); idx >= 0 {
		branch = source[idx+1:]
		source = source[:idx]
		logging.Logger.Debug("Extracted branch from URL", "branch", branch)
	}

	rs := &repoSource{
		branch:   branch,
		isRemote: isGitURL(source),
		path:     source,
	}

	// Try to extract owner/repo from URL
	if rs.isRemote {
		// Remove .git suffix if present
		cleanURL := strings.TrimSuffix(source, ".git")

		var ownerRepo string

		// Parse different URL formats:
		// - https://github.com/owner/repo.git
		// - git@github.com:owner/repo.git
		// - ssh://git@github.com/owner/repo.git
		// - https://gitlab.com/owner/repo.git
		// - git@gitlab.com:owner/repo.git

		if strings.HasPrefix(cleanURL, "https://") || strings.HasPrefix(cleanURL, "http://") {
			// https://github.com/owner/repo
			// Find the host and extract everything after it
			parts := strings.SplitN(cleanURL, "://", 2)
			if len(parts) == 2 {
				// Split by / to get path components
				pathParts := strings.Split(parts[1], "/")
				if len(pathParts) >= 3 {
					// Extract last two components as owner/repo
					owner := pathParts[len(pathParts)-2]
					repo := pathParts[len(pathParts)-1]
					ownerRepo = fmt.Sprintf("%s/%s", owner, repo)
				}
			}
		} else if strings.HasPrefix(cleanURL, "git@") {
			// git@github.com:owner/repo
			parts := strings.Split(cleanURL, ":")
			if len(parts) >= 2 {
				// Join remaining parts (in case there are colons in the path)
				ownerRepo = strings.Join(parts[1:], ":")
			}
		} else if strings.HasPrefix(cleanURL, "ssh://") {
			// ssh://git@github.com/owner/repo
			// Remove ssh:// prefix
			path := strings.TrimPrefix(cleanURL, "ssh://")
			// Remove user@ prefix if present
			if idx := strings.Index(path, "@"); idx >= 0 {
				path = path[idx+1:]
			}
			// Split by / and get last two components
			parts := strings.Split(path, "/")
			if len(parts) >= 3 {
				owner := parts[len(parts)-2]
				repo := parts[len(parts)-1]
				ownerRepo = fmt.Sprintf("%s/%s", owner, repo)
			}
		}

		if ownerRepo != "" {
			parts := strings.Split(ownerRepo, "/")
			if len(parts) == 2 {
				rs.owner = parts[0]
				rs.repo = parts[1]
				logging.Logger.Debug("Parsed remote repo", "owner", rs.owner, "repo", rs.repo)
			}
		} else {
			logging.Logger.Warn("Could not extract owner/repo from URL", "url", source)
		}
	}

	return rs, nil
}

// cloneRepository clones git repo to target path
// If branch is specified, clones only that branch (--single-branch)
// If branch is empty, clones all branches (for shared .main)
func cloneRepository(url, targetPath, branch string) error {
	logging.Logger.Info("Cloning repository", "url", url, "target", targetPath, "branch", branch)

	// Ensure parent directory exists
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		logging.Logger.Error("Failed to create parent directory", "error", err, "path", parentDir)
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Build git clone command
	args := []string{"clone"}

	// IMPORTANT: Only use --single-branch if branch is specified
	// This allows .main to support multiple branches dynamically
	if branch != "" {
		args = append(args, "--branch", branch, "--single-branch")
		logging.Logger.Debug("Cloning single branch", "branch", branch)
	} else {
		// Clone all branches (no --single-branch flag)
		logging.Logger.Info("Cloning all branches for shared .main")
	}

	args = append(args, url, targetPath)

	// Clone the repository
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logging.Logger.Error("Git clone failed", "error", err, "output", string(output))
		return fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
	}

	logging.Logger.Info("Repository cloned successfully", "path", targetPath, "branch", branch)
	return nil
}

// checkoutBranch ensures the specified branch is checked out in the repo
// If branch doesn't exist locally, fetches from remote and creates it
func checkoutBranch(repoPath, branch string) error {
	if branch == "" {
		return nil // No branch specified, use current branch
	}

	logging.Logger.Info("Checking out branch", "repo", repoPath, "branch", branch)

	// First, try a simple checkout (works if branch exists locally)
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Branch doesn't exist locally, fetch all refs from remote
		logging.Logger.Debug("Branch not found locally, fetching from remote", "branch", branch)

		// Fetch all refs from origin to get the latest remote branches
		fetchCmd := exec.Command("git", "fetch", "origin")
		fetchCmd.Dir = repoPath
		if fetchOutput, fetchErr := fetchCmd.CombinedOutput(); fetchErr != nil {
			return fmt.Errorf("failed to fetch from origin: %w\nOutput: %s", fetchErr, string(fetchOutput))
		}

		// Now try checkout again - git will auto-create local branch from remote
		cmd = exec.Command("git", "checkout", branch)
		cmd.Dir = repoPath
		if output, err = cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to checkout branch %s: %w\nOutput: %s", branch, err, string(output))
		}
	}

	logging.Logger.Info("Successfully checked out branch", "branch", branch)
	return nil
}

// getCurrentBranch returns the currently checked out branch
func getCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getOrCloneRepository ensures repo exists locally
// Local path: validate and return
// Remote URL: clone to {worktreeBase}/{owner}/{repo}/.main
// Returns: localPath, repoSource, error
func getOrCloneRepository(source, worktreeBase string) (string, *repoSource, error) {
	logging.Logger.Debug("Getting or cloning repository", "source", source, "worktree_base", worktreeBase)

	// Parse the source
	repoSource, err := parseRepoSource(source)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse repo source: %w", err)
	}

	if !repoSource.isRemote {
		// Local path - validate it exists and is a git repo
		logging.Logger.Debug("Source is local path", "path", source)

		// Expand ~ in path
		if strings.HasPrefix(source, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			source = filepath.Join(homeDir, source[1:])
		}

		// Check if path exists
		if _, err := os.Stat(source); os.IsNotExist(err) {
			return "", nil, fmt.Errorf("local path does not exist: %s", source)
		}

		// Check if it's a git repository
		isGit, repoRoot := isGitRepo(source)
		if !isGit {
			return "", nil, fmt.Errorf("local path is not a git repository: %s", source)
		}

		// Try to extract owner/repo from local git repo
		repoInfo := getRepoInfo(repoRoot)
		if repoInfo != "" && strings.Contains(repoInfo, "/") {
			parts := strings.Split(repoInfo, "/")
			if len(parts) == 2 {
				repoSource.owner = parts[0]
				repoSource.repo = parts[1]
			}
		}

		logging.Logger.Info("Using local git repository", "path", repoRoot)
		return repoRoot, repoSource, nil
	}

	// Remote URL - clone to worktree base
	logging.Logger.Debug("Source is remote URL", "url", source)

	// Build target path: {worktreeBase}/{owner}/{repo}/.main
	if repoSource.owner == "" || repoSource.repo == "" {
		return "", nil, fmt.Errorf("could not extract owner/repo from URL: %s", source)
	}

	targetPath := filepath.Join(worktreeBase, repoSource.owner, repoSource.repo, ".main")
	logging.Logger.Debug("Target clone path", "path", targetPath)

	// Check if .main directory already exists
	if _, err := os.Stat(targetPath); err == nil {
		logging.Logger.Debug("Clone target already exists, verifying and checking out branch")

		// Verify it's a git repo
		isGit, repoRoot := isGitRepo(targetPath)
		if !isGit {
			return "", nil, fmt.Errorf(".main directory exists but is not a git repository: %s", targetPath)
		}

		// Verify remote URL matches
		existingURL := getRemoteURL(repoRoot)
		if existingURL != "" && !isSameRepo(existingURL, repoSource.path) {
			return "", nil, fmt.Errorf(".main directory exists with different remote URL.\nExisting: %s\nRequested: %s", existingURL, repoSource.path)
		}

		// CRITICAL FIX: Checkout the requested branch before returning
		if repoSource.branch != "" {
			if err := checkoutBranch(repoRoot, repoSource.branch); err != nil {
				return "", nil, fmt.Errorf("failed to checkout branch: %w", err)
			}
		}

		// Pull latest changes
		pullCmd := exec.Command("git", "pull", "--rebase")
		pullCmd.Dir = repoRoot
		pullCmd.Run() // Ignore errors (might be detached HEAD)

		logging.Logger.Info("Reusing .main with correct branch", "path", repoRoot, "branch", repoSource.branch)
		return repoRoot, repoSource, nil
	}

	// Clone repository (with all branches for shared .main)
	// NOTE: Pass empty string for branch to clone all branches
	if err := cloneRepository(repoSource.path, targetPath, ""); err != nil {
		// Cleanup on failure
		os.RemoveAll(targetPath)
		return "", nil, err
	}

	// If branch was specified, checkout that branch after cloning
	if repoSource.branch != "" {
		if err := checkoutBranch(targetPath, repoSource.branch); err != nil {
			os.RemoveAll(targetPath)
			return "", nil, err
		}
	}

	return targetPath, repoSource, nil
}

// getRemoteURL gets the remote URL for origin
func getRemoteURL(repoPath string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// isSameRepo checks if two URLs point to the same repository
// Normalizes URLs for comparison (handles .git suffix, https vs ssh, etc.)
func isSameRepo(url1, url2 string) bool {
	normalize := func(url string) string {
		// Remove .git suffix
		url = strings.TrimSuffix(url, ".git")
		url = strings.TrimSuffix(url, "/")

		// Convert to lowercase for comparison
		url = strings.ToLower(url)

		// Normalize different URL formats to canonical form (host/owner/repo)
		// Handle: https://github.com/owner/repo
		if strings.HasPrefix(url, "https://") {
			url = strings.TrimPrefix(url, "https://")
		} else if strings.HasPrefix(url, "http://") {
			url = strings.TrimPrefix(url, "http://")
		}
		// Handle: ssh://git@github.com/owner/repo
		if strings.HasPrefix(url, "ssh://") {
			url = strings.TrimPrefix(url, "ssh://")
			// Remove user@ part
			if idx := strings.Index(url, "@"); idx >= 0 {
				url = url[idx+1:]
			}
		}
		// Handle: git@github.com:owner/repo
		if strings.Contains(url, "@") && strings.Contains(url, ":") {
			// Format: git@host:path
			parts := strings.SplitN(url, "@", 2)
			if len(parts) == 2 {
				// parts[1] is "host:path"
				url = strings.Replace(parts[1], ":", "/", 1)
			}
		}

		return url
	}

	return normalize(url1) == normalize(url2)
}
