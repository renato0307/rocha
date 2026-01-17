package ui

import (
	"fmt"
	"log"
	"os/exec"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/tmux"
	"rocha/version"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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
	stateFiltering
)

const escTimeout = 500 * time.Millisecond

type Model struct {
	tmuxClient         tmux.Client
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

	// Filter fields
	filterInput      textinput.Model
	filterText       string
	filteredSessions []*tmux.Session
	escPressCount    int
	escPressTime     time.Time
}

func NewModel(tmuxClient tmux.Client, worktreePath string) Model {
	// Load session state - this is the source of truth
	sessionState, stateErr := state.Load()
	var errMsg error
	if stateErr != nil {
		log.Printf("Warning: failed to load session state: %v", stateErr)
		errMsg = fmt.Errorf("failed to load state: %w", stateErr)
		sessionState = &state.SessionState{Sessions: make(map[string]state.SessionInfo)}
	}

	// Create session list from state (source of truth)
	sessions := sessionsFromState(sessionState)

	// Initialize filter input
	filterInput := textinput.New()
	filterInput.Placeholder = "Type to filter sessions..."
	filterInput.CharLimit = 100
	filterInput.Width = 50

	return Model{
		tmuxClient:       tmuxClient,
		sessions:         sessions,
		sessionState:     sessionState,
		cursor:           0,
		state:            stateList,
		err:              errMsg,
		worktreePath:     worktreePath,
		filterInput:      filterInput,
		filteredSessions: sessions, // Initially show all
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
	case stateFiltering:
		return m.updateFiltering(msg)
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

		case "esc":
			// ESC×2 to clear filter when filter is active
			if m.filterText != "" {
				now := time.Now()
				if now.Sub(m.escPressTime) < escTimeout && m.escPressCount >= 1 {
					// Second ESC - clear filter
					m.clearFilter()
					return m, nil
				}
				// First ESC
				m.escPressCount = 1
				m.escPressTime = now
			}
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			displaySessions := m.sessions
			if m.filterText != "" {
				displaySessions = m.filteredSessions
			}
			if m.cursor < len(displaySessions)-1 {
				m.cursor++
			}

		case "n":
			// Create session creation form
			m.sessionForm = NewSessionForm(m.tmuxClient, m.worktreePath, m.sessionState)
			m.state = stateCreatingSession
			return m, m.sessionForm.Init()

		case "/":
			m.state = stateFiltering
			m.filterInput.Focus()
			m.filterInput.SetValue(m.filterText) // Restore previous filter
			return m, textinput.Blink

		case "enter":
			displaySessions := m.sessions
			if m.filterText != "" {
				displaySessions = m.filteredSessions
			}

			if len(displaySessions) > 0 && m.cursor < len(displaySessions) {
				session := displaySessions[m.cursor]

				// Ensure session exists (recreate if needed for race condition protection)
				if !m.ensureSessionExists(session) {
					return m, nil
				}

				// Use tea.ExecProcess to suspend Bubble Tea and attach to tmux
				c := exec.Command("tmux", "attach-session", "-t", session.Name)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					if err != nil {
						return err
					}
					return detachedMsg{}
				})
			}

		case "x":
			displaySessions := m.sessions
			if m.filterText != "" {
				displaySessions = m.filteredSessions
			}

			if len(displaySessions) > 0 && m.cursor < len(displaySessions) {
				session := displaySessions[m.cursor]

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

		case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7":
			// Quick attach to session by number
			displaySessions := m.sessions
			if m.filterText != "" {
				displaySessions = m.filteredSessions
			}

			// Extract number from key (alt+1 -> '1' -> 1)
			numStr := msg.String()[4:] // Skip "alt+"
			num := int(numStr[0] - '0')
			index := num - 1 // Convert to 0-based index

			if index >= 0 && index < len(displaySessions) {
				session := displaySessions[index]

				// Ensure session exists (recreate if needed for race condition protection)
				if !m.ensureSessionExists(session) {
					return m, nil
				}

				c := exec.Command("tmux", "attach-session", "-t", session.Name)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					if err != nil {
						return err
					}
					return detachedMsg{}
				})
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case detachedMsg:
		// Returned from attached state
		m.state = stateList
		// Refresh session list from state (source of truth)
		sessionState, err := state.Load()
		if err != nil {
			m.err = fmt.Errorf("failed to refresh sessions: %w", err)
		} else {
			m.sessionState = sessionState
			m.sessions = sessionsFromState(sessionState)

			// Recompute filtered sessions
			if m.filterText != "" {
				m.filteredSessions = m.filterSessions()
			} else {
				m.filteredSessions = m.sessions
			}

			// Adjust cursor with filtered sessions
			displaySessions := m.sessions
			if m.filterText != "" {
				displaySessions = m.filteredSessions
			}
			if m.cursor >= len(displaySessions) {
				m.cursor = len(displaySessions) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		}

	}

	return m, nil
}

func (m Model) updateFiltering(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			// Double-ESC detection
			now := time.Now()
			if now.Sub(m.escPressTime) < escTimeout && m.escPressCount >= 1 {
				// Second ESC - clear filter and exit
				m.clearFilter()
				m.state = stateList
				return m, nil
			}
			// First ESC
			m.escPressCount = 1
			m.escPressTime = now
			return m, nil

		case "enter":
			// Apply filter and return to list
			m.state = stateList
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.filteredSessions)-1 {
				m.cursor++
			}

		case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7":
			// Quick attach to session by number
			numStr := msg.String()[4:] // Skip "alt+"
			num := int(numStr[0] - '0')
			index := num - 1 // Convert to 0-based index

			if index >= 0 && index < len(m.filteredSessions) {
				session := m.filteredSessions[index]
				c := exec.Command("tmux", "attach-session", "-t", session.Name)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					if err != nil {
						return err
					}
					return detachedMsg{}
				})
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update filter input and refilter
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.updateFilterText(m.filterInput.Value())

	return m, cmd
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

		// Check if session creation failed
		if result.Error != nil {
			m.err = fmt.Errorf("failed to create session: %w", result.Error)
			return m, nil
		}

		if !result.Cancelled {
			// Reload session state (source of truth)
			sessionState, err := state.Load()
			if err != nil {
				m.err = fmt.Errorf("failed to refresh sessions: %w", err)
				log.Printf("Warning: failed to reload session state: %v", err)
			} else {
				m.sessionState = sessionState
				m.sessions = sessionsFromState(sessionState)
				m.cursor = len(m.sessions) - 1 // Jump to newly created session
				if m.cursor < 0 {
					m.cursor = 0
				}
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

	if err := m.tmuxClient.Kill(session.Name); err != nil {
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

// sessionsFromState rebuilds the session list from state.json
func sessionsFromState(sessionState *state.SessionState) []*tmux.Session {
	var sessions []*tmux.Session
	for name, info := range sessionState.Sessions {
		sessions = append(sessions, &tmux.Session{
			Name:      name,
			CreatedAt: info.LastUpdated,
		})
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Name < sessions[j].Name
	})
	return sessions
}

// ensureSessionExists checks if a session exists and recreates it if needed
// Returns true if session is ready to attach, false if recreation failed
func (m *Model) ensureSessionExists(session *tmux.Session) bool {
	if m.tmuxClient.Exists(session.Name) {
		return true
	}

	logging.Logger.Info("Session no longer exists, recreating", "name", session.Name)

	// Try to get stored metadata to recreate with same worktree
	var worktreePath string
	if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok {
		worktreePath = sessionInfo.WorktreePath
		logging.Logger.Info("Recreating session with stored worktree", "name", session.Name, "worktree", worktreePath)
	} else {
		logging.Logger.Warn("No stored metadata for session, creating without worktree", "name", session.Name)
	}

	// Recreate the session
	if _, err := m.tmuxClient.Create(session.Name, worktreePath); err != nil {
		m.err = fmt.Errorf("failed to recreate session: %w", err)
		return false
	}

	return true
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

func (m *Model) clearFilter() {
	m.filterText = ""
	m.filterInput.SetValue("")
	m.filteredSessions = m.sessions
	m.cursor = 0
	m.escPressCount = 0
}

func (m *Model) updateFilterText(newText string) {
	if m.filterText != newText {
		m.filterText = newText
		m.filteredSessions = m.filterSessions()

		// Reset and bound cursor
		m.cursor = 0
		if len(m.filteredSessions) > 0 && m.cursor >= len(m.filteredSessions) {
			m.cursor = len(m.filteredSessions) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

func (m Model) filterSessions() []*tmux.Session {
	if m.filterText == "" {
		return m.sessions
	}

	filterLower := strings.ToLower(m.filterText)
	var filtered []*tmux.Session

	for _, session := range m.sessions {
		sessionInfo, ok := m.sessionState.Sessions[session.Name]

		// Build searchable text
		var searchText strings.Builder
		searchText.WriteString(strings.ToLower(session.Name))
		if ok {
			if sessionInfo.DisplayName != "" {
				searchText.WriteString(" ")
				searchText.WriteString(strings.ToLower(sessionInfo.DisplayName))
			}
			if sessionInfo.RepoInfo != "" {
				searchText.WriteString(" ")
				searchText.WriteString(strings.ToLower(sessionInfo.RepoInfo))
			}
			if sessionInfo.BranchName != "" {
				searchText.WriteString(" ")
				searchText.WriteString(strings.ToLower(sessionInfo.BranchName))
			}
		}

		if strings.Contains(searchText.String(), filterLower) {
			filtered = append(filtered, session)
		}
	}

	return filtered
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
	case stateFiltering:
		return m.viewFiltering()
	}
	return ""
}

func (m Model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rocha"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render(version.Tagline))
	b.WriteString("\n\n")

	// Use filtered sessions if filter is active
	displaySessions := m.sessions
	if m.filterText != "" {
		displaySessions = m.filteredSessions
	}

	if len(displaySessions) == 0 {
		if m.filterText != "" {
			b.WriteString(normalStyle.Render("No sessions match filter. Press ESC twice to clear."))
		} else {
			b.WriteString(normalStyle.Render("No Claude Code sessions yet. Press 'n' to create one."))
		}
	} else {
		for i, session := range displaySessions {
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

	var helpText string
	if m.filterText != "" {
		helpText = fmt.Sprintf("🔍 Filter: %s • ESC×2: clear\n", m.filterText)
	}
	helpText += "↑/k: up • ↓/j: down • /: filter • n: new\n"
	helpText += "enter/Alt+1-7: attach (Ctrl+B D or Ctrl+Q to detach) • x: kill • q: quit"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

func (m Model) viewFiltering() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rocha - Filter Sessions"))
	b.WriteString("\n\n")
	b.WriteString(m.filterInput.View())
	b.WriteString("\n\n")

	resultCount := len(m.filteredSessions)
	totalCount := len(m.sessions)
	countText := fmt.Sprintf("Showing %d of %d sessions", resultCount, totalCount)
	b.WriteString(branchStyle.Render(countText))
	b.WriteString("\n\n")

	if resultCount == 0 {
		b.WriteString(normalStyle.Render("No sessions match filter."))
	} else {
		for i, session := range m.filteredSessions {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}

			displayName := session.Name
			var gitRef string
			var sessionState string

			if sessionInfo, ok := m.sessionState.Sessions[session.Name]; ok {
				if sessionInfo.DisplayName != "" {
					displayName = sessionInfo.DisplayName
				}
				if sessionInfo.RepoInfo != "" && sessionInfo.BranchName != "" {
					gitRef = fmt.Sprintf("%s:%s", sessionInfo.RepoInfo, sessionInfo.BranchName)
				} else if sessionInfo.BranchName != "" {
					gitRef = sessionInfo.BranchName
				}
				sessionState = sessionInfo.State
			}

			line := fmt.Sprintf("%s %d. %s", cursor, i+1, displayName)
			b.WriteString(normalStyle.Render(line))

			if gitRef != "" {
				b.WriteString(branchStyle.Render(fmt.Sprintf(" (%s)", gitRef)))
			}

			switch sessionState {
			case state.StateWorking:
				b.WriteString(" " + workingIconStyle.Render("●"))
			case state.StateWaiting:
				b.WriteString(" " + waitingIconStyle.Render("○"))
			}

			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")
	helpText := "Type to filter • ↑/↓: navigate • enter/Alt+1-7: apply/attach\n"
	helpText += "ESC×2: clear • Ctrl+C: quit"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}
