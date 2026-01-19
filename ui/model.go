package ui

import (
	"context"
	"fmt"
	"log"
	"rocha/editor"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/storage"
	"rocha/tmux"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Padding(1, 0)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 0)

	helpShortcutStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true)

	helpLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	branchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")) // Dimmed/gray

	workingIconStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("2")) // Green - actively working

	idleIconStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")) // Yellow - finished/idle

	waitingIconStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("1")) // Red - waiting for prompt

	exitedIconStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // Gray - Claude has exited

	additionsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")) // Green for additions

	deletionsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")) // Red for deletions
)

type uiState int

const (
	stateList uiState = iota
	stateCreatingSession
	stateConfirmingArchive
	stateConfirmingWorktreeRemoval
	stateHelp
	stateRenamingSession
	stateSettingStatus
	stateCommentingSession
)

type Model struct {
	devMode                   bool           // Development mode (shows version info in dialogs)
	editor                    string         // Editor to open sessions in
	err                       error
	errorClearDelay           time.Duration  // Duration before errors auto-clear
	formRemoveWorktree        *bool          // Worktree removal decision (pointer to persist across updates)
	formRemoveWorktreeArchive *bool          // Worktree removal decision for archive (pointer to persist across updates)
	height                    int
	helpScreen                *Dialog               // Help screen dialog
	keys                      KeyMap                // Keyboard shortcuts
	sessionCommentForm        *Dialog               // Session comment dialog
	sessionForm               *Dialog               // Session creation dialog
	sessionList               *SessionList          // Session list component
	sessionRenameForm         *Dialog               // Session rename dialog
	sessionState              *storage.SessionState // State data for git metadata and status
	sessionStatusForm         *Dialog               // Session status dialog
	sessionToArchive          *tmux.Session         // Session being archived (for worktree removal)
	sessionToKill             *tmux.Session         // Session being killed (for worktree removal)
	state                     uiState
	statusConfig              *StatusConfig         // Status configuration for implementation statuses
	store                     *storage.Store        // Storage for persistent state
	timestampConfig           *TimestampColorConfig // Timestamp color configuration
	timestampMode             TimestampMode
	tmuxClient                tmux.Client
	width                     int
	worktreeRemovalForm       *Dialog // Worktree removal dialog
	worktreePath              string
}

func NewModel(tmuxClient tmux.Client, store *storage.Store, worktreePath string, editor string, errorClearDelay time.Duration, statusConfig *StatusConfig, timestampConfig *TimestampColorConfig, devMode bool, showTimestamps bool) *Model {
	// Load session state - this is the source of truth
	sessionState, stateErr := store.Load(context.Background(), false)
	var errMsg error
	if stateErr != nil {
		log.Printf("Warning: failed to load session state: %v", stateErr)
		errMsg = fmt.Errorf("failed to load state: %w", stateErr)
		sessionState = &storage.SessionState{Sessions: make(map[string]storage.SessionInfo)}
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

	// Create session list component
	sessionList := NewSessionList(tmuxClient, store, editor, statusConfig, timestampConfig, devMode, initialMode, keys)

	return &Model{
		devMode:         devMode,
		editor:          editor,
		err:             errMsg,
		errorClearDelay: errorClearDelay,
		keys:            keys,
		sessionList:     sessionList,
		sessionState:    sessionState,
		state:           stateList,
		statusConfig:    statusConfig,
		store:           store,
		timestampConfig: timestampConfig,
		timestampMode:   initialMode,
		tmuxClient:      tmuxClient,
		worktreePath:    worktreePath,
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
		m.clearError()
		return m, nil
	}

	// Handle window size updates
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		// SessionList handles its own sizing via Update()
	}

	// Handle detach message - session list auto-refreshes via polling
	if _, ok := msg.(detachedMsg); ok {
		m.state = stateList
		m.sessionList.RefreshFromState()
		return m, m.sessionList.Init()
	}

	// Hidden test command: alt+shift+e generates Model-level error (persists 5 seconds)
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "alt+E" {
		m.setError(fmt.Errorf("this is a persistent Model-level test error that demonstrates the error display functionality with automatic height adjustment and will clear after five seconds to verify that the list height properly expands back to normal and ensures all session items remain visible throughout the entire error lifecycle"))
		return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
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

	if m.sessionList.SelectedSession != nil {
		session := m.sessionList.SelectedSession
		m.sessionList.SelectedSession = nil // Clear

		// Attach to session using tmux abstraction
		return m, m.attachToSession(session.Name)
	}

	// Check if user selected a shell session
	if m.sessionList.SelectedShellSession != nil {
		session := m.sessionList.SelectedShellSession
		m.sessionList.SelectedShellSession = nil // Reset

		shellSessionName := m.getOrCreateShellSession(session)
		if shellSessionName != "" {
			// Use existing attachToSession() helper (follows tmux abstraction)
			return m, m.attachToSession(shellSessionName)
		}
		return m, cmd
	}

	if m.sessionList.SessionToKill != nil {
		session := m.sessionList.SessionToKill
		m.sessionList.SessionToKill = nil // Clear

		// Check if session has worktree
		if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok && sessionInfo.WorktreePath != "" {
			m.sessionToKill = session
			removeWorktree := false
			m.formRemoveWorktree = &removeWorktree
			m.worktreeRemovalForm = m.createWorktreeRemovalDialog(sessionInfo.WorktreePath)
			m.state = stateConfirmingWorktreeRemoval
			return m, m.worktreeRemovalForm.Init()
		} else {
			return m, m.killSession(session)
		}
	}

	if m.sessionList.SessionToRename != nil {
		session := m.sessionList.SessionToRename
		m.sessionList.SessionToRename = nil // Clear

		// Get current display name
		currentDisplayName := session.Name
		if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok && sessionInfo.DisplayName != "" {
			currentDisplayName = sessionInfo.DisplayName
		}

		contentForm := NewSessionRenameForm(m.tmuxClient, m.store, m.sessionState, session.Name, currentDisplayName)
		m.sessionRenameForm = NewDialog("Rename Session", contentForm, m.devMode)
		m.state = stateRenamingSession
		return m, m.sessionRenameForm.Init()
	}

	if m.sessionList.SessionToSetStatus != nil {
		session := m.sessionList.SessionToSetStatus
		m.sessionList.SessionToSetStatus = nil // Clear

		// Get current status
		var currentStatus *string
		if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok {
			currentStatus = sessionInfo.Status
		}

		contentForm := NewSessionStatusForm(m.store, session.Name, currentStatus, m.statusConfig)
		m.sessionStatusForm = NewDialog("Set Status", contentForm, m.devMode)
		m.state = stateSettingStatus
		return m, m.sessionStatusForm.Init()
	}

	if m.sessionList.SessionToComment != nil {
		session := m.sessionList.SessionToComment
		m.sessionList.SessionToComment = nil // Clear

		// Get current comment
		currentComment := ""
		if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok {
			currentComment = sessionInfo.Comment
		}

		contentForm := NewSessionCommentForm(m.store, session.Name, currentComment)
		m.sessionCommentForm = NewDialog("Edit Session Comment", contentForm, m.devMode)
		m.state = stateCommentingSession
		return m, m.sessionCommentForm.Init()
	}

	if m.sessionList.SessionToOpenEditor != nil {
		session := m.sessionList.SessionToOpenEditor
		m.sessionList.SessionToOpenEditor = nil

		sessionInfo, exists := m.sessionState.Sessions[session.Name]
		if !exists || sessionInfo.WorktreePath == "" {
			m.err = fmt.Errorf("no worktree associated with session '%s'", session.Name)
			return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
		}

		if err := editor.OpenInEditor(sessionInfo.WorktreePath, m.editor); err != nil {
			m.err = fmt.Errorf("failed to open editor: %w", err)
			return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
		}

		return m, m.sessionList.Init()
	}

	if m.sessionList.SessionToToggleFlag != nil {
		session := m.sessionList.SessionToToggleFlag
		m.sessionList.SessionToToggleFlag = nil

		if err := m.store.ToggleFlag(context.Background(), session.Name); err != nil {
			m.err = fmt.Errorf("failed to toggle flag: %w", err)
			return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
		}

		// Reload session state
		sessionState, err := m.store.Load(context.Background(), false)
		if err != nil {
			m.err = fmt.Errorf("failed to refresh sessions: %w", err)
			m.sessionList.RefreshFromState()
			return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
		}
		m.sessionState = sessionState

		// Refresh UI
		m.sessionList.RefreshFromState()
		return m, m.sessionList.Init()
	}

	if m.sessionList.SessionToArchive != nil {
		session := m.sessionList.SessionToArchive
		m.sessionList.SessionToArchive = nil

		// Check if session has worktree
		if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok && sessionInfo.WorktreePath != "" {
			m.sessionToArchive = session
			removeWorktree := false
			m.formRemoveWorktreeArchive = &removeWorktree
			form := m.createArchiveWorktreeRemovalForm(sessionInfo.WorktreePath)
			m.worktreeRemovalForm = NewDialog("Archive Session", form, m.devMode)
			m.state = stateConfirmingArchive
			return m, m.worktreeRemovalForm.Init()
		} else {
			return m, m.archiveSession(session, false)
		}
	}

	if m.sessionList.RequestHelp {
		m.sessionList.RequestHelp = false
		contentForm := NewHelpScreen(&m.keys)
		m.helpScreen = NewDialog("Help", contentForm, m.devMode)
		m.state = stateHelp
		return m, m.helpScreen.Init()
	}

	if m.sessionList.RequestNewSession {
		m.sessionList.RequestNewSession = false
		contentForm := NewSessionForm(m.tmuxClient, m.store, m.worktreePath, m.sessionState)
		m.sessionForm = NewDialog("Create Session", contentForm, m.devMode)
		m.state = stateCreatingSession
		return m, m.sessionForm.Init()
	}

	if m.sessionList.RequestTestError {
		m.sessionList.RequestTestError = false
		m.setError(fmt.Errorf("this is a very long test error message to verify that the error display truncation functionality works correctly and ensures that error text wraps properly across multiple lines and eventually gets truncated with ellipsis if it exceeds the maximum allowed length of three lines which should be enforced by the formatErrorForDisplay function"))
		return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
	}

	return m, cmd
}

func (m *Model) updateCreatingSession(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			m.state = stateList
			m.sessionForm = nil
			return m, m.sessionList.Init()
		}
	}

	// Forward message to Dialog
	updated, cmd := m.sessionForm.Update(msg)
	m.sessionForm = updated.(*Dialog)

	// Access wrapped content to check completion
	if content, ok := m.sessionForm.Content().(*SessionForm); ok {
		if content.Completed {
			result := content.Result()

			// Return to list state
			m.state = stateList
			m.sessionForm = nil

			// Check if session creation failed
			if result.Error != nil {
				m.setError(fmt.Errorf("failed to create session: %w", result.Error))
				return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
			}

			if !result.Cancelled {
				// Reload session state
				sessionState, err := m.store.Load(context.Background(), false)
				if err != nil {
					m.setError(fmt.Errorf("failed to refresh sessions: %w", err))
					log.Printf("Warning: failed to reload session state: %v", err)
					m.sessionList.RefreshFromState()
					return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
				} else {
					m.sessionState = sessionState
				}
				// Refresh session list component
				m.sessionList.RefreshFromState()
				// Select the newly added session (always at position 0)
				m.sessionList.list.Select(0)
			}

			return m, m.sessionList.Init()
		}
	}

	return m, cmd
}

func (m *Model) updateRenamingSession(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			m.state = stateList
			m.sessionRenameForm = nil
			return m, m.sessionList.Init()
		}
	}

	// Forward message to Dialog
	updated, cmd := m.sessionRenameForm.Update(msg)
	m.sessionRenameForm = updated.(*Dialog)

	// Access wrapped content to check completion
	if content, ok := m.sessionRenameForm.Content().(*SessionRenameForm); ok {
		if content.Completed {
			result := content.Result()

			// Return to list state
			m.state = stateList
			m.sessionRenameForm = nil

			// Check if rename failed
			if result.Error != nil {
				m.setError(fmt.Errorf("failed to rename session: %w", result.Error))
				return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
			}

			if !result.Cancelled {
				// Reload session state
				sessionState, err := m.store.Load(context.Background(), false)
				if err != nil {
					m.setError(fmt.Errorf("failed to refresh sessions: %w", err))
					m.sessionList.RefreshFromState()
					return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
				} else {
					m.sessionState = sessionState
				}
				// Refresh session list component
				m.sessionList.RefreshFromState()
			}

			return m, m.sessionList.Init()
		}
	}

	return m, cmd
}

func (m *Model) updateSettingStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			m.state = stateList
			m.sessionStatusForm = nil
			return m, m.sessionList.Init()
		}
	}

	// Forward message to Dialog
	updated, cmd := m.sessionStatusForm.Update(msg)
	m.sessionStatusForm = updated.(*Dialog)

	// Access wrapped content to check completion
	if content, ok := m.sessionStatusForm.Content().(*SessionStatusForm); ok {
		if content.Completed {
			result := content.Result()

			// Return to list state
			m.state = stateList
			m.sessionStatusForm = nil

			// Check if status update failed
			if result.Error != nil {
				m.setError(fmt.Errorf("failed to update status: %w", result.Error))
				return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
			}

			if !result.Cancelled {
				// Reload session state
				sessionState, err := m.store.Load(context.Background(), false)
				if err != nil {
					m.setError(fmt.Errorf("failed to refresh sessions: %w", err))
					m.sessionList.RefreshFromState()
					return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
				} else {
					m.sessionState = sessionState
				}
				// Refresh session list component
				m.sessionList.RefreshFromState()
			}

			return m, m.sessionList.Init()
		}
	}

	return m, cmd
}

func (m *Model) updateCommentingSession(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			m.state = stateList
			m.sessionCommentForm = nil
			return m, m.sessionList.Init()
		}
	}

	// Forward message to Dialog
	updated, cmd := m.sessionCommentForm.Update(msg)
	m.sessionCommentForm = updated.(*Dialog)

	// Access wrapped content to check completion
	if content, ok := m.sessionCommentForm.Content().(*SessionCommentForm); ok {
		if content.Completed {
			result := content.Result()

			// Return to list state
			m.state = stateList
			m.sessionCommentForm = nil

			// Check if comment update failed
			if result.Error != nil {
				m.setError(fmt.Errorf("failed to update comment: %w", result.Error))
				return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
			}

			if !result.Cancelled {
				// Reload session state
				sessionState, err := m.store.Load(context.Background(), false)
				if err != nil {
					m.setError(fmt.Errorf("failed to refresh sessions: %w", err))
					m.sessionList.RefreshFromState()
					return m, tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
				} else {
					m.sessionState = sessionState
				}
				// Refresh session list component
				m.sessionList.RefreshFromState()
			}

			return m, m.sessionList.Init()
		}
	}

	return m, cmd
}

func (m *Model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			m.state = stateList
			m.helpScreen = nil
			return m, m.sessionList.Init()
		}
	}

	// Forward message to Dialog
	updated, cmd := m.helpScreen.Update(msg)
	m.helpScreen = updated.(*Dialog)

	// Access wrapped content to check completion
	if content, ok := m.helpScreen.Content().(*HelpScreen); ok {
		if content.Completed {
			// Return to list state
			m.state = stateList
			m.helpScreen = nil
			return m, m.sessionList.Init()
		}
	}

	return m, cmd
}

type detachedMsg struct{}

type clearErrorMsg struct{}

// clearErrorAfterDelay returns a command that sends clearErrorMsg after the configured delay
func (m *Model) clearErrorAfterDelay() tea.Cmd {
	return tea.Tick(m.errorClearDelay, func(time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

// attachToSession suspends Bubble Tea, attaches to a tmux session via the abstraction layer,
// and returns a detachedMsg when the user detaches
func (m *Model) attachToSession(sessionName string) tea.Cmd {
	logging.Logger.Info("Attaching to session via abstraction layer", "name", sessionName)

	// Get the attach command from the abstraction layer
	cmd := m.tmuxClient.GetAttachCommand(sessionName)

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

// getOrCreateShellSession returns shell session name, creating if needed
// Returns empty string on error (error stored in m.err)
func (m *Model) getOrCreateShellSession(session *tmux.Session) string {
	sessionInfo, ok := m.sessionState.Sessions[session.Name]
	if !ok {
		m.err = fmt.Errorf("session info not found: %s", session.Name)
		return ""
	}

	// Check if shell session already exists
	if sessionInfo.ShellSession != nil {
		// Check if tmux session exists
		if m.tmuxClient.Exists(sessionInfo.ShellSession.Name) {
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
	_, err := m.tmuxClient.CreateShellSession(shellSessionName, workingDir)
	if err != nil {
		m.err = fmt.Errorf("failed to create shell session: %w", err)
		return ""
	}

	// Create nested SessionInfo for shell session
	shellInfo := storage.SessionInfo{
		Name:         shellSessionName,
		ShellSession: nil, // Shell sessions don't have their own shells
		DisplayName:  "",  // No display name for shell sessions
		State:        state.StateIdle,
		ExecutionID:  sessionInfo.ExecutionID,
		LastUpdated:  time.Now().UTC(),
		RepoPath:     sessionInfo.RepoPath,
		RepoInfo:     sessionInfo.RepoInfo,
		BranchName:   sessionInfo.BranchName,
		WorktreePath: sessionInfo.WorktreePath,
	}

	// Update parent session with nested shell info
	sessionInfo.ShellSession = &shellInfo
	m.sessionState.Sessions[session.Name] = sessionInfo

	if err := m.store.Save(context.Background(), m.sessionState); err != nil {
		logging.Logger.Warn("Failed to save shell session to state", "error", err)
	}

	return shellSessionName
}

// killSession kills a session and removes it from state
func (m *Model) killSession(session *tmux.Session) tea.Cmd {
	logging.Logger.Info("Killing session", "name", session.Name)

	// Get session info to check for shell session
	sessionInfo, hasInfo := m.sessionState.Sessions[session.Name]

	// Kill shell session if it exists
	if hasInfo && sessionInfo.ShellSession != nil {
		logging.Logger.Info("Killing shell session", "name", sessionInfo.ShellSession.Name)
		if err := m.tmuxClient.Kill(sessionInfo.ShellSession.Name); err != nil {
			logging.Logger.Warn("Failed to kill shell session", "error", err)
		}
	}

	// Kill main Claude session
	if err := m.tmuxClient.Kill(session.Name); err != nil {
		m.setError(fmt.Errorf("failed to kill session '%s': %w", session.Name, err))
		return tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay()) // Continue polling and clear error after delay
	}

	// Check if session has worktree and remove it from state
	if hasInfo && sessionInfo.WorktreePath != "" {
		logging.Logger.Info("Session had worktree", "path", sessionInfo.WorktreePath)
	}

	// Remove session from state
	st, err := m.store.Load(context.Background(), false)
	if err != nil {
		log.Printf("Warning: failed to load state: %v", err)
	} else {
		delete(st.Sessions, session.Name)
		if err := m.store.Save(context.Background(), st); err != nil {
			log.Printf("Warning: failed to save state: %w", err)
		}
		m.sessionState = st
	}

	// Refresh session list component
	m.sessionList.RefreshFromState()
	return m.sessionList.Init() // Continue polling
}

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
			return m, m.archiveSession(session, removeWorktree)
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
				if err := git.RemoveWorktree(repoPath, worktreePath); err != nil {
					m.setError(fmt.Errorf("failed to remove worktree: %w", err))
					logging.Logger.Error("Failed to remove worktree", "error", err, "path", worktreePath)
					worktreeErr = true
				} else {
					logging.Logger.Info("Worktree removed successfully", "path", worktreePath)
				}
			} else {
				logging.Logger.Info("Keeping worktree", "path", worktreePath)
			}

			// Kill the session
			killCmd := m.killSession(session)

			// Reset state
			m.state = stateList
			m.worktreeRemovalForm = nil
			m.sessionToKill = nil
			m.formRemoveWorktree = nil

			// If there was a worktree error, add clearErrorAfterDelay to the batch
			if worktreeErr {
				return m, tea.Batch(killCmd, m.clearErrorAfterDelay())
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

		// Always reserve 2 lines for errors (keeps layout stable)
		view += "\n"
		if m.err != nil {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
			errorText := formatErrorForDisplay(m.err, m.width)
			view += errorStyle.Render(errorText)
		} else {
			// Empty line to maintain spacing
			view += " "
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
	}
	return ""
}

// setError sets model error.
// Error will be displayed in the reserved 2-line error area.
func (m *Model) setError(err error) {
	m.err = err
}

// clearError clears model error.
func (m *Model) clearError() {
	m.err = nil
}

// archiveSession archives a session and optionally removes its worktree
func (m *Model) archiveSession(session *tmux.Session, removeWorktree bool) tea.Cmd {
	logging.Logger.Info("Archiving session", "name", session.Name, "removeWorktree", removeWorktree)

	// Get session info
	sessionInfo, hasInfo := m.sessionState.Sessions[session.Name]

	// Remove worktree if requested
	if removeWorktree && hasInfo && sessionInfo.WorktreePath != "" {
		logging.Logger.Info("Removing worktree", "path", sessionInfo.WorktreePath, "repo", sessionInfo.RepoPath)
		if err := git.RemoveWorktree(sessionInfo.RepoPath, sessionInfo.WorktreePath); err != nil {
			m.setError(fmt.Errorf("failed to remove worktree, continuing with archive: %w", err))
			logging.Logger.Error("Failed to remove worktree", "error", err, "path", sessionInfo.WorktreePath)
		} else {
			logging.Logger.Info("Worktree removed successfully", "path", sessionInfo.WorktreePath)
		}
	}

	// Toggle archive state
	if err := m.store.ToggleArchive(context.Background(), session.Name); err != nil {
		m.setError(fmt.Errorf("failed to archive session: %w", err))
		return tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
	}

	// Reload session state (showArchived=false, so archived session will disappear)
	sessionState, err := m.store.Load(context.Background(), false)
	if err != nil {
		m.setError(fmt.Errorf("failed to refresh sessions: %w", err))
		m.sessionList.RefreshFromState()
		return tea.Batch(m.sessionList.Init(), m.clearErrorAfterDelay())
	}
	m.sessionState = sessionState

	// Refresh UI
	m.sessionList.RefreshFromState()
	return m.sessionList.Init()
}
