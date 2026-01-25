package ui

import (
	"context"
	"fmt"
	"log"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/storage"
	"rocha/tmux"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// SessionOperations handles session lifecycle operations.
// Responsible for kill, archive, attach, and shell session management.
type SessionOperations struct {
	errorManager       *ErrorManager
	store              *storage.Store
	tmuxClient         tmux.Client
	tmuxStatusPosition string
}

// NewSessionOperations creates a new SessionOperations component.
func NewSessionOperations(
	errorManager *ErrorManager,
	store *storage.Store,
	tmuxClient tmux.Client,
	tmuxStatusPosition string,
) *SessionOperations {
	return &SessionOperations{
		errorManager:       errorManager,
		store:              store,
		tmuxClient:         tmuxClient,
		tmuxStatusPosition: tmuxStatusPosition,
	}
}

// AttachToSession suspends Bubble Tea, attaches to a tmux session via the abstraction layer,
// and returns a detachedMsg when the user detaches.
func (so *SessionOperations) AttachToSession(sessionName string) tea.Cmd {
	logging.Logger.Info("Attaching to session via abstraction layer", "name", sessionName)

	// Get the attach command from the abstraction layer
	cmd := so.tmuxClient.GetAttachCommand(sessionName)

	// Log the exact command being executed
	logging.Logger.Debug("Executing tmux attach command",
		"command", cmd.Path,
		"args", cmd.Args,
		"session_name", sessionName)

	// Use tea.ExecProcess to suspend Bubble Tea and run the command
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			logging.Logger.Error("Failed to attach to session", "error", err, "name", sessionName)
			return err
		}
		logging.Logger.Info("Detached from session", "name", sessionName)
		return detachedMsg{}
	})
}

// GetOrCreateShellSession returns shell session name, creating if needed.
// Returns empty string on error (error stored in errorManager).
// Updates sessionState if shell session is created.
func (so *SessionOperations) GetOrCreateShellSession(
	session *tmux.Session,
	sessionState *storage.SessionState,
) string {
	sessionInfo, ok := sessionState.Sessions[session.Name]
	if !ok {
		so.errorManager.SetError(fmt.Errorf("session info not found: %s", session.Name))
		return ""
	}

	// Check if shell session already exists
	if sessionInfo.ShellSession != nil {
		// Check if tmux session exists
		if so.tmuxClient.Exists(sessionInfo.ShellSession.Name) {
			return sessionInfo.ShellSession.Name
		}
	}

	// Create shell session name
	shellSessionName := fmt.Sprintf("%s-shell", session.Name)

	// Determine working directory
	workingDir := sessionInfo.WorktreePath
	if workingDir == "" {
		workingDir = sessionInfo.RepoPath
	}

	// Create shell session in tmux
	_, err := so.tmuxClient.CreateShellSession(shellSessionName, workingDir, so.tmuxStatusPosition)
	if err != nil {
		so.errorManager.SetError(fmt.Errorf("failed to create shell session: %w", err))
		return ""
	}

	// Create nested SessionInfo for shell session
	shellInfo := storage.SessionInfo{
		BranchName:   sessionInfo.BranchName,
		DisplayName:  "", // No display name for shell sessions
		ExecutionID:  sessionInfo.ExecutionID,
		LastUpdated:  time.Now().UTC(),
		Name:         shellSessionName,
		RepoInfo:     sessionInfo.RepoInfo,
		RepoPath:     sessionInfo.RepoPath,
		ShellSession: nil, // Shell sessions don't have their own shells
		State:        state.StateIdle,
		WorktreePath: sessionInfo.WorktreePath,
	}

	// Update parent session with nested shell info
	sessionInfo.ShellSession = &shellInfo
	sessionState.Sessions[session.Name] = sessionInfo

	if err := so.store.Save(context.Background(), sessionState); err != nil {
		logging.Logger.Warn("Failed to save shell session to state", "error", err)
	}

	return shellSessionName
}

// KillSession kills a session and removes it from state.
// Updates sessionState and sessionList, returns tea.Cmd.
func (so *SessionOperations) KillSession(
	session *tmux.Session,
	sessionState *storage.SessionState,
	sessionList *SessionList,
) tea.Cmd {
	logging.Logger.Info("Killing session", "name", session.Name)

	// Get session info to check for shell session
	sessionInfo, hasInfo := sessionState.Sessions[session.Name]

	// Kill shell session if it exists
	if hasInfo && sessionInfo.ShellSession != nil {
		logging.Logger.Info("Killing shell session", "name", sessionInfo.ShellSession.Name)
		if err := so.tmuxClient.Kill(sessionInfo.ShellSession.Name); err != nil {
			logging.Logger.Warn("Failed to kill shell session", "error", err)
		}
	}

	// Kill main Claude session (continue with deletion even if kill fails - session may already be exited)
	if err := so.tmuxClient.Kill(session.Name); err != nil {
		logging.Logger.Warn("Failed to kill session (may already be exited)", "name", session.Name, "error", err)
	}

	// Check if session has worktree and remove it from state
	if hasInfo && sessionInfo.WorktreePath != "" {
		logging.Logger.Info("Session had worktree", "path", sessionInfo.WorktreePath)
	}

	// Remove session from state
	st, err := so.store.Load(context.Background(), false)
	if err != nil {
		log.Printf("Warning: failed to load state: %v", err)
	} else {
		delete(st.Sessions, session.Name)
		if err := so.store.Save(context.Background(), st); err != nil {
			log.Printf("Warning: failed to save state: %v", err)
		}
		// Update the passed sessionState pointer
		*sessionState = *st
	}

	// Refresh session list component
	sessionList.RefreshFromState()
	return sessionList.Init()
}

// ArchiveSession archives a session and optionally removes its worktree.
// Updates sessionState and sessionList, returns tea.Cmd.
func (so *SessionOperations) ArchiveSession(
	session *tmux.Session,
	removeWorktree bool,
	sessionState *storage.SessionState,
	sessionList *SessionList,
) tea.Cmd {
	logging.Logger.Info("Archiving session", "name", session.Name, "removeWorktree", removeWorktree)

	// Get session info
	sessionInfo, hasInfo := sessionState.Sessions[session.Name]

	// Remove worktree if requested
	if removeWorktree && hasInfo && sessionInfo.WorktreePath != "" {
		logging.Logger.Info("Removing worktree", "path", sessionInfo.WorktreePath, "repo", sessionInfo.RepoPath)
		if err := git.RemoveWorktree(sessionInfo.RepoPath, sessionInfo.WorktreePath); err != nil {
			so.errorManager.SetError(fmt.Errorf("failed to remove worktree, continuing with archive: %w", err))
			logging.Logger.Error("Failed to remove worktree", "error", err, "path", sessionInfo.WorktreePath)
		} else {
			logging.Logger.Info("Worktree removed successfully", "path", sessionInfo.WorktreePath)
		}
	}

	// Toggle archive state
	if err := so.store.ToggleArchive(context.Background(), session.Name); err != nil {
		so.errorManager.SetError(fmt.Errorf("failed to archive session: %w", err))
		return tea.Batch(sessionList.Init(), so.errorManager.ClearAfterDelay())
	}

	// Reload session state (showArchived=false, so archived session will disappear)
	newState, err := so.store.Load(context.Background(), false)
	if err != nil {
		so.errorManager.SetError(fmt.Errorf("failed to refresh sessions: %w", err))
		sessionList.RefreshFromState()
		return tea.Batch(sessionList.Init(), so.errorManager.ClearAfterDelay())
	}
	// Update the passed sessionState pointer
	*sessionState = *newState

	// Refresh UI
	sessionList.RefreshFromState()
	return sessionList.Init()
}
