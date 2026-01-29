package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/services"
	"github.com/renato0307/rocha/internal/theme"
)

type uiState int

const (
	stateList uiState = iota
	stateCommandPalette
	stateCommentingSession
	stateConfirmingArchive
	stateConfirmingWorktreeRemoval
	stateCreatingSession
	stateHelp
	stateRenamingSession
	stateSendingText
	stateSettingStatus
)

type Model struct {
	allowDangerouslySkipPermissionsDefault bool                         // Default value from settings for new sessions
	commandPalette                         *CommandPalette              // Command palette overlay
	devMode                                bool                         // Development mode (shows version info in dialogs)
	editor                                 string                       // Editor to open sessions in
	errorManager                           *ErrorManager                // Error display and auto-clearing
	formRemoveWorktree                     *bool                        // Worktree removal decision (pointer to persist across updates)
	formRemoveWorktreeArchive              *bool                        // Worktree removal decision for archive (pointer to persist across updates)
	gitService                             *services.GitService         // Git operations service
	height                                 int
	helpScreen                             *Dialog                      // Help screen dialog
	keys                                   KeyMap                       // Keyboard shortcuts
	sendTextForm                           *Dialog                      // Send text to tmux dialog
	sessionCommentForm                     *Dialog                      // Session comment dialog
	sessionForm                            *Dialog                      // Session creation dialog
	sessionList                            *SessionList                 // Session list component
	sessionOps                             *SessionOperations           // Session lifecycle operations
	sessionRenameForm                      *Dialog                      // Session rename dialog
	sessionService                         *services.SessionService     // Session lifecycle service
	sessionState                           *domain.SessionCollection    // State data for git metadata and status
	sessionStatusForm                      *Dialog                      // Session status dialog
	sessionToArchive                       *ports.TmuxSession           // Session being archived (for worktree removal)
	sessionToKill                          *ports.TmuxSession           // Session being killed (for worktree removal)
	shellService                           *services.ShellService       // Shell session service
	state                                  uiState
	statusConfig                           *config.StatusConfig         // Status configuration for implementation statuses
	timestampConfig                        *config.TimestampColorConfig // Timestamp color configuration
	timestampMode                          TimestampMode
	tmuxStatusPosition                     string
	tokenChart                             *TokenChart                  // Token usage chart component
	width                                  int
	worktreeRemovalForm                    *Dialog                      // Worktree removal dialog
}

func NewModel(
	editor string,
	errorClearDelay time.Duration,
	statusConfig *config.StatusConfig,
	timestampConfig *config.TimestampColorConfig,
	devMode bool,
	showTimestamps bool,
	showTokenChart bool,
	tmuxStatusPosition string,
	allowDangerouslySkipPermissionsDefault bool,
	tipsConfig TipsConfig,
	keysConfig config.KeyBindingsConfig,
	gitService *services.GitService,
	sessionService *services.SessionService,
	shellService *services.ShellService,
	tokenStatsService *services.TokenStatsService,
) *Model {
	// Load session state - this is the source of truth
	sessionState, stateErr := sessionService.LoadState(context.Background(), false)
	errorManager := NewErrorManager(errorClearDelay)
	if stateErr != nil {
		logging.Logger.Warn("Failed to load session state", "error", stateErr)
		errorManager.SetError(fmt.Errorf("failed to load state: %w", stateErr))
		sessionState = &domain.SessionCollection{Sessions: make(map[string]domain.Session)}
	}

	// Convert showTimestamps flag to TimestampMode
	var initialMode TimestampMode
	if showTimestamps {
		initialMode = TimestampRelative
	} else {
		initialMode = TimestampHidden
	}

	// Create shared key map
	keys := NewKeyMap(keysConfig)

	// Create session operations component
	sessionOps := NewSessionOperations(errorManager, tmuxStatusPosition, sessionService, shellService)

	// Create session list component
	sessionList := NewSessionList(sessionService, gitService, editor, statusConfig, timestampConfig, devMode, initialMode, keys, tmuxStatusPosition, tipsConfig)

	// Create token chart component
	tokenChart := NewTokenChart(tokenStatsService)
	if showTokenChart {
		tokenChart.SetVisible(true)
	}

	return &Model{
		allowDangerouslySkipPermissionsDefault: allowDangerouslySkipPermissionsDefault,
		devMode:                                devMode,
		editor:                                 editor,
		errorManager:                           errorManager,
		gitService:                             gitService,
		keys:                                   keys,
		sessionList:                            sessionList,
		sessionOps:                             sessionOps,
		sessionService:                         sessionService,
		sessionState:                           sessionState,
		shellService:                           shellService,
		state:                                  stateList,
		statusConfig:                           statusConfig,
		timestampConfig:                        timestampConfig,
		timestampMode:                          initialMode,
		tmuxStatusPosition:                     tmuxStatusPosition,
		tokenChart:                             tokenChart,
	}
}

func (m *Model) Init() tea.Cmd {
	// Delegate to session list component (starts auto-refresh polling)
	return m.sessionList.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateCommandPalette:
		return m.updateCommandPalette(msg)
	case stateCommentingSession:
		return m.updateCommentingSession(msg)
	case stateConfirmingArchive:
		return m.updateConfirmingArchive(msg)
	case stateConfirmingWorktreeRemoval:
		return m.updateConfirmingWorktreeRemoval(msg)
	case stateCreatingSession:
		return m.updateCreatingSession(msg)
	case stateHelp:
		return m.updateHelp(msg)
	case stateRenamingSession:
		return m.updateRenamingSession(msg)
	case stateSendingText:
		return m.updateSendingText(msg)
	case stateSettingStatus:
		return m.updateSettingStatus(msg)
	}
	return m, nil
}

func (m *Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle action messages from SessionList
	switch msg := msg.(type) {
	// Phase 1: Foundation messages
	case QuitMsg:
		return m, tea.Quit
	case ShowHelpMsg:
		contentForm := NewHelpScreen(&m.keys)
		m.helpScreen = NewDialog("Help", contentForm, m.devMode)
		m.state = stateHelp
		// Send initial WindowSizeMsg so viewport can initialize
		initCmd := m.helpScreen.Init()
		updatedDialog, sizeCmd := m.helpScreen.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.helpScreen = updatedDialog.(*Dialog)
		return m, tea.Batch(initCmd, sizeCmd)
	case AttachSessionMsg:
		return m, m.sessionOps.AttachToSession(msg.Session.Name)

	// Phase 2: Dialog action messages
	case RenameSessionMsg:
		// Get current display name
		currentDisplayName := msg.SessionName
		if sessionInfo, ok := m.sessionState.Sessions[msg.SessionName]; ok && sessionInfo.DisplayName != "" {
			currentDisplayName = sessionInfo.DisplayName
		}
		contentForm := NewSessionRenameForm(m.sessionService, m.sessionState, msg.SessionName, currentDisplayName)
		m.sessionRenameForm = NewDialog("Rename Session", contentForm, m.devMode)
		m.state = stateRenamingSession
		return m, m.sessionRenameForm.Init()

	case CommentSessionMsg:
		// Get current comment
		currentComment := ""
		if sessionInfo, ok := m.sessionState.Sessions[msg.SessionName]; ok {
			currentComment = sessionInfo.Comment
		}
		contentForm := NewSessionCommentForm(m.sessionService, msg.SessionName, currentComment)
		m.sessionCommentForm = NewDialog("Edit Session Comment", contentForm, m.devMode)
		m.state = stateCommentingSession
		return m, m.sessionCommentForm.Init()

	case SetStatusSessionMsg:
		// Get current status
		var currentStatus *string
		if sessionInfo, ok := m.sessionState.Sessions[msg.SessionName]; ok {
			currentStatus = sessionInfo.Status
		}
		contentForm := NewSessionStatusForm(m.sessionService, msg.SessionName, currentStatus, m.statusConfig)
		m.sessionStatusForm = NewDialog("Set Status", contentForm, m.devMode)
		m.state = stateSettingStatus
		return m, m.sessionStatusForm.Init()

	case SendTextSessionMsg:
		contentForm := NewSendTextForm(m.shellService, msg.SessionName)
		m.sendTextForm = NewDialog("Send Text to Claude", contentForm, m.devMode)
		m.state = stateSendingText
		return m, m.sendTextForm.Init()

	case OpenEditorSessionMsg:
		sessionInfo, exists := m.sessionState.Sessions[msg.SessionName]
		if !exists || sessionInfo.WorktreePath == "" {
			m.errorManager.SetError(fmt.Errorf("no worktree associated with session '%s'", msg.SessionName))
			return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
		}
		if err := m.shellService.OpenEditor(sessionInfo.WorktreePath, m.editor); err != nil {
			m.errorManager.SetError(fmt.Errorf("failed to open editor: %w", err))
			return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
		}
		return m, m.sessionList.Init()

	case NewSessionMsg:
		// Pre-fill repo field if starting in a git folder
		defaultRepoSource := msg.DefaultRepoSource
		if defaultRepoSource == "" {
			cwd, err := os.Getwd()
			if err != nil {
				logging.Logger.Debug("Failed to get current working directory", "error", err)
			}
			if isGit, repoPath := m.gitService.IsGitRepo(cwd); isGit {
				if remoteURL := m.gitService.GetRemoteURL(repoPath); remoteURL != "" {
					defaultRepoSource = remoteURL
					logging.Logger.Info("Pre-filling repository field with remote URL", "remote_url", remoteURL)
				} else {
					logging.Logger.Warn("Git repository has no remote configured, leaving repo field empty")
				}
			}
		}
		logging.Logger.Debug("Creating new session dialog",
			"allow_dangerously_skip_permissions_default", m.allowDangerouslySkipPermissionsDefault,
			"default_repo_source", defaultRepoSource)
		contentForm := NewSessionForm(m.gitService, m.sessionService, m.sessionState, m.tmuxStatusPosition, m.allowDangerouslySkipPermissionsDefault, defaultRepoSource)
		m.sessionForm = NewDialog("Create Session", contentForm, m.devMode)
		m.state = stateCreatingSession
		return m, m.sessionForm.Init()

	case NewSessionFromTemplateMsg:
		// Get the repo source from the template session
		var repoSource string
		if sessionInfo, exists := m.sessionState.Sessions[msg.TemplateSessionName]; exists {
			repoSource = sessionInfo.RepoSource

			// Sanitize: remove branch suffix from URL (e.g., #feature-branch)
			if repoSource != "" {
				if parsed, err := m.gitService.ParseRepoSource(repoSource); err == nil {
					repoSource = parsed.Path
					logging.Logger.Debug("Sanitized repo source for template", "original", sessionInfo.RepoSource, "sanitized", repoSource)
				}
			}

			// If RepoSource is empty but RepoPath exists, fetch remote URL and update DB
			if repoSource == "" && sessionInfo.RepoPath != "" {
				logging.Logger.Info("RepoSource empty, fetching remote URL from RepoPath", "repo_path", sessionInfo.RepoPath)
				if remoteURL := m.gitService.GetRemoteURL(sessionInfo.RepoPath); remoteURL != "" {
					repoSource = remoteURL
					logging.Logger.Info("Fetched remote URL, updating database", "remote_url", remoteURL)

					// Update the session in the database with the fetched RepoSource
					if err := m.sessionService.UpdateRepoSource(context.Background(), msg.TemplateSessionName, remoteURL); err != nil {
						logging.Logger.Error("Failed to update RepoSource in database", "error", err)
					} else {
						// Also update in-memory state
						sessionInfo.RepoSource = remoteURL
						m.sessionState.Sessions[msg.TemplateSessionName] = sessionInfo
					}
				}
			}

			logging.Logger.Info("Creating new session from template", "template_session", msg.TemplateSessionName, "repo_source", repoSource)
		}
		logging.Logger.Debug("Creating new session from template dialog",
			"allow_dangerously_skip_permissions_default", m.allowDangerouslySkipPermissionsDefault,
			"default_repo_source", repoSource)
		contentForm := NewSessionForm(m.gitService, m.sessionService, m.sessionState, m.tmuxStatusPosition, m.allowDangerouslySkipPermissionsDefault, repoSource)
		m.sessionForm = NewDialog("Create Session (from same repo)", contentForm, m.devMode)
		m.state = stateCreatingSession
		return m, m.sessionForm.Init()

	// Phase 3: Complex action messages
	case KillSessionMsg:
		return m.handleKillSession(msg.SessionName)

	case ArchiveSessionMsg:
		return m.handleArchiveSession(msg.SessionName)

	case ToggleFlagSessionMsg:
		return m.handleToggleFlag(msg.SessionName)

	case AttachShellSessionMsg:
		shellSessionName := m.sessionOps.GetOrCreateShellSession(msg.Session, m.sessionState)
		if shellSessionName != "" {
			return m, m.sessionOps.AttachToSession(shellSessionName)
		}
		return m, nil

	case TestErrorMsg:
		m.errorManager.SetError(fmt.Errorf("this is a very long test error message to verify that the error display truncation functionality works correctly and ensures that error text wraps properly across multiple lines and eventually gets truncated with ellipsis if it exceeds the maximum allowed length of three lines which should be enforced by the formatErrorForDisplay function"))
		return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())

	// Command palette messages
	case ShowCommandPaletteMsg:
		// Get selected session info for the palette header
		var session *ports.TmuxSession
		var sessionName string
		if item, ok := m.sessionList.list.SelectedItem().(SessionItem); ok {
			session = item.Session
			sessionName = item.DisplayName
		}

		m.commandPalette = NewCommandPalette(session, sessionName)
		m.state = stateCommandPalette

		// Send initial window size
		initCmd := m.commandPalette.Init()
		_, sizeCmd := m.commandPalette.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, tea.Batch(initCmd, sizeCmd)

	case ToggleTimestampsMsg:
		// Cycle timestamps (same logic as existing key handler)
		switch m.timestampMode {
		case TimestampRelative:
			m.timestampMode = TimestampAbsolute
		case TimestampAbsolute:
			m.timestampMode = TimestampHidden
		case TimestampHidden:
			m.timestampMode = TimestampRelative
		}
		m.sessionList.timestampMode = m.timestampMode
		refreshCmd := m.sessionList.RefreshFromState()
		return m, tea.Batch(refreshCmd, m.sessionList.Init())

	case ToggleTokenChartMsg:
		m.tokenChart.Toggle()
		m.recalculateListHeight()
		return m, m.sessionList.Init()

	case CycleStatusMsg:
		// Delegate to session list's cycleSessionStatus
		return m, m.sessionList.cycleSessionStatus(msg.SessionName)
	}

	// Handle clear error message
	if _, ok := msg.(clearErrorMsg); ok {
		m.errorManager.ClearError()
		return m, nil
	}

	// Refresh token chart on poll cycle (when visible)
	if _, ok := msg.(checkStateMsg); ok && m.tokenChart.IsVisible() {
		m.tokenChart.Refresh()
	}

	// Handle window size updates
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height

		// Recalculate list height
		m.recalculateListHeight()
	}

	// Handle detach message - session list auto-refreshes via polling
	if _, ok := msg.(detachedMsg); ok {
		m.state = stateList
		refreshCmd := m.sessionList.RefreshFromState()
		return m, tea.Batch(refreshCmd, m.sessionList.Init())
	}

	// Handle errors from attach failures (e.g., tmux nested session errors)
	if err, ok := msg.(error); ok {
		m.errorManager.SetError(fmt.Errorf("failed to attach to session: %w", err))
		return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
	}

	// Hidden test command: alt+shift+e generates Model-level error (persists 5 seconds)
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "alt+E" {
		m.errorManager.SetError(fmt.Errorf("this is a persistent Model-level test error that demonstrates the error display functionality with automatic height adjustment and will clear after five seconds to verify that the list height properly expands back to normal and ensures all session items remain visible throughout the entire error lifecycle"))
		return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
	}

	// Toggle timestamps display mode
	// Cycle: Relative -> Absolute -> Hidden -> Relative -> ...
	if keyMsg, ok := msg.(tea.KeyMsg); ok && key.Matches(keyMsg, m.keys.Application.Timestamps.Binding) {
		switch m.timestampMode {
		case TimestampRelative:
			m.timestampMode = TimestampAbsolute
		case TimestampAbsolute:
			m.timestampMode = TimestampHidden
		case TimestampHidden:
			m.timestampMode = TimestampRelative
		}
		m.sessionList.timestampMode = m.timestampMode
		refreshCmd := m.sessionList.RefreshFromState()
		return m, tea.Batch(refreshCmd, m.sessionList.Init())
	}

	// Toggle token chart
	if keyMsg, ok := msg.(tea.KeyMsg); ok && key.Matches(keyMsg, m.keys.Application.TokenChart.Binding) {
		m.tokenChart.Toggle()
		// Recalculate list height
		m.recalculateListHeight()
		return m, m.sessionList.Init()
	}

	// Delegate to SessionList component
	newList, cmd := m.sessionList.Update(msg)
	if sl, ok := newList.(*SessionList); ok {
		m.sessionList = sl
	}

	return m, cmd
}

func (m *Model) updateCommandPalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Forward window size to palette
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
	}

	// Delegate to palette
	updated, cmd := m.commandPalette.Update(msg)
	m.commandPalette = updated.(*CommandPalette)

	// Check if palette completed
	if m.commandPalette.Completed {
		result := m.commandPalette.Result
		m.state = stateList
		m.commandPalette = nil

		if result.Cancelled || result.Action == nil {
			return m, m.sessionList.Init()
		}

		// Get selected session for dispatcher
		var session *ports.TmuxSession
		if item, ok := m.sessionList.list.SelectedItem().(SessionItem); ok {
			session = item.Session
		}

		// Dispatch the action
		dispatcher := NewActionDispatcher(session)
		actionMsg := dispatcher.Dispatch(*result.Action)

		if actionMsg != nil {
			// Process the action message through updateList
			return m.updateList(actionMsg)
		}

		return m, m.sessionList.Init()
	}

	return m, cmd
}

func (m *Model) updateCreatingSession(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to dialog (it handles cancel internally)
	updated, cmd := m.sessionForm.Update(msg)
	m.sessionForm = updated.(*Dialog)

	// Check if dialog completed
	if content, ok := m.sessionForm.Content().(*SessionForm); ok && content.Completed {
		result := content.Result()
		m.state = stateList
		m.sessionForm = nil

		if result.Error != nil {
			m.errorManager.SetError(fmt.Errorf("failed to create session: %w", result.Error))
			return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
		}

		if !result.Cancelled {
			// Use helper - eliminates duplication
			refreshCmd, err := m.reloadSessionStateAfterDialog()
			if err != nil {
				m.errorManager.SetError(err)
				logging.Logger.Warn("Failed to reload session state", "error", err)
				return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
			}
			// Select the newly added session (always at position 0)
			m.sessionList.list.Select(0)
			return m, tea.Batch(refreshCmd, m.sessionList.Init())
		}

		return m, m.sessionList.Init()
	}

	return m, cmd
}

func (m *Model) updateRenamingSession(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to dialog (it handles cancel internally)
	updated, cmd := m.sessionRenameForm.Update(msg)
	m.sessionRenameForm = updated.(*Dialog)

	// Check if dialog completed
	if content, ok := m.sessionRenameForm.Content().(*SessionRenameForm); ok && content.Completed {
		result := content.Result()
		m.state = stateList
		m.sessionRenameForm = nil

		if result.Error != nil {
			m.errorManager.SetError(fmt.Errorf("failed to rename session: %w", result.Error))
			return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
		}

		if !result.Cancelled {
			// Use helper - eliminates duplication
			refreshCmd, err := m.reloadSessionStateAfterDialog()
			if err != nil {
				m.errorManager.SetError(err)
				return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
			}
			return m, tea.Batch(refreshCmd, m.sessionList.Init())
		}

		return m, m.sessionList.Init()
	}

	return m, cmd
}

func (m *Model) updateSettingStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to dialog (it handles cancel internally)
	updated, cmd := m.sessionStatusForm.Update(msg)
	m.sessionStatusForm = updated.(*Dialog)

	// Check if dialog completed
	if content, ok := m.sessionStatusForm.Content().(*SessionStatusForm); ok && content.Completed {
		result := content.Result()
		m.state = stateList
		m.sessionStatusForm = nil

		if result.Error != nil {
			m.errorManager.SetError(fmt.Errorf("failed to update status: %w", result.Error))
			return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
		}

		if !result.Cancelled {
			// Use helper - eliminates duplication
			refreshCmd, err := m.reloadSessionStateAfterDialog()
			if err != nil {
				m.errorManager.SetError(err)
				return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
			}
			return m, tea.Batch(refreshCmd, m.sessionList.Init())
		}

		return m, m.sessionList.Init()
	}

	return m, cmd
}

func (m *Model) updateCommentingSession(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to dialog (it handles cancel internally)
	updated, cmd := m.sessionCommentForm.Update(msg)
	m.sessionCommentForm = updated.(*Dialog)

	// Check if dialog completed
	if content, ok := m.sessionCommentForm.Content().(*SessionCommentForm); ok && content.Completed {
		result := content.Result()
		m.state = stateList
		m.sessionCommentForm = nil

		if result.Error != nil {
			m.errorManager.SetError(fmt.Errorf("failed to update comment: %w", result.Error))
			return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
		}

		if !result.Cancelled {
			// Use helper - eliminates duplication
			refreshCmd, err := m.reloadSessionStateAfterDialog()
			if err != nil {
				m.errorManager.SetError(err)
				return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
			}
			return m, tea.Batch(refreshCmd, m.sessionList.Init())
		}

		return m, m.sessionList.Init()
	}

	return m, cmd
}

func (m *Model) updateSendingText(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to dialog (it handles cancel internally)
	updated, cmd := m.sendTextForm.Update(msg)
	m.sendTextForm = updated.(*Dialog)

	// Check if dialog completed
	if content, ok := m.sendTextForm.Content().(*SendTextForm); ok && content.Completed {
		result := content.Result()
		m.state = stateList
		m.sendTextForm = nil

		if result.Error != nil {
			m.errorManager.SetError(fmt.Errorf("failed to send text: %w", result.Error))
			return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
		}

		return m, m.sessionList.Init()
	}

	return m, cmd
}

// reloadSessionStateAfterDialog reloads session state and refreshes the list.
// Returns the command from RefreshFromState for pagination updates.
func (m *Model) reloadSessionStateAfterDialog() (tea.Cmd, error) {
	newState, err := m.sessionService.LoadState(context.Background(), false)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh sessions: %w", err)
	}
	*m.sessionState = *newState
	return m.sessionList.RefreshFromState(), nil
}

// getFreshSessionInfo loads fresh session info from the database to avoid stale state issues.
// Returns the Session and true if found, or zero value and false if not found.
func (m *Model) getFreshSessionInfo(sessionName string) (domain.Session, bool) {
	freshState, err := m.sessionService.LoadState(context.Background(), false)
	if err != nil {
		logging.Logger.Error("Failed to load fresh state", "error", err)
		// Fall back to cached state
		freshState = m.sessionState
	}
	sessionInfo, ok := freshState.Sessions[sessionName]
	return sessionInfo, ok
}

// handleKillSession handles the kill session action
func (m *Model) handleKillSession(sessionName string) (tea.Model, tea.Cmd) {
	session := &ports.TmuxSession{Name: sessionName}

	// Use fresh state to avoid race condition with polling
	if sessionInfo, ok := m.getFreshSessionInfo(sessionName); ok && sessionInfo.WorktreePath != "" {
		m.sessionToKill = session
		removeWorktree := false
		m.formRemoveWorktree = &removeWorktree
		m.worktreeRemovalForm = m.createWorktreeRemovalDialog(sessionInfo.WorktreePath)
		m.state = stateConfirmingWorktreeRemoval
		return m, m.worktreeRemovalForm.Init()
	}
	return m, m.sessionOps.KillSession(session, m.sessionState, m.sessionList)
}

// handleArchiveSession handles the archive session action
func (m *Model) handleArchiveSession(sessionName string) (tea.Model, tea.Cmd) {
	session := &ports.TmuxSession{Name: sessionName}

	// Use fresh state to avoid race condition with polling
	if sessionInfo, ok := m.getFreshSessionInfo(sessionName); ok && sessionInfo.WorktreePath != "" {
		m.sessionToArchive = session
		removeWorktree := false
		m.formRemoveWorktreeArchive = &removeWorktree
		form := m.createArchiveWorktreeRemovalForm(sessionInfo.WorktreePath)
		m.worktreeRemovalForm = NewDialog("Archive Session", form, m.devMode)
		m.state = stateConfirmingArchive
		return m, m.worktreeRemovalForm.Init()
	}
	return m, m.sessionOps.ArchiveSession(session, false, m.sessionState, m.sessionList)
}

// handleToggleFlag handles the toggle flag action
func (m *Model) handleToggleFlag(sessionName string) (tea.Model, tea.Cmd) {
	if err := m.sessionService.ToggleFlag(context.Background(), sessionName); err != nil {
		m.errorManager.SetError(fmt.Errorf("failed to toggle flag: %w", err))
		return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
	}

	// Reload session state
	newSessionState, err := m.sessionService.LoadState(context.Background(), false)
	if err != nil {
		m.errorManager.SetError(fmt.Errorf("failed to refresh sessions: %w", err))
		refreshCmd := m.sessionList.RefreshFromState()
		return m, tea.Batch(refreshCmd, m.sessionList.Init(), m.errorManager.ClearAfterDelay())
	}
	*m.sessionState = *newSessionState

	// Refresh UI
	refreshCmd := m.sessionList.RefreshFromState()
	return m, tea.Batch(refreshCmd, m.sessionList.Init())
}

// recalculateListHeight calculates and sets the list height based on current state
func (m *Model) recalculateListHeight() {
	// Layout breakdown:
	// - Header (2 lines) + Legend (1 line) + spacing (1) = 4 lines from SessionList fixed content
	// - Bottom section: separator (1) + tip/error (2) = 3 lines
	// - With chart: chart height (includes its leading newline)
	overhead := 7 // header + legend + spacing + bottom section
	if m.tokenChart.IsVisible() {
		overhead += m.tokenChart.Height() // chart (includes leading newline)
	}

	listHeight := m.height - overhead
	if listHeight < 1 {
		listHeight = 1
	}
	m.sessionList.SetSize(m.width, m.height, listHeight)
}

func (m *Model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to dialog (it handles cancel internally)
	updated, cmd := m.helpScreen.Update(msg)
	m.helpScreen = updated.(*Dialog)

	// Check if dialog completed
	if content, ok := m.helpScreen.Content().(*HelpScreen); ok && content.Completed {
		m.state = stateList
		m.helpScreen = nil
		return m, m.sessionList.Init()
	}

	return m, cmd
}

type detachedMsg struct{}

func (m *Model) updateConfirmingArchive(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, m.keys.Navigation.ClearFilter.Binding, m.keys.Application.ForceQuit.Binding) {
			m.state = stateList
			m.worktreeRemovalForm = nil
			m.sessionToArchive = nil
			m.formRemoveWorktreeArchive = nil
			return m, nil
		}
	}

	// Safety check for nil form
	if m.worktreeRemovalForm == nil {
		m.state = stateList
		m.sessionToArchive = nil
		return m, nil
	}

	// Forward message to Dialog
	updated, cmd := m.worktreeRemovalForm.Update(msg)
	m.worktreeRemovalForm = updated.(*Dialog)

	// Access wrapped huh.Form to check completion
	if form, ok := m.worktreeRemovalForm.Content().(*huh.Form); ok {
		// Check if form completed
		if form.State == huh.StateCompleted {
			removeWorktree := *m.formRemoveWorktreeArchive // Dereference pointer
			session := m.sessionToArchive

			logging.Logger.Info("Archive worktree removal decision", "remove", removeWorktree, "session", session.Name)

			// Reset state
			m.state = stateList
			m.worktreeRemovalForm = nil
			m.sessionToArchive = nil
			m.formRemoveWorktreeArchive = nil

			// Archive with worktree removal decision
			return m, m.sessionOps.ArchiveSession(session, removeWorktree, m.sessionState, m.sessionList)
		}
	}

	return m, cmd
}

func (m *Model) updateConfirmingWorktreeRemoval(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, m.keys.Navigation.ClearFilter.Binding, m.keys.Application.ForceQuit.Binding) {
			m.state = stateList
			m.worktreeRemovalForm = nil
			m.sessionToKill = nil
			m.formRemoveWorktree = nil
			return m, nil
		}
	}

	// Safety check for nil form
	if m.worktreeRemovalForm == nil {
		m.state = stateList
		m.sessionToKill = nil
		return m, nil
	}

	// Forward message to Dialog
	updated, cmd := m.worktreeRemovalForm.Update(msg)
	m.worktreeRemovalForm = updated.(*Dialog)

	// Access wrapped huh.Form to check completion
	if form, ok := m.worktreeRemovalForm.Content().(*huh.Form); ok {
		// Check if form completed
		if form.State == huh.StateCompleted {
			removeWorktree := *m.formRemoveWorktree // Dereference pointer
			session := m.sessionToKill

			logging.Logger.Info("Worktree removal decision", "remove", removeWorktree, "session", session.Name)

			// Get worktree path and repo path from session info
			sessionInfo := m.sessionState.Sessions[session.Name]
			worktreePath := sessionInfo.WorktreePath
			repoPath := sessionInfo.RepoPath

			// Remove worktree if requested
			var worktreeErr bool
			if removeWorktree {
				logging.Logger.Info("Removing worktree", "path", worktreePath, "repo", repoPath)
				if err := m.gitService.RemoveWorktree(repoPath, worktreePath); err != nil {
					m.errorManager.SetError(fmt.Errorf("failed to remove worktree: %w", err))
					logging.Logger.Error("Failed to remove worktree", "error", err, "path", worktreePath)
					worktreeErr = true
				} else {
					logging.Logger.Info("Worktree removed successfully", "path", worktreePath)
				}
			} else {
				logging.Logger.Info("Keeping worktree", "path", worktreePath)
			}

			// Kill the session
			killCmd := m.sessionOps.KillSession(session, m.sessionState, m.sessionList)

			// Reset state
			m.state = stateList
			m.worktreeRemovalForm = nil
			m.sessionToKill = nil
			m.formRemoveWorktree = nil

			// If there was a worktree error, add clearErrorAfterDelay to the batch
			if worktreeErr {
				return m, tea.Batch(killCmd, m.errorManager.ClearAfterDelay())
			}
			return m, killCmd
		}
	}

	return m, cmd
}

// createArchiveWorktreeRemovalForm creates a confirmation form for removing a worktree when archiving
func (m *Model) createArchiveWorktreeRemovalForm(worktreePath string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Remove worktree at %s?", worktreePath)).
				Description("Archive will hide the session. Remove the worktree too?").
				Value(m.formRemoveWorktreeArchive).
				Affirmative("Remove").
				Negative("Keep"),
		),
	)
}

// createWorktreeRemovalForm creates a confirmation form for removing a worktree
func (m *Model) createWorktreeRemovalDialog(worktreePath string) *Dialog {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Remove worktree at %s?", worktreePath)).
				Description("This will delete the working tree but preserve commits.").
				Value(m.formRemoveWorktree). // Already a pointer, don't take address again
				Affirmative("Remove").
				Negative("Keep"),
		),
	)

	return NewDialog("Remove Worktree", form, m.devMode)
}

func (m *Model) View() string {
	switch m.state {
	case stateList:
		view := m.sessionList.View()

		// Token chart (if visible)
		if m.tokenChart.IsVisible() {
			view += "\n" + m.tokenChart.View() + "\n"
		}

		// Bottom section - fixed 2 lines (error or tip or empty)
		// Error takes priority over tip (tip is hidden while error displays)
		view += "\n"
		if m.errorManager.HasError() {
			errorText := formatErrorForDisplay(m.errorManager.GetError(), m.width)
			view += theme.ErrorStyle.Render(errorText)
		} else if tip := m.sessionList.GetCurrentTip(); tip != "" {
			view += tip + "\n "
		} else {
			view += " \n "
		}

		return view
	case stateCommandPalette:
		if m.commandPalette != nil {
			// Render dimmed background
			background := m.sessionList.View()
			if m.tokenChart.IsVisible() {
				background += "\n" + m.tokenChart.View() + "\n"
			}
			dimmed := applyDimOverlay(background)

			// Render palette centered
			palette := m.commandPalette.View()
			return compositeOverlay(dimmed, palette, m.width, m.height)
		}
	case stateCommentingSession:
		if m.sessionCommentForm != nil {
			return m.sessionCommentForm.View()
		}
	case stateConfirmingArchive:
		if m.worktreeRemovalForm != nil {
			return m.worktreeRemovalForm.View()
		}
	case stateConfirmingWorktreeRemoval:
		if m.worktreeRemovalForm != nil {
			return m.worktreeRemovalForm.View()
		}
	case stateCreatingSession:
		if m.sessionForm != nil {
			return m.sessionForm.View()
		}
	case stateHelp:
		if m.helpScreen != nil {
			return m.helpScreen.View()
		}
	case stateRenamingSession:
		if m.sessionRenameForm != nil {
			return m.sessionRenameForm.View()
		}
	case stateSendingText:
		if m.sendTextForm != nil {
			return m.sendTextForm.View()
		}
	case stateSettingStatus:
		if m.sessionStatusForm != nil {
			return m.sessionStatusForm.View()
		}
	}
	return ""
}

// applyDimOverlay applies a dimmed style to all lines of the background.
func applyDimOverlay(background string) string {
	lines := strings.Split(background, "\n")
	for i, line := range lines {
		// Strip existing ANSI codes and apply dim style
		lines[i] = theme.DimmedStyle.Render(stripAnsi(line))
	}
	return strings.Join(lines, "\n")
}

// stripAnsi removes ANSI escape codes from a string.
func stripAnsi(s string) string {
	// Simple regex-free approach: skip escape sequences
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// compositeOverlay centers the palette over the dimmed background.
func compositeOverlay(background, palette string, width, height int) string {
	// Position the palette centered
	positioned := lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		palette,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(theme.ColorDimmed),
	)

	return positioned
}
