package ui

import (
	"context"
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"rocha/internal/config"
	"rocha/internal/domain"
	"rocha/internal/logging"
	"rocha/internal/ports"
	"rocha/internal/services"
	"rocha/internal/theme"
)

type uiState int

const (
	stateList uiState = iota
	stateCreatingSession
	stateConfirmingArchive
	stateConfirmingWorktreeRemoval
	stateHelp
	stateRenamingSession
	stateSendingText
	stateSettingStatus
	stateCommentingSession
)

type Model struct {
	allowDangerouslySkipPermissionsDefault bool                         // Default value from settings for new sessions
	devMode                                bool                         // Development mode (shows version info in dialogs)
	editor                                 string                       // Editor to open sessions in
	errorManager                           *ErrorManager                // Error display and auto-clearing
	formRemoveWorktree                     *bool                        // Worktree removal decision (pointer to persist across updates)
	formRemoveWorktreeArchive              *bool                        // Worktree removal decision for archive (pointer to persist across updates)
	gitService                             *services.GitService         // Git operations service
	height                                 int
	helpScreen                             *Dialog                      // Help screen dialog
	keys                                   KeyMap                       // Keyboard shortcuts
	listActionHandler                      *ListActionHandler           // Session list action processing
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
	tmuxStatusPosition string,
	allowDangerouslySkipPermissionsDefault bool,
	tipsConfig TipsConfig,
	gitService *services.GitService,
	sessionService *services.SessionService,
	shellService *services.ShellService,
) *Model {
	// Load session state - this is the source of truth
	sessionState, stateErr := sessionService.LoadState(context.Background(), false)
	errorManager := NewErrorManager(errorClearDelay)
	if stateErr != nil {
		log.Printf("Warning: failed to load session state: %v", stateErr)
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
	keys := NewKeyMap()

	// Create session operations component
	sessionOps := NewSessionOperations(errorManager, tmuxStatusPosition, sessionService, shellService)

	// Create session list component
	sessionList := NewSessionList(sessionService, gitService, editor, statusConfig, timestampConfig, devMode, initialMode, keys, tmuxStatusPosition, tipsConfig)

	// Create list action handler
	listActionHandler := NewListActionHandler(
		sessionList,
		sessionState,
		gitService,
		editor,
		statusConfig,
		errorManager,
		sessionOps,
		tmuxStatusPosition,
		devMode,
		allowDangerouslySkipPermissionsDefault,
		sessionService,
		shellService,
	)

	return &Model{
		allowDangerouslySkipPermissionsDefault: allowDangerouslySkipPermissionsDefault,
		devMode:                                devMode,
		editor:                                 editor,
		errorManager:                           errorManager,
		gitService:                             gitService,
		keys:                                   keys,
		listActionHandler:                      listActionHandler,
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
	case stateCreatingSession:
		return m.updateCreatingSession(msg)
	case stateConfirmingArchive:
		return m.updateConfirmingArchive(msg)
	case stateConfirmingWorktreeRemoval:
		return m.updateConfirmingWorktreeRemoval(msg)
	case stateHelp:
		return m.updateHelp(msg)
	case stateRenamingSession:
		return m.updateRenamingSession(msg)
	case stateSendingText:
		return m.updateSendingText(msg)
	case stateSettingStatus:
		return m.updateSettingStatus(msg)
	case stateCommentingSession:
		return m.updateCommentingSession(msg)
	}
	return m, nil
}

func (m *Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle clear error message
	if _, ok := msg.(clearErrorMsg); ok {
		m.errorManager.ClearError()
		return m, nil
	}

	// Handle window size updates
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available height for SessionList's internal list
		// Layout: Header (2) + spacing (1) + Legend (1) + spacing (1) + [List] + Bottom section (3)
		// Bottom section: separator (1) + content (2 lines fixed)
		// Total overhead: 8 lines
		listHeight := msg.Height - 8
		if listHeight < 1 {
			listHeight = 1
		}
		m.sessionList.SetSize(msg.Width, msg.Height, listHeight)
	}

	// Handle detach message - session list auto-refreshes via polling
	if _, ok := msg.(detachedMsg); ok {
		m.state = stateList
		m.sessionList.RefreshFromState()
		return m, m.sessionList.Init()
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

	// Toggle timestamps display mode with 't' key
	// Cycle: Relative -> Absolute -> Hidden -> Relative -> ...
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "t" {
		switch m.timestampMode {
		case TimestampRelative:
			m.timestampMode = TimestampAbsolute
		case TimestampAbsolute:
			m.timestampMode = TimestampHidden
		case TimestampHidden:
			m.timestampMode = TimestampRelative
		}
		m.sessionList.timestampMode = m.timestampMode
		m.sessionList.RefreshFromState()
		return m, m.sessionList.Init()
	}

	// Delegate to SessionList component
	newList, cmd := m.sessionList.Update(msg)
	if sl, ok := newList.(*SessionList); ok {
		m.sessionList = sl
	}

	// Handle SessionList results
	if m.sessionList.ShouldQuit {
		return m, tea.Quit
	}

	// Process actions via ListActionHandler
	actionResult := m.listActionHandler.ProcessActions()
	return m.handleActionResult(actionResult, cmd)
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
			if err := m.reloadSessionStateAfterDialog(); err != nil {
				m.errorManager.SetError(err)
				log.Printf("Warning: failed to reload session state: %v", err)
				return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
			}
			// Select the newly added session (always at position 0)
			m.sessionList.list.Select(0)
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
			if err := m.reloadSessionStateAfterDialog(); err != nil {
				m.errorManager.SetError(err)
				return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
			}
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
			if err := m.reloadSessionStateAfterDialog(); err != nil {
				m.errorManager.SetError(err)
				return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
			}
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
			if err := m.reloadSessionStateAfterDialog(); err != nil {
				m.errorManager.SetError(err)
				return m, tea.Batch(m.sessionList.Init(), m.errorManager.ClearAfterDelay())
			}
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

// reloadSessionStateAfterDialog reloads state and refreshes list.
// Eliminates the 8 duplications of reload pattern across dialog handlers.
func (m *Model) reloadSessionStateAfterDialog() error {
	newState, err := m.sessionService.LoadState(context.Background(), false)
	if err != nil {
		return fmt.Errorf("failed to refresh sessions: %w", err)
	}
	*m.sessionState = *newState
	m.sessionList.RefreshFromState()
	return nil
}

// handleActionResult processes the result from ListActionHandler and takes appropriate action.
func (m *Model) handleActionResult(result ActionResult, fallbackCmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch result.ActionType {
	case ActionNone:
		if result.Cmd != nil {
			return m, result.Cmd
		}
		return m, fallbackCmd

	case ActionAttachSession, ActionAttachShellSession:
		return m, result.Cmd

	case ActionKillSession, ActionArchiveSession, ActionOpenEditor, ActionToggleFlag:
		return m, result.Cmd

	case ActionShowKillWorktreeDialog:
		m.sessionToKill = result.SessionToKill
		removeWorktree := false
		m.formRemoveWorktree = &removeWorktree
		m.worktreeRemovalForm = m.createWorktreeRemovalDialog(result.WorktreePath)
		m.state = stateConfirmingWorktreeRemoval
		return m, m.worktreeRemovalForm.Init()

	case ActionShowArchiveWorktreeDialog:
		m.sessionToArchive = result.SessionToArchive
		removeWorktree := false
		m.formRemoveWorktreeArchive = &removeWorktree
		form := m.createArchiveWorktreeRemovalForm(result.WorktreePath)
		m.worktreeRemovalForm = NewDialog("Archive Session", form, m.devMode)
		m.state = stateConfirmingArchive
		return m, m.worktreeRemovalForm.Init()

	case ActionShowRenameDialog:
		m.sessionRenameForm = NewDialog(result.DialogTitle, result.DialogContent, m.devMode)
		m.state = result.NewState
		return m, m.sessionRenameForm.Init()

	case ActionShowStatusDialog:
		m.sessionStatusForm = NewDialog(result.DialogTitle, result.DialogContent, m.devMode)
		m.state = result.NewState
		return m, m.sessionStatusForm.Init()

	case ActionShowCommentDialog:
		m.sessionCommentForm = NewDialog(result.DialogTitle, result.DialogContent, m.devMode)
		m.state = result.NewState
		return m, m.sessionCommentForm.Init()

	case ActionShowSendTextDialog:
		m.sendTextForm = NewDialog(result.DialogTitle, result.DialogContent, m.devMode)
		m.state = result.NewState
		return m, m.sendTextForm.Init()

	case ActionShowHelpDialog:
		contentForm := NewHelpScreen(&m.keys)
		m.helpScreen = NewDialog("Help", contentForm, m.devMode)
		m.state = result.NewState
		// Send initial WindowSizeMsg so viewport can initialize
		initCmd := m.helpScreen.Init()
		updatedDialog, sizeCmd := m.helpScreen.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.helpScreen = updatedDialog.(*Dialog)
		return m, tea.Batch(initCmd, sizeCmd)

	case ActionShowNewSessionDialog:
		logging.Logger.Debug("Creating new session dialog",
			"allow_dangerously_skip_permissions_default", m.allowDangerouslySkipPermissionsDefault,
			"default_repo_source", result.DefaultRepoSource)
		contentForm := NewSessionForm(m.gitService, m.sessionService, m.sessionState, m.tmuxStatusPosition, m.allowDangerouslySkipPermissionsDefault, result.DefaultRepoSource)
		m.sessionForm = NewDialog("Create Session", contentForm, m.devMode)
		m.state = result.NewState
		return m, m.sessionForm.Init()

	case ActionShowNewSessionFromDialog:
		logging.Logger.Debug("Creating new session from template dialog",
			"allow_dangerously_skip_permissions_default", m.allowDangerouslySkipPermissionsDefault,
			"default_repo_source", result.DefaultRepoSource)
		contentForm := NewSessionForm(m.gitService, m.sessionService, m.sessionState, m.tmuxStatusPosition, m.allowDangerouslySkipPermissionsDefault, result.DefaultRepoSource)
		m.sessionForm = NewDialog("Create Session (from same repo)", contentForm, m.devMode)
		m.state = result.NewState
		return m, m.sessionForm.Init()

	default:
		return m, fallbackCmd
	}
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
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
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
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
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

		// Bottom section - fixed 2 lines (error or tip or empty)
		// Always reserve 2 lines to prevent layout shift
		view += "\n"
		if m.errorManager.HasError() {
			errorText := formatErrorForDisplay(m.errorManager.GetError(), m.width)
			view += theme.ErrorStyle.Render(errorText)
			// Clear tip when error is shown
			m.sessionList.ClearCurrentTip()
		} else if tip := m.sessionList.GetCurrentTip(); tip != "" {
			view += tip + "\n "
		} else {
			// Empty lines to maintain fixed 2-line spacing
			view += " \n "
		}

		return view
	case stateCreatingSession:
		if m.sessionForm != nil {
			return m.sessionForm.View()
		}
	case stateConfirmingArchive:
		if m.worktreeRemovalForm != nil {
			return m.worktreeRemovalForm.View()
		}
	case stateConfirmingWorktreeRemoval:
		if m.worktreeRemovalForm != nil {
			return m.worktreeRemovalForm.View()
		}
	case stateHelp:
		if m.helpScreen != nil {
			return m.helpScreen.View()
		}
	case stateRenamingSession:
		if m.sessionRenameForm != nil {
			return m.sessionRenameForm.View()
		}
	case stateSettingStatus:
		if m.sessionStatusForm != nil {
			return m.sessionStatusForm.View()
		}
	case stateCommentingSession:
		if m.sessionCommentForm != nil {
			return m.sessionCommentForm.View()
		}
	case stateSendingText:
		if m.sendTextForm != nil {
			return m.sendTextForm.View()
		}
	}
	return ""
}
