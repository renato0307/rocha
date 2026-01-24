package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"rocha/git"
	"rocha/logging"
	"rocha/paths"
	"rocha/state"
	"rocha/storage"
	"rocha/tmux"
)

// AttachCmd attaches to a tmux session, creating it if needed
type AttachCmd struct {
	AllowDangerouslySkipPermissions bool   `help:"Skip permission prompts in Claude (DANGEROUS - use with caution)"`
	Branch                          string `help:"Override branch name (default: auto-detect from git)"`
	Repo                            string `help:"Override repository path (default: auto-detect)"`
	SessionName                     string `help:"Override session name (default: auto-detect from branch/directory)"`
	TmuxStatusPosition              string `help:"Tmux status bar position (top or bottom)" default:"bottom" enum:"top,bottom"`
	Worktree                        string `help:"Override worktree path (default: current directory)"`
}

// Run executes the attach command
func (a *AttachCmd) Run(tmuxClient tmux.Client, cli *CLI) error {
	logging.Logger.Info("Attach command started")

	// Apply TmuxStatusPosition setting with proper precedence
	if a.TmuxStatusPosition == tmux.DefaultStatusPosition {
		if _, hasEnv := os.LookupEnv("ROCHA_TMUX_STATUS_POSITION"); !hasEnv {
			if cli.settings != nil && cli.settings.TmuxStatusPosition != "" {
				a.TmuxStatusPosition = cli.settings.TmuxStatusPosition
			}
		}
	}

	// Open database
	store, err := storage.NewStore(paths.GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

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
	sessionName = tmux.SanitizeSessionName(sessionName)

	if sessionName == "" {
		return fmt.Errorf("could not determine session name (no branch, no directory name)")
	}

	logging.Logger.Info("Session parameters determined",
		"session_name", sessionName,
		"repo_path", repoPath,
		"branch_name", branchName,
		"worktree_path", worktreePath)

	// Step 3.5: Check for duplicate sessions with same branch or worktree
	st, err := store.Load(context.Background(), false)
	if err != nil {
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
	if !tmuxClient.Exists(sessionName) {
		logging.Logger.Info("Session does not exist, creating", "name", sessionName)
		// Note: attach command doesn't support claudeDir parameter, use empty string
		_, err := tmuxClient.Create(sessionName, worktreePath, "", a.TmuxStatusPosition)
		if err != nil {
			return fmt.Errorf("failed to create tmux session: %w", err)
		}
		fmt.Printf("Created new session '%s'\n", sessionName)
	} else {
		logging.Logger.Info("Session already exists", "name", sessionName)
		fmt.Printf("Attaching to existing session '%s'\n", sessionName)
	}

	// Step 5: Update database
	// Reload state in case it changed
	st, err = store.Load(context.Background(), false)
	if err != nil {
		logging.Logger.Warn("Failed to load state", "error", err)
	}
	if st == nil {
		st = &storage.SessionState{Sessions: make(map[string]storage.SessionInfo)}
	}

	// Create or update session info - preserve DisplayName and State if session exists
	sessionInfo, exists := st.Sessions[sessionName]
	if exists {
		// Session exists - preserve State and DisplayName, update timestamp and git metadata
		sessionInfo.LastUpdated = time.Now().UTC()

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
		// New session - create with "idle" state (ready for user input)
		// Generate new execution ID for this session
		executionID := uuid.New().String()
		logging.Logger.Info("Creating new session with execution ID", "execution_id", executionID)

		sessionInfo = storage.SessionInfo{
			AllowDangerouslySkipPermissions: a.AllowDangerouslySkipPermissions,
			BranchName:                      branchName,
			DisplayName:                     sessionName,
			ExecutionID:                     executionID,
			LastUpdated:                     time.Now().UTC(),
			Name:                            sessionName,
			RepoInfo:                        repoInfo,
			RepoPath:                        repoPath,
			State:                           state.StateIdle,
			WorktreePath:                    worktreePath,
		}
	}

	st.Sessions[sessionName] = sessionInfo

	if err := store.Save(context.Background(), st); err != nil {
		logging.Logger.Error("Failed to save state", "error", err)
		// Continue anyway - session is created
	}

	// Step 6: Inform user that session is ready
	logging.Logger.Info("Session registered successfully", "name", sessionName)

	fmt.Printf("\nSession '%s' is ready!\n", sessionName)
	fmt.Printf("Start 'rocha' to view and attach to your sessions.\n")

	return nil
}

