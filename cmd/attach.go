package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/tmux"
)

// AttachCmd attaches to a tmux session, creating it if needed
type AttachCmd struct {
	SessionName string      `help:"Override session name (default: auto-detect from branch/directory)"`
	Repo        string      `help:"Override repository path (default: auto-detect)"`
	Branch      string      `help:"Override branch name (default: auto-detect from git)"`
	Worktree    string      `help:"Override worktree path (default: current directory)"`
	TmuxClient  tmux.Client `kong:"-"`
}

// Run executes the attach command
func (a *AttachCmd) Run() error {
	logging.Logger.Info("Attach command started")

	// Step 1: Auto-detect parameters from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	var repoPath, branchName, repoInfo, worktreePath, sessionName string

	// Check if in git repo
	isGit, _ := git.IsGitRepo(cwd)
	if isGit {
		// Get main repo path (handles worktrees correctly)
		mainRepoPath, err := git.GetMainRepoPath(cwd)
		if err != nil {
			logging.Logger.Warn("Failed to get main repo path, using detected path", "error", err)
			mainRepoPath = cwd
		}
		repoPath = mainRepoPath
		branchName = git.GetBranchName(cwd)
		repoInfo = git.GetRepoInfo(repoPath)
		worktreePath = cwd
		sessionName = branchName // Use branch as default session name
	} else {
		repoPath = cwd
		worktreePath = cwd
		sessionName = filepath.Base(cwd) // Use directory name
	}

	// Step 2: Apply flag overrides
	if a.SessionName != "" {
		sessionName = a.SessionName
	}
	if a.Repo != "" {
		repoPath = a.Repo
	}
	if a.Branch != "" {
		branchName = a.Branch
	}
	if a.Worktree != "" {
		worktreePath = a.Worktree
	}

	// Step 3: Sanitize ONLY session name for tmux (NOT repo/branch/worktree)
	sessionName = sanitizeSessionName(sessionName)

	if sessionName == "" {
		return fmt.Errorf("could not determine session name (no branch, no directory name)")
	}

	logging.Logger.Info("Session parameters determined",
		"session_name", sessionName,
		"repo_path", repoPath,
		"branch_name", branchName,
		"worktree_path", worktreePath)

	// Step 3.5: Check for duplicate sessions with same branch or worktree
	st, err := state.Load()
	if err != nil && !os.IsNotExist(err) {
		logging.Logger.Warn("Failed to load state for duplicate check", "error", err)
	}
	if st != nil && st.Sessions != nil {
		for existingName, existingSession := range st.Sessions {
			// Skip if it's the same session name (we'll just attach to it)
			if existingName == sessionName {
				continue
			}

			// Check for duplicate branch (if in git repo)
			if branchName != "" && existingSession.BranchName == branchName && existingSession.RepoPath == repoPath {
				return fmt.Errorf("session '%s' already exists for branch '%s' in repo '%s'", existingName, branchName, repoPath)
			}

			// Check for duplicate worktree path
			if worktreePath != "" && existingSession.WorktreePath == worktreePath {
				return fmt.Errorf("session '%s' already exists for worktree path '%s'", existingName, worktreePath)
			}
		}
	}

	// Step 4: Check if tmux session exists, create if needed
	if !a.TmuxClient.Exists(sessionName) {
		logging.Logger.Info("Session does not exist, creating", "name", sessionName)
		_, err := a.TmuxClient.Create(sessionName, worktreePath)
		if err != nil {
			return fmt.Errorf("failed to create tmux session: %w", err)
		}
		fmt.Printf("Created new session '%s'\n", sessionName)
	} else {
		logging.Logger.Info("Session already exists", "name", sessionName)
		fmt.Printf("Attaching to existing session '%s'\n", sessionName)
	}

	// Step 5: Update state.json
	// Reload state in case it changed
	st, err = state.Load()
	if err != nil && !os.IsNotExist(err) {
		logging.Logger.Warn("Failed to load state", "error", err)
	}
	if st == nil {
		st = &state.SessionState{Sessions: make(map[string]state.SessionInfo)}
	}

	// Create or update session info - preserve DisplayName if session exists
	sessionInfo, exists := st.Sessions[sessionName]
	if exists {
		// Session exists - preserve DisplayName and update only necessary fields
		sessionInfo.State = "working"
		sessionInfo.ExecutionID = fmt.Sprintf("%d", time.Now().Unix())
		sessionInfo.LastUpdated = time.Now()
		// Update git metadata if provided
		if branchName != "" {
			sessionInfo.BranchName = branchName
		}
		if worktreePath != "" {
			sessionInfo.WorktreePath = worktreePath
		}
		if repoPath != "" {
			sessionInfo.RepoPath = repoPath
		}
		if repoInfo != "" {
			sessionInfo.RepoInfo = repoInfo
		}
	} else {
		// New session - create with all fields
		sessionInfo = state.SessionInfo{
			Name:         sessionName,
			DisplayName:  sessionName,
			State:        "working",
			ExecutionID:  fmt.Sprintf("%d", time.Now().Unix()),
			LastUpdated:  time.Now(),
			BranchName:   branchName,
			WorktreePath: worktreePath,
			RepoPath:     repoPath,
			RepoInfo:     repoInfo,
		}
	}

	st.Sessions[sessionName] = sessionInfo

	if err := st.Save(); err != nil {
		logging.Logger.Error("Failed to save state", "error", err)
		// Continue anyway - session is created
	}

	// Step 6: Attach to tmux session
	logging.Logger.Info("Attaching to tmux session", "name", sessionName)

	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logging.Logger.Error("Tmux attach failed", "error", err)
		return fmt.Errorf("failed to attach to session: %w", err)
	}

	return nil
}

// sanitizeSessionName cleans a string to be tmux-safe
func sanitizeSessionName(name string) string {
	// Replace spaces and special characters
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ToLower(name)

	// Remove any other problematic characters
	// Tmux doesn't like: . : (in some contexts)
	// Keep: a-z 0-9 - _
	var result strings.Builder
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			result.WriteRune(ch)
		}
	}

	return result.String()
}
