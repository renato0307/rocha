package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"rocha/domain"
	"rocha/logging"
	"rocha/paths"
	"rocha/ports"
	"rocha/adapters/tmux"
)

// SessionService handles session lifecycle operations
type SessionService struct {
	claudeDirResolver ClaudeDirResolver
	gitRepo           ports.GitRepository
	sessionRepo       ports.SessionRepository
	tmuxClient        ports.TmuxSessionLifecycle
}

// NewSessionService creates a new SessionService
func NewSessionService(
	sessionRepo ports.SessionRepository,
	gitRepo ports.GitRepository,
	tmuxClient ports.TmuxSessionLifecycle,
	claudeDirResolver ClaudeDirResolver,
) *SessionService {
	return &SessionService{
		claudeDirResolver: claudeDirResolver,
		gitRepo:           gitRepo,
		sessionRepo:       sessionRepo,
		tmuxClient:        tmuxClient,
	}
}

// CreateSession orchestrates session creation with optional worktree
func (s *SessionService) CreateSession(
	ctx context.Context,
	params CreateSessionParams,
) (*CreateSessionResult, error) {
	sessionName := params.SessionName
	branchName := params.BranchNameOverride
	repoSource := params.RepoSource

	// Automatically create worktree if repo is provided
	createWorktree := repoSource != ""

	logging.Logger.Info("Creating session",
		"name", sessionName,
		"create_worktree", createWorktree,
		"branch", branchName,
		"repo_source", repoSource)

	// Generate tmux-compatible name
	tmuxName := tmux.SanitizeSessionName(sessionName)

	var claudeDir string
	var repoInfo string
	var repoPath string
	var worktreePath string
	var sourceBranch string

	// 1. Determine repository source
	if repoSource != "" {
		logging.Logger.Info("Using user-provided repository source", "source", repoSource)

		worktreeBase := paths.GetWorktreePath()

		localPath, src, err := s.gitRepo.GetOrCloneRepository(repoSource, worktreeBase)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository: %w", err)
		}
		repoPath = localPath
		if src.Owner != "" && src.Repo != "" {
			repoInfo = fmt.Sprintf("%s/%s", src.Owner, src.Repo)
		}
		sourceBranch = src.Branch
		if sourceBranch != "" {
			logging.Logger.Info("Branch specified in URL", "branch", sourceBranch)
		}
		logging.Logger.Info("Repository ready", "path", repoPath, "repo_info", repoInfo)
	} else {
		logging.Logger.Info("Using current directory as repository source")
		cwd, _ := os.Getwd()
		isGit, repo := s.gitRepo.IsGitRepo(cwd)
		if isGit {
			repoPath = repo
			repoInfo = s.gitRepo.GetRepoInfo(repo)
			logging.Logger.Info("Extracted repo info from current directory", "repo_info", repoInfo)

			// Get remote URL from git repo
			if remoteURL := s.gitRepo.GetRemoteURL(repo); remoteURL != "" {
				repoSource = remoteURL
				logging.Logger.Info("Fetched remote URL from git repo", "remote_url", remoteURL)
			}
		}
	}

	// 2. Resolve ClaudeDir
	claudeDir = s.claudeDirResolver.Resolve(repoInfo, params.ClaudeDirOverride)
	logging.Logger.Info("Resolved ClaudeDir", "path", claudeDir)

	// If ClaudeDir is system default, don't set custom override
	homeDir, err := os.UserHomeDir()
	if err == nil {
		systemDefault := filepath.Join(homeDir, ".claude")
		if claudeDir == systemDefault {
			logging.Logger.Info("ClaudeDir is system default, not setting custom override", "default", systemDefault)
			claudeDir = ""
		}
	}

	// 3. Create worktree if requested
	if createWorktree && repoPath != "" {
		if branchName == "" {
			var err error
			branchName, err = s.gitRepo.SanitizeBranchName(sessionName)
			if err != nil {
				return nil, fmt.Errorf("failed to generate branch name from session name: %w", err)
			}
			logging.Logger.Info("Auto-generated branch name from session name", "branch", branchName)
		}

		worktreeBase := paths.GetWorktreePath()
		worktreePath = s.gitRepo.BuildWorktreePath(worktreeBase, repoInfo, tmuxName)
		logging.Logger.Info("Creating worktree", "path", worktreePath, "branch", branchName)

		if err := s.gitRepo.CreateWorktree(repoPath, worktreePath, branchName); err != nil {
			return nil, fmt.Errorf("failed to create worktree: %w", err)
		}
	} else if createWorktree && repoPath == "" {
		logging.Logger.Warn("Cannot create worktree: not in a git repository")
	} else if !createWorktree && sourceBranch != "" {
		branchName = sourceBranch
		logging.Logger.Info("Using branch from URL (no worktree)", "branch", branchName)
	}

	// 4. Create tmux session
	tmuxSession, err := s.tmuxClient.CreateSession(tmuxName, worktreePath, claudeDir, params.TmuxStatusPosition)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// 5. Build domain session and save
	executionID := os.Getenv("ROCHA_EXECUTION_ID")

	session := domain.Session{
		AllowDangerouslySkipPermissions: params.AllowDangerouslySkipPermissions,
		BranchName:                      branchName,
		ClaudeDir:                       claudeDir,
		DisplayName:                     sessionName,
		ExecutionID:                     executionID,
		LastUpdated:                     time.Now().UTC(),
		Name:                            tmuxName,
		RepoInfo:                        repoInfo,
		RepoPath:                        repoPath,
		RepoSource:                      repoSource,
		State:                           domain.StateWaiting,
		WorktreePath:                    worktreePath,
	}

	if err := s.sessionRepo.Add(ctx, session); err != nil {
		logging.Logger.Error("Failed to add session to database", "error", err)
		return nil, err
	}

	logging.Logger.Info("Session created successfully",
		"name", tmuxSession.Name,
		"claude_dir", claudeDir,
		"repo_source", repoSource)

	return &CreateSessionResult{
		Session:      &session,
		WorktreePath: worktreePath,
	}, nil
}

// KillSession kills a session and removes it from state
func (s *SessionService) KillSession(
	ctx context.Context,
	sessionName string,
) error {
	logging.Logger.Info("Killing session", "name", sessionName)

	// Get session info to check for shell session
	session, err := s.sessionRepo.Get(ctx, sessionName)
	if err != nil {
		logging.Logger.Warn("Could not get session info", "name", sessionName, "error", err)
	}

	// Kill shell session if it exists
	if session != nil && session.ShellSession != nil {
		logging.Logger.Info("Killing shell session", "name", session.ShellSession.Name)
		if err := s.tmuxClient.KillSession(session.ShellSession.Name); err != nil {
			logging.Logger.Warn("Failed to kill shell session", "error", err)
		}
		// Delete shell session from DB
		if err := s.sessionRepo.Delete(ctx, session.ShellSession.Name); err != nil {
			logging.Logger.Warn("Failed to delete shell session from DB", "error", err)
		}
	}

	// Kill main Claude session
	if err := s.tmuxClient.KillSession(sessionName); err != nil {
		logging.Logger.Warn("Failed to kill session (may already be exited)", "name", sessionName, "error", err)
	}

	// Remove session from database
	if err := s.sessionRepo.Delete(ctx, sessionName); err != nil {
		logging.Logger.Warn("Failed to delete session from DB", "error", err)
	}

	logging.Logger.Info("Session killed", "name", sessionName)
	return nil
}

// DeleteSessionOptions configures session deletion behavior
type DeleteSessionOptions struct {
	KillTmux       bool // Kill tmux sessions before deleting
	RemoveWorktree bool // Remove worktree from filesystem
}

// DeleteSession removes a session from database with optional tmux kill and worktree removal
func (s *SessionService) DeleteSession(
	ctx context.Context,
	sessionName string,
	opts DeleteSessionOptions,
) error {
	logging.Logger.Info("Deleting session",
		"session", sessionName,
		"killTmux", opts.KillTmux,
		"removeWorktree", opts.RemoveWorktree)

	// Get session info before deleting (to get worktree path and shell session)
	session, err := s.sessionRepo.Get(ctx, sessionName)
	if err != nil {
		logging.Logger.Error("Failed to get session for deletion", "session", sessionName, "error", err)
		return fmt.Errorf("failed to get session %s: %w", sessionName, err)
	}

	// Kill tmux sessions if requested
	if opts.KillTmux {
		logging.Logger.Debug("Killing tmux sessions", "session", sessionName)
		// Kill shell session if exists
		if session.ShellSession != nil {
			logging.Logger.Debug("Killing shell session", "session", session.ShellSession.Name)
			if err := s.tmuxClient.KillSession(session.ShellSession.Name); err != nil {
				logging.Logger.Warn("Failed to kill shell session", "session", session.ShellSession.Name, "error", err)
				fmt.Printf("⚠ Warning: Failed to kill shell session %s: %v\n", session.ShellSession.Name, err)
			}
		}

		// Kill main session
		if err := s.tmuxClient.KillSession(sessionName); err != nil {
			logging.Logger.Warn("Failed to kill tmux session", "session", sessionName, "error", err)
			fmt.Printf("⚠ Warning: Failed to kill tmux session %s: %v\n", sessionName, err)
		}
	}

	// Delete from database (cascade deletes extension tables)
	logging.Logger.Debug("Deleting session from database", "session", sessionName)
	if err := s.sessionRepo.Delete(ctx, sessionName); err != nil {
		logging.Logger.Error("Failed to delete session from database", "session", sessionName, "error", err)
		return fmt.Errorf("failed to delete session %s from database: %w", sessionName, err)
	}

	// Remove worktree if requested and exists
	if opts.RemoveWorktree && session.WorktreePath != "" && session.RepoPath != "" {
		logging.Logger.Info("Removing worktree", "session", sessionName, "path", session.WorktreePath)
		if err := s.gitRepo.RemoveWorktree(session.RepoPath, session.WorktreePath); err != nil {
			logging.Logger.Warn("Failed to remove worktree", "session", sessionName, "path", session.WorktreePath, "error", err)
			fmt.Printf("⚠ Warning: Failed to remove worktree for %s: %v\n", sessionName, err)
		} else {
			logging.Logger.Info("Worktree removed successfully", "session", sessionName)
		}
	}

	logging.Logger.Info("Session deleted successfully", "session", sessionName)
	return nil
}

// ArchiveSession archives a session and optionally removes its worktree
func (s *SessionService) ArchiveSession(
	ctx context.Context,
	sessionName string,
	removeWorktree bool,
) error {
	logging.Logger.Info("Archiving session", "name", sessionName, "removeWorktree", removeWorktree)

	// Get session info
	session, err := s.sessionRepo.Get(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session info: %w", err)
	}

	// Remove worktree if requested
	if removeWorktree && session.WorktreePath != "" {
		logging.Logger.Info("Removing worktree", "path", session.WorktreePath, "repo", session.RepoPath)
		if err := s.gitRepo.RemoveWorktree(session.RepoPath, session.WorktreePath); err != nil {
			logging.Logger.Error("Failed to remove worktree", "error", err, "path", session.WorktreePath)
			// Continue with archive even if worktree removal fails
		} else {
			logging.Logger.Info("Worktree removed successfully", "path", session.WorktreePath)
		}
	}

	// Toggle archive state
	if err := s.sessionRepo.ToggleArchive(ctx, sessionName); err != nil {
		return fmt.Errorf("failed to archive session: %w", err)
	}

	logging.Logger.Info("Session archived", "name", sessionName)
	return nil
}

