package ui

import (
	"fmt"
	"log"
	"os/exec"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/tmux"
	"strings"

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
				Foreground(lipgloss.Color("2")) // Green

	waitingIconStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("3")) // Yellow
)

type uiState int

const (
	stateList uiState = iota
	stateCreatingSession
	stateConfirmingWorktreeRemoval
)

type Model struct {
	sessions           []*tmux.Session
	sessionState       *state.SessionState // State data for git metadata and status
	cursor             int
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

func NewModel(worktreePath string) Model {
	sessions, err := tmux.List()
	var errMsg error
	if err != nil {
		errMsg = fmt.Errorf("failed to load sessions: %w", err)
		sessions = []*tmux.Session{}
	}

	// Load session state for git metadata
	sessionState, stateErr := state.Load()
	if stateErr != nil {
		log.Printf("Warning: failed to load session state: %v", stateErr)
		sessionState = &state.SessionState{Sessions: make(map[string]state.SessionInfo)}
	}

	return Model{
		sessions:     sessions,
		sessionState: sessionState,
		cursor:       0,
		state:        stateList,
		err:          errMsg,
		worktreePath: worktreePath,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		m.err = msg
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}

		case "n":
			// Create session creation form
			m.sessionForm = NewSessionForm(m.worktreePath, m.sessionState)
			m.state = stateCreatingSession
			return m, m.sessionForm.Init()

		case "enter":
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				// Use tea.ExecProcess to suspend Bubble Tea and attach to tmux
				session := m.sessions[m.cursor]
				c := exec.Command("tmux", "attach-session", "-t", session.Name)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					if err != nil {
						return err
					}
					return detachedMsg{}
				})
			}

		case "x":
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]

				// Check if session has a worktree
				if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok && sessionInfo.WorktreePath != "" {
					// Session has a worktree, show confirmation form
					logging.Logger.Info("Session has worktree, showing removal confirmation", "session", session.Name, "worktree", sessionInfo.WorktreePath)
					m.sessionToKill = session
					removeWorktree := false
					m.formRemoveWorktree = &removeWorktree // Create pointer to persist with form
					m.form = m.createWorktreeRemovalForm(sessionInfo.WorktreePath)
					m.state = stateConfirmingWorktreeRemoval
					return m, m.form.Init()
				} else {
					// No worktree, just kill the session
					m.killSession(session)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case detachedMsg:
		// Returned from attached state
		m.state = stateList
		// Refresh session list after detaching
		sessions, err := tmux.List()
		if err != nil {
			m.err = fmt.Errorf("failed to refresh sessions: %w", err)
		} else {
			m.sessions = sessions
			if m.cursor >= len(m.sessions) {
				m.cursor = len(m.sessions) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		}

	}

	return m, nil
}

func (m Model) updateCreatingSession(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			m.state = stateList
			m.sessionForm = nil
			return m, nil
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

		if !result.Cancelled {
			// Refresh session list
			sessions, err := tmux.List()
			if err != nil {
				m.err = fmt.Errorf("failed to refresh sessions: %w", err)
			} else {
				m.sessions = sessions
				m.cursor = len(m.sessions) - 1 // Jump to newly created session
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			// Reload session state
			sessionState, err := state.Load()
			if err != nil {
				log.Printf("Warning: failed to reload session state: %v", err)
			} else {
				m.sessionState = sessionState
			}
		}

		return m, nil
	}

	return m, cmd
}

type detachedMsg struct{}

// killSession kills a session and removes it from state
func (m *Model) killSession(session *tmux.Session) {
	logging.Logger.Info("Killing session", "name", session.Name)

	if err := session.Kill(); err != nil {
		m.err = err
		return
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
	}

	// Remove from session list
	for i, s := range m.sessions {
		if s.Name == session.Name {
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			if m.cursor >= len(m.sessions) && m.cursor > 0 {
				m.cursor--
			}
			break
		}
	}
}

func (m Model) updateConfirmingWorktreeRemoval(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if removeWorktree {
			logging.Logger.Info("Removing worktree", "path", worktreePath, "repo", repoPath)
			if err := git.RemoveWorktree(repoPath, worktreePath); err != nil {
				m.err = fmt.Errorf("failed to remove worktree: %w", err)
				logging.Logger.Error("Failed to remove worktree", "error", err, "path", worktreePath)
			} else {
				logging.Logger.Info("Worktree removed successfully", "path", worktreePath)
			}
		} else {
			logging.Logger.Info("Keeping worktree", "path", worktreePath)
		}

		// Kill the session
		m.killSession(session)

		// Reset state
		m.state = stateList
		m.form = nil
		m.sessionToKill = nil
		m.formRemoveWorktree = nil

		return m, nil
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

func (m Model) View() string {
	switch m.state {
	case stateList:
		return m.viewList()
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

func (m Model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rocha - Claude Code Session Manager"))
	b.WriteString("\n\n")

	if len(m.sessions) == 0 {
		b.WriteString(normalStyle.Render("No Claude Code sessions yet. Press 'n' to create one."))
	} else {
		for i, session := range m.sessions {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}

			// Get display name from state, fallback to tmux name
			displayName := session.Name
			var gitRef string
			var sessionState string

			if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok {
				if sessionInfo.DisplayName != "" {
					displayName = sessionInfo.DisplayName
				}
				// Build git reference in standard format: owner/repo:branch
				if sessionInfo.RepoInfo != "" && sessionInfo.BranchName != "" {
					gitRef = fmt.Sprintf("%s:%s", sessionInfo.RepoInfo, sessionInfo.BranchName)
				} else if sessionInfo.BranchName != "" {
					// Fallback to just branch name if no repo info
					gitRef = sessionInfo.BranchName
				}
				sessionState = sessionInfo.State
			}

			// Build session line with cursor indicator
			line := fmt.Sprintf("%s %d. %s", cursor, i+1, displayName)
			b.WriteString(normalStyle.Render(line))

			// Add git reference if available
			if gitRef != "" {
				b.WriteString(branchStyle.Render(fmt.Sprintf(" (%s)", gitRef)))
			}

			// Add status icon
			switch sessionState {
			case state.StateWorking:
				b.WriteString(" " + workingIconStyle.Render("●"))
			case state.StateWaiting:
				b.WriteString(" " + waitingIconStyle.Render("○"))
			}

			b.WriteString("\n")
		}
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", m.err)))
		m.err = nil // Clear error after showing
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("↑/k: up • ↓/j: down • n: new • enter: attach (Ctrl+B D or Ctrl+Q to detach) • x: kill • q: quit"))

	return b.String()
}
