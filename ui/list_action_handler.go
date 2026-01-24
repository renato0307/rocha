package ui

import (
	"context"
	"fmt"
	"os"
	"rocha/editor"
	"rocha/git"
	"rocha/logging"
	"rocha/storage"
	"rocha/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// ActionType indicates what action Model should take
type ActionType int

const (
	ActionNone ActionType = iota
	ActionAttachSession
	ActionAttachShellSession
	ActionShowRenameDialog
	ActionShowStatusDialog
	ActionShowCommentDialog
	ActionShowSendTextDialog
	ActionShowHelpDialog
	ActionShowNewSessionDialog
	ActionShowNewSessionFromDialog
	ActionShowKillWorktreeDialog
	ActionShowArchiveWorktreeDialog
	ActionKillSession
	ActionArchiveSession
	ActionOpenEditor
	ActionToggleFlag
)

// ActionResult tells Model what action to take
type ActionResult struct {
	ActionType      ActionType
	Cmd             tea.Cmd
	SessionName     string
	SessionToKill   *tmux.Session
	SessionToArchive *tmux.Session
	WorktreePath    string

	// For dialog creation
	DialogTitle     string
	DialogContent   tea.Model
	NewState        uiState

	// For new session dialog
	DefaultRepoSource string
}

// ListActionHandler processes SessionList action requests
type ListActionHandler struct {
	allowDangerouslySkipPermissionsDefault bool
	devMode                                bool
	editor                                 string
	errorManager                           *ErrorManager
	sessionList                            *SessionList
	sessionOps                             *SessionOperations
	sessionState                           *storage.SessionState
	statusConfig                           *StatusConfig
	store                                  *storage.Store
	tmuxClient                             tmux.Client
	tmuxStatusPosition                     string
}

// NewListActionHandler creates a new ListActionHandler
func NewListActionHandler(
	sessionList *SessionList,
	sessionState *storage.SessionState,
	store *storage.Store,
	editor string,
	statusConfig *StatusConfig,
	errorManager *ErrorManager,
	sessionOps *SessionOperations,
	tmuxClient tmux.Client,
	tmuxStatusPosition string,
	devMode bool,
	allowDangerouslySkipPermissionsDefault bool,
) *ListActionHandler {
	return &ListActionHandler{
		allowDangerouslySkipPermissionsDefault: allowDangerouslySkipPermissionsDefault,
		devMode:                                devMode,
		editor:                                 editor,
		errorManager:                           errorManager,
		sessionList:                            sessionList,
		sessionOps:                             sessionOps,
		sessionState:                           sessionState,
		statusConfig:                           statusConfig,
		store:                                  store,
		tmuxClient:                             tmuxClient,
		tmuxStatusPosition:                     tmuxStatusPosition,
	}
}

// ProcessActions checks SessionList for action requests and returns what Model should do
func (lah *ListActionHandler) ProcessActions() ActionResult {
	// Check for attach session
	if lah.sessionList.SelectedSession != nil {
		session := lah.sessionList.SelectedSession
		lah.sessionList.SelectedSession = nil
		return ActionResult{
			ActionType: ActionAttachSession,
			Cmd:        lah.sessionOps.AttachToSession(session.Name),
		}
	}

	// Check for shell session
	if lah.sessionList.SelectedShellSession != nil {
		session := lah.sessionList.SelectedShellSession
		lah.sessionList.SelectedShellSession = nil
		shellSessionName := lah.sessionOps.GetOrCreateShellSession(session, lah.sessionState)
		if shellSessionName != "" {
			return ActionResult{
				ActionType: ActionAttachShellSession,
				Cmd:        lah.sessionOps.AttachToSession(shellSessionName),
			}
		}
		return ActionResult{ActionType: ActionNone}
	}

	// Check for kill session
	if lah.sessionList.SessionToKill != nil {
		session := lah.sessionList.SessionToKill
		lah.sessionList.SessionToKill = nil

		// Check if session has worktree
		if sessionInfo, ok := lah.sessionState.Sessions[session.Name]; ok && sessionInfo.WorktreePath != "" {
			return ActionResult{
				ActionType:    ActionShowKillWorktreeDialog,
				SessionToKill: session,
				WorktreePath:  sessionInfo.WorktreePath,
			}
		}
		return ActionResult{
			ActionType: ActionKillSession,
			Cmd:        lah.sessionOps.KillSession(session, lah.sessionState, lah.sessionList),
		}
	}

	// Check for rename session
	if lah.sessionList.SessionToRename != nil {
		session := lah.sessionList.SessionToRename
		lah.sessionList.SessionToRename = nil

		// Get current display name
		currentDisplayName := session.Name
		if sessionInfo, ok := lah.sessionState.Sessions[session.Name]; ok && sessionInfo.DisplayName != "" {
			currentDisplayName = sessionInfo.DisplayName
		}

		contentForm := NewSessionRenameForm(lah.tmuxClient, lah.store, lah.sessionState, session.Name, currentDisplayName)
		return ActionResult{
			ActionType:    ActionShowRenameDialog,
			DialogTitle:   "Rename Session",
			DialogContent: contentForm,
			NewState:      stateRenamingSession,
		}
	}

	// Check for set status
	if lah.sessionList.SessionToSetStatus != nil {
		session := lah.sessionList.SessionToSetStatus
		lah.sessionList.SessionToSetStatus = nil

		// Get current status
		var currentStatus *string
		if sessionInfo, ok := lah.sessionState.Sessions[session.Name]; ok {
			currentStatus = sessionInfo.Status
		}

		contentForm := NewSessionStatusForm(lah.store, session.Name, currentStatus, lah.statusConfig)
		return ActionResult{
			ActionType:    ActionShowStatusDialog,
			DialogTitle:   "Set Status",
			DialogContent: contentForm,
			NewState:      stateSettingStatus,
		}
	}

	// Check for comment session
	if lah.sessionList.SessionToComment != nil {
		session := lah.sessionList.SessionToComment
		lah.sessionList.SessionToComment = nil

		// Get current comment
		currentComment := ""
		if sessionInfo, ok := lah.sessionState.Sessions[session.Name]; ok {
			currentComment = sessionInfo.Comment
		}

		contentForm := NewSessionCommentForm(lah.store, session.Name, currentComment)
		return ActionResult{
			ActionType:    ActionShowCommentDialog,
			DialogTitle:   "Edit Session Comment",
			DialogContent: contentForm,
			NewState:      stateCommentingSession,
		}
	}

	// Check for send text
	if lah.sessionList.SessionToSendText != nil {
		session := lah.sessionList.SessionToSendText
		lah.sessionList.SessionToSendText = nil

		contentForm := NewSendTextForm(lah.tmuxClient, session.Name)
		return ActionResult{
			ActionType:    ActionShowSendTextDialog,
			DialogTitle:   "Send Text to Claude",
			DialogContent: contentForm,
			NewState:      stateSendingText,
		}
	}

	// Check for open editor
	if lah.sessionList.SessionToOpenEditor != nil {
		session := lah.sessionList.SessionToOpenEditor
		lah.sessionList.SessionToOpenEditor = nil

		sessionInfo, exists := lah.sessionState.Sessions[session.Name]
		if !exists || sessionInfo.WorktreePath == "" {
			lah.errorManager.SetError(fmt.Errorf("no worktree associated with session '%s'", session.Name))
			return ActionResult{
				ActionType: ActionNone,
				Cmd:        tea.Batch(lah.sessionList.Init(), lah.errorManager.ClearAfterDelay()),
			}
		}

		if err := editor.OpenInEditor(sessionInfo.WorktreePath, lah.editor); err != nil {
			lah.errorManager.SetError(fmt.Errorf("failed to open editor: %w", err))
			return ActionResult{
				ActionType: ActionNone,
				Cmd:        tea.Batch(lah.sessionList.Init(), lah.errorManager.ClearAfterDelay()),
			}
		}

		return ActionResult{
			ActionType: ActionOpenEditor,
			Cmd:        lah.sessionList.Init(),
		}
	}

	// Check for toggle flag
	if lah.sessionList.SessionToToggleFlag != nil {
		session := lah.sessionList.SessionToToggleFlag
		lah.sessionList.SessionToToggleFlag = nil

		if err := lah.store.ToggleFlag(context.Background(), session.Name); err != nil {
			lah.errorManager.SetError(fmt.Errorf("failed to toggle flag: %w", err))
			return ActionResult{
				ActionType: ActionNone,
				Cmd:        tea.Batch(lah.sessionList.Init(), lah.errorManager.ClearAfterDelay()),
			}
		}

		// Reload session state
		sessionState, err := lah.store.Load(context.Background(), false)
		if err != nil {
			lah.errorManager.SetError(fmt.Errorf("failed to refresh sessions: %w", err))
			lah.sessionList.RefreshFromState()
			return ActionResult{
				ActionType: ActionNone,
				Cmd:        tea.Batch(lah.sessionList.Init(), lah.errorManager.ClearAfterDelay()),
			}
		}
		*lah.sessionState = *sessionState

		// Refresh UI
		lah.sessionList.RefreshFromState()
		return ActionResult{
			ActionType: ActionToggleFlag,
			Cmd:        lah.sessionList.Init(),
		}
	}

	// Check for archive session
	if lah.sessionList.SessionToArchive != nil {
		session := lah.sessionList.SessionToArchive
		lah.sessionList.SessionToArchive = nil

		// Check if session has worktree
		if sessionInfo, ok := lah.sessionState.Sessions[session.Name]; ok && sessionInfo.WorktreePath != "" {
			return ActionResult{
				ActionType:       ActionShowArchiveWorktreeDialog,
				SessionToArchive: session,
				WorktreePath:     sessionInfo.WorktreePath,
			}
		}
		return ActionResult{
			ActionType: ActionArchiveSession,
			Cmd:        lah.sessionOps.ArchiveSession(session, false, lah.sessionState, lah.sessionList),
		}
	}

	// Check for help request
	if lah.sessionList.RequestHelp {
		lah.sessionList.RequestHelp = false
		return ActionResult{
			ActionType: ActionShowHelpDialog,
			NewState:   stateHelp,
		}
	}

	// Check for new session
	if lah.sessionList.RequestNewSession {
		lah.sessionList.RequestNewSession = false

		// Pre-fill repo field if starting in a git folder
		defaultRepoSource := ""
		cwd, _ := os.Getwd()
		if isGit, repoPath := git.IsGitRepo(cwd); isGit {
			if remoteURL := git.GetRemoteURL(repoPath); remoteURL != "" {
				defaultRepoSource = remoteURL
				logging.Logger.Info("Pre-filling repository field with remote URL", "remote_url", remoteURL)
			} else {
				logging.Logger.Warn("Git repository has no remote configured, leaving repo field empty")
			}
		}

		return ActionResult{
			ActionType:        ActionShowNewSessionDialog,
			DefaultRepoSource: defaultRepoSource,
			NewState:          stateCreatingSession,
		}
	}

	// Check for new session from template
	if lah.sessionList.RequestNewSessionFrom {
		lah.sessionList.RequestNewSessionFrom = false
		// Get the repo source from the selected session
		var repoSource string
		if lah.sessionList.SessionForTemplate != nil {
			if sessionInfo, exists := lah.sessionState.Sessions[lah.sessionList.SessionForTemplate.Name]; exists {
				repoSource = sessionInfo.RepoSource

				// If RepoSource is empty but RepoPath exists, fetch remote URL and update DB
				if repoSource == "" && sessionInfo.RepoPath != "" {
					logging.Logger.Info("RepoSource empty, fetching remote URL from RepoPath", "repo_path", sessionInfo.RepoPath)
					if remoteURL := git.GetRemoteURL(sessionInfo.RepoPath); remoteURL != "" {
						repoSource = remoteURL
						logging.Logger.Info("Fetched remote URL, updating database", "remote_url", remoteURL)

						// Update the session in the database with the fetched RepoSource
						if err := lah.store.UpdateSessionRepoSource(context.Background(), lah.sessionList.SessionForTemplate.Name, remoteURL); err != nil {
							logging.Logger.Error("Failed to update RepoSource in database", "error", err)
						} else {
							// Also update in-memory state
							sessionInfo.RepoSource = remoteURL
							lah.sessionState.Sessions[lah.sessionList.SessionForTemplate.Name] = sessionInfo
						}
					}
				}

				logging.Logger.Info("Creating new session from template", "template_session", lah.sessionList.SessionForTemplate.Name, "repo_source", repoSource)
			}
		}
		lah.sessionList.SessionForTemplate = nil // Clear template reference

		return ActionResult{
			ActionType:        ActionShowNewSessionFromDialog,
			DefaultRepoSource: repoSource,
			NewState:          stateCreatingSession,
		}
	}

	// Check for test error
	if lah.sessionList.RequestTestError {
		lah.sessionList.RequestTestError = false
		lah.errorManager.SetError(fmt.Errorf("this is a very long test error message to verify that the error display truncation functionality works correctly and ensures that error text wraps properly across multiple lines and eventually gets truncated with ellipsis if it exceeds the maximum allowed length of three lines which should be enforced by the formatErrorForDisplay function"))
		return ActionResult{
			ActionType: ActionNone,
			Cmd:        tea.Batch(lah.sessionList.Init(), lah.errorManager.ClearAfterDelay()),
		}
	}

	return ActionResult{ActionType: ActionNone}
}
