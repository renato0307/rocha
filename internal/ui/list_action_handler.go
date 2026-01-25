package ui

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/services"
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
	ActionType       ActionType
	Cmd              tea.Cmd
	SessionName      string
	SessionToArchive *ports.TmuxSession
	SessionToKill    *ports.TmuxSession
	WorktreePath     string

	// For dialog creation
	DialogContent tea.Model
	DialogTitle   string
	NewState      uiState

	// For new session dialog
	DefaultRepoSource string
}

// ListActionHandler processes SessionList action requests
type ListActionHandler struct {
	allowDangerouslySkipPermissionsDefault bool
	devMode                                bool
	editor                                 string
	errorManager                           *ErrorManager
	gitService                             *services.GitService
	sessionList                            *SessionList
	sessionOps                             *SessionOperations
	sessionService                         *services.SessionService
	sessionState                           *domain.SessionCollection
	shellService                           *services.ShellService
	statusConfig                           *config.StatusConfig
	tmuxStatusPosition                     string
}

// NewListActionHandler creates a new ListActionHandler
func NewListActionHandler(
	sessionList *SessionList,
	sessionState *domain.SessionCollection,
	gitService *services.GitService,
	editor string,
	statusConfig *config.StatusConfig,
	errorManager *ErrorManager,
	sessionOps *SessionOperations,
	tmuxStatusPosition string,
	devMode bool,
	allowDangerouslySkipPermissionsDefault bool,
	sessionService *services.SessionService,
	shellService *services.ShellService,
) *ListActionHandler {
	return &ListActionHandler{
		allowDangerouslySkipPermissionsDefault: allowDangerouslySkipPermissionsDefault,
		devMode:                                devMode,
		editor:                                 editor,
		errorManager:                           errorManager,
		gitService:                             gitService,
		sessionList:                            sessionList,
		sessionOps:                             sessionOps,
		sessionService:                         sessionService,
		sessionState:                           sessionState,
		shellService:                           shellService,
		statusConfig:                           statusConfig,
		tmuxStatusPosition:                     tmuxStatusPosition,
	}
}

// getFreshSessionInfo loads fresh session info from the database to avoid stale state issues.
// Returns the Session and true if found, or zero value and false if not found.
func (lah *ListActionHandler) getFreshSessionInfo(sessionName string) (domain.Session, bool) {
	freshState, err := lah.sessionService.LoadState(context.Background(), false)
	if err != nil {
		logging.Logger.Error("Failed to load fresh state", "error", err)
		// Fall back to cached state
		freshState = lah.sessionState
	}
	sessionInfo, ok := freshState.Sessions[sessionName]
	return sessionInfo, ok
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

		// Use fresh state to avoid race condition with polling
		if sessionInfo, ok := lah.getFreshSessionInfo(session.Name); ok && sessionInfo.WorktreePath != "" {
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

		contentForm := NewSessionRenameForm(lah.sessionService, lah.sessionState, session.Name, currentDisplayName)
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

		contentForm := NewSessionStatusForm(lah.sessionService, session.Name, currentStatus, lah.statusConfig)
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

		contentForm := NewSessionCommentForm(lah.sessionService, session.Name, currentComment)
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

		contentForm := NewSendTextForm(lah.shellService, session.Name)
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

		if err := lah.shellService.OpenEditor(sessionInfo.WorktreePath, lah.editor); err != nil {
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

		if err := lah.sessionService.ToggleFlag(context.Background(), session.Name); err != nil {
			lah.errorManager.SetError(fmt.Errorf("failed to toggle flag: %w", err))
			return ActionResult{
				ActionType: ActionNone,
				Cmd:        tea.Batch(lah.sessionList.Init(), lah.errorManager.ClearAfterDelay()),
			}
		}

		// Reload session state
		newSessionState, err := lah.sessionService.LoadState(context.Background(), false)
		if err != nil {
			lah.errorManager.SetError(fmt.Errorf("failed to refresh sessions: %w", err))
			lah.sessionList.RefreshFromState()
			return ActionResult{
				ActionType: ActionNone,
				Cmd:        tea.Batch(lah.sessionList.Init(), lah.errorManager.ClearAfterDelay()),
			}
		}
		*lah.sessionState = *newSessionState

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

		// Use fresh state to avoid race condition with polling
		if sessionInfo, ok := lah.getFreshSessionInfo(session.Name); ok && sessionInfo.WorktreePath != "" {
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
		if isGit, repoPath := lah.gitService.IsGitRepo(cwd); isGit {
			if remoteURL := lah.gitService.GetRemoteURL(repoPath); remoteURL != "" {
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

				// Sanitize: remove branch suffix from URL (e.g., #feature-branch)
				// so the new session form shows only the base repository URL
				if repoSource != "" {
					if parsed, err := lah.gitService.ParseRepoSource(repoSource); err == nil {
						repoSource = parsed.Path
						logging.Logger.Debug("Sanitized repo source for template", "original", sessionInfo.RepoSource, "sanitized", repoSource)
					}
				}

				// If RepoSource is empty but RepoPath exists, fetch remote URL and update DB
				if repoSource == "" && sessionInfo.RepoPath != "" {
					logging.Logger.Info("RepoSource empty, fetching remote URL from RepoPath", "repo_path", sessionInfo.RepoPath)
					if remoteURL := lah.gitService.GetRemoteURL(sessionInfo.RepoPath); remoteURL != "" {
						repoSource = remoteURL
						logging.Logger.Info("Fetched remote URL, updating database", "remote_url", remoteURL)

						// Update the session in the database with the fetched RepoSource
						if err := lah.sessionService.UpdateRepoSource(context.Background(), lah.sessionList.SessionForTemplate.Name, remoteURL); err != nil {
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
