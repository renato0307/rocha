package ui

import (
	"fmt"
	"log"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
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
)

type uiState int

const (
	stateList uiState = iota
	stateCreatingSession
	stateConfirmingWorktreeRemoval
)

type Model struct {
	tmuxClient         tmux.Client
	sessionList        *SessionList   // Session list component
	sessionState       *state.SessionState // State data for git metadata and status
	state              uiState
	width              int
	height             int
	err                error
	worktreePath       string
	form               *huh.Form      // Form for worktree removal confirmation
	sessionForm        *SessionForm   // Session creation form
	sessionToKill      *tmux.Session  // Session being killed (for worktree removal)
	formRemoveWorktree *bool          // Worktree removal decision (pointer to persist across updates)
}

func NewModel(tmuxClient tmux.Client, worktreePath string) *Model {
	// Load session state - this is the source of truth
	sessionState, stateErr := state.Load()
	var errMsg error
	if stateErr != nil {
		log.Printf("Warning: failed to load session state: %v", stateErr)
		errMsg = fmt.Errorf("failed to load state: %w", stateErr)
		sessionState = &state.SessionState{Sessions: make(map[string]state.SessionInfo)}
	}

	// Create session list component
	sessionList := NewSessionList(tmuxClient)

	return &Model{
		tmuxClient:   tmuxClient,
		sessionList:  sessionList,
		sessionState: sessionState,
		state:        stateList,
		err:          errMsg,
		worktreePath: worktreePath,
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
	case stateConfirmingWorktreeRemoval:
		return m.updateConfirmingWorktreeRemoval(msg)
	}
	return m, nil
}

func (m *Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle clear error message
	if _, ok := msg.(clearErrorMsg); ok {
		m.err = nil
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

	if m.sessionList.SessionToKill != nil {
		session := m.sessionList.SessionToKill
		m.sessionList.SessionToKill = nil // Clear

		// Check if session has worktree
		if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok && sessionInfo.WorktreePath != "" {
			m.sessionToKill = session
			removeWorktree := false
			m.formRemoveWorktree = &removeWorktree
			m.form = m.createWorktreeRemovalForm(sessionInfo.WorktreePath)
			m.state = stateConfirmingWorktreeRemoval
			return m, m.form.Init()
		} else {
			return m, m.killSession(session)
		}
	}

	if m.sessionList.RequestNewSession {
		m.sessionList.RequestNewSession = false
		m.sessionForm = NewSessionForm(m.tmuxClient, m.worktreePath, m.sessionState)
		m.state = stateCreatingSession
		return m, m.sessionForm.Init()
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

	// Forward message to SessionForm
	newForm, cmd := m.sessionForm.Update(msg)
	if sf, ok := newForm.(*SessionForm); ok {
		m.sessionForm = sf
	}

	// Check if form is completed
	if m.sessionForm.Completed {
		result := m.sessionForm.Result()

		// Return to list state
		m.state = stateList
		m.sessionForm = nil

		// Check if session creation failed
		if result.Error != nil {
			m.err = fmt.Errorf("failed to create session: %w", result.Error)
			return m, tea.Batch(m.sessionList.Init(), clearErrorAfterDelay())
		}

		if !result.Cancelled {
			// Reload session state
			sessionState, err := state.Load()
			if err != nil {
				m.err = fmt.Errorf("failed to refresh sessions: %w", err)
				log.Printf("Warning: failed to reload session state: %v", err)
				m.sessionList.RefreshFromState()
				return m, tea.Batch(m.sessionList.Init(), clearErrorAfterDelay())
			} else {
				m.sessionState = sessionState
			}
			// Refresh session list component
			m.sessionList.RefreshFromState()
		}

		return m, m.sessionList.Init()
	}

	return m, cmd
}

type detachedMsg struct{}

type clearErrorMsg struct{}

// clearErrorAfterDelay returns a command that sends clearErrorMsg after a delay
func clearErrorAfterDelay() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
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

// killSession kills a session and removes it from state
func (m *Model) killSession(session *tmux.Session) tea.Cmd {
	logging.Logger.Info("Killing session", "name", session.Name)

	if err := m.tmuxClient.Kill(session.Name); err != nil {
		m.err = fmt.Errorf("failed to kill session '%s': %w", session.Name, err)
		return tea.Batch(m.sessionList.Init(), clearErrorAfterDelay()) // Continue polling and clear error after delay
	}

	// Check if session has worktree and remove it from state
	if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok && sessionInfo.WorktreePath != "" {
		logging.Logger.Info("Session had worktree", "path", sessionInfo.WorktreePath)
	}

	// Remove session from state
	st, err := state.Load()
	if err != nil {
		log.Printf("Warning: failed to load state: %v", err)
	} else {
		if err := st.RemoveSession(session.Name); err != nil {
			log.Printf("Warning: failed to remove session from state: %v", err)
		}
		m.sessionState = st
	}

	// Refresh session list component
	m.sessionList.RefreshFromState()
	return m.sessionList.Init() // Continue polling
}

func (m *Model) updateConfirmingWorktreeRemoval(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			m.state = stateList
			m.form = nil
			m.sessionToKill = nil
			m.formRemoveWorktree = nil
			return m, nil
		}
	}

	// Safety check for nil form
	if m.form == nil {
		m.state = stateList
		m.sessionToKill = nil
		return m, nil
	}

	// Forward message to form
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	// Check if form completed
	if m.form.State == huh.StateCompleted {
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
				m.err = fmt.Errorf("failed to remove worktree: %w", err)
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
		m.form = nil
		m.sessionToKill = nil
		m.formRemoveWorktree = nil

		// If there was a worktree error, add clearErrorAfterDelay to the batch
		if worktreeErr {
			return m, tea.Batch(killCmd, clearErrorAfterDelay())
		}
		return m, killCmd
	}

	return m, cmd
}

// createWorktreeRemovalForm creates a confirmation form for removing a worktree
func (m *Model) createWorktreeRemovalForm(worktreePath string) *huh.Form {
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

	return form
}

func (m *Model) View() string {
	switch m.state {
	case stateList:
		view := m.sessionList.View()

		// Display model-level errors (e.g., from killSession failures)
		if m.err != nil {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
			view += "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		}

		return view
	case stateCreatingSession:
		if m.sessionForm != nil {
			return m.sessionForm.View()
		}
	case stateConfirmingWorktreeRemoval:
		if m.form != nil {
			return m.form.View()
		}
	}
	return ""
}
