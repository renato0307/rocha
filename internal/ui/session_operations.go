package ui

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/services"
)

// SessionOperations handles session lifecycle operations.
// Responsible for kill, archive, attach, and shell session management.
type SessionOperations struct {
	errorManager       *ErrorManager
	sessionService     *services.SessionService
	shellService       *services.ShellService
	tmuxStatusPosition string
}

// NewSessionOperations creates a new SessionOperations component.
func NewSessionOperations(
	errorManager *ErrorManager,
	tmuxStatusPosition string,
	sessionService *services.SessionService,
	shellService *services.ShellService,
) *SessionOperations {
	return &SessionOperations{
		errorManager:       errorManager,
		sessionService:     sessionService,
		shellService:       shellService,
		tmuxStatusPosition: tmuxStatusPosition,
	}
}

// AttachToSession suspends Bubble Tea, attaches to a tmux session via the abstraction layer,
// and returns a detachedMsg when the user detaches.
func (so *SessionOperations) AttachToSession(sessionName string) tea.Cmd {
	logging.Logger.Info("Attaching to session via abstraction layer", "name", sessionName)

	cmd := so.shellService.GetAttachCommand(sessionName)

	logging.Logger.Debug("Executing tmux attach command",
		"command", cmd.Path,
		"args", cmd.Args,
		"session_name", sessionName)

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
func (so *SessionOperations) GetOrCreateShellSession(
	session *ports.TmuxSession,
	sessionState *domain.SessionCollection,
) string {
	shellName, err := so.shellService.GetOrCreateShellSession(
		context.Background(),
		session.Name,
		so.tmuxStatusPosition,
	)
	if err != nil {
		so.errorManager.SetError(err)
		return ""
	}

	// Reload session state to get updated shell info
	newState, err := so.sessionService.LoadState(context.Background(), false)
	if err != nil {
		logging.Logger.Warn("Failed to reload state after shell creation", "error", err)
	} else {
		*sessionState = *newState
	}

	return shellName
}

// KillSession kills a session and removes it from state.
// Updates sessionState and sessionList, returns tea.Cmd.
func (so *SessionOperations) KillSession(
	session *ports.TmuxSession,
	sessionState *domain.SessionCollection,
	sessionList *SessionList,
) tea.Cmd {
	logging.Logger.Info("Killing session", "name", session.Name)

	if err := so.sessionService.KillSession(context.Background(), session.Name); err != nil {
		logging.Logger.Error("Failed to kill session", "error", err)
	}

	// Reload session state
	newState, err := so.sessionService.LoadState(context.Background(), false)
	if err != nil {
		log.Printf("Warning: failed to load state: %v", err)
	} else {
		*sessionState = *newState
	}

	refreshCmd := sessionList.RefreshFromState()
	return tea.Batch(refreshCmd, sessionList.Init())
}

// ArchiveSession archives a session and optionally removes its worktree.
// Updates sessionState and sessionList, returns tea.Cmd.
func (so *SessionOperations) ArchiveSession(
	session *ports.TmuxSession,
	removeWorktree bool,
	sessionState *domain.SessionCollection,
	sessionList *SessionList,
) tea.Cmd {
	logging.Logger.Info("Archiving session", "name", session.Name, "removeWorktree", removeWorktree)

	if err := so.sessionService.ArchiveSession(context.Background(), session.Name, removeWorktree); err != nil {
		so.errorManager.SetError(fmt.Errorf("failed to archive session: %w", err))
		return tea.Batch(sessionList.Init(), so.errorManager.ClearAfterDelay())
	}

	// Reload session state
	newState, err := so.sessionService.LoadState(context.Background(), false)
	if err != nil {
		so.errorManager.SetError(fmt.Errorf("failed to refresh sessions: %w", err))
		refreshCmd := sessionList.RefreshFromState()
		return tea.Batch(refreshCmd, sessionList.Init(), so.errorManager.ClearAfterDelay())
	}
	*sessionState = *newState

	refreshCmd := sessionList.RefreshFromState()
	return tea.Batch(refreshCmd, sessionList.Init())
}
