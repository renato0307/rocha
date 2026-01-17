package ui

import (
	"fmt"
	"rocha/logging"
	"rocha/state"
	"rocha/tmux"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const escTimeout = 500 * time.Millisecond

// Internal messages for SessionList
type checkStateMsg struct{}          // Triggers state file check
type sessionListDetachedMsg struct{} // Session list returned from attached state

// SessionList is a Bubble Tea component for displaying and managing sessions
type SessionList struct {
	tmuxClient       tmux.Client
	sessions         []*tmux.Session
	sessionState     *state.SessionState
	cursor           int
	err              error

	// Filter fields
	filterInput      textinput.Model
	filterText       string
	filteredSessions []*tmux.Session
	isFiltering      bool
	escPressCount    int
	escPressTime     time.Time

	// Dimensions
	width  int
	height int

	// Result fields - set by component, read by Model
	SelectedSession   *tmux.Session // Session user wants to attach to
	SessionToKill     *tmux.Session // Session user wants to kill
	RequestNewSession bool          // User pressed 'n'
	ShouldQuit        bool          // User pressed 'q' or Ctrl+C
}

// NewSessionList creates a new session list component
func NewSessionList(tmuxClient tmux.Client) *SessionList {
	// Load session state
	sessionState, err := state.Load()
	if err != nil {
		logging.Logger.Warn("Failed to load session state", "error", err)
		sessionState = &state.SessionState{Sessions: make(map[string]state.SessionInfo)}
	}

	sessions := sessionsFromState(sessionState)

	// Initialize filter input
	filterInput := textinput.New()
	filterInput.Placeholder = "Type to filter sessions..."
	filterInput.CharLimit = 100
	filterInput.Width = 50

	return &SessionList{
		tmuxClient:       tmuxClient,
		sessions:         sessions,
		sessionState:     sessionState,
		cursor:           0,
		filterInput:      filterInput,
		filteredSessions: sessions,
		err:              err,
	}
}

// Init starts the session list component, including auto-refresh polling
func (sl *SessionList) Init() tea.Cmd {
	return pollStateCmd() // Start auto-refresh polling
}

// Update handles messages for the session list component
func (sl *SessionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sl.isFiltering {
		return sl.updateFiltering(msg)
	}
	return sl.updateList(msg)
}

// View renders the session list component
func (sl *SessionList) View() string {
	if sl.isFiltering {
		return sl.viewFiltering()
	}
	return sl.viewList()
}

// RefreshFromState reloads the session list from state
func (sl *SessionList) RefreshFromState() {
	sessionState, err := state.Load()
	if err != nil {
		sl.err = fmt.Errorf("failed to refresh sessions: %w", err)
		logging.Logger.Error("Failed to refresh session state", "error", err)
		return
	}

	sl.sessionState = sessionState
	sl.sessions = sessionsFromState(sessionState)

	// Recompute filtered sessions
	if sl.filterText != "" {
		sl.filteredSessions = sl.filterSessions()
	} else {
		sl.filteredSessions = sl.sessions
	}

	// Adjust cursor
	displaySessions := sl.sessions
	if sl.filterText != "" {
		displaySessions = sl.filteredSessions
	}
	sl.adjustCursor(len(displaySessions))
}

// pollStateCmd returns a command that waits 2 seconds then sends checkStateMsg
func pollStateCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return checkStateMsg{}
	})
}

// updateList handles messages when in list mode
func (sl *SessionList) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case checkStateMsg:
		// Auto-refresh: Check if state file has changed
		newState, err := state.Load()
		if err != nil {
			// Continue polling even on error
			return sl, pollStateCmd()
		}

		// Only update if state actually changed
		if !newState.UpdatedAt.Equal(sl.sessionState.UpdatedAt) {
			sl.sessionState = newState
			sl.sessions = sessionsFromState(newState)

			// Recompute filtered sessions if filter active
			if sl.filterText != "" {
				sl.filteredSessions = sl.filterSessions()
				sl.adjustCursor(len(sl.filteredSessions))
			} else {
				sl.filteredSessions = sl.sessions
				sl.adjustCursor(len(sl.sessions))
			}
		}

		return sl, pollStateCmd()

	case error:
		sl.err = msg
		return sl, pollStateCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			sl.ShouldQuit = true
			return sl, nil

		case "esc":
			// ESC×2 to clear filter when filter is active
			if sl.filterText != "" {
				now := time.Now()
				if now.Sub(sl.escPressTime) < escTimeout && sl.escPressCount >= 1 {
					// Second ESC - clear filter
					sl.clearFilter()
					return sl, pollStateCmd()
				}
				// First ESC
				sl.escPressCount = 1
				sl.escPressTime = now
			}
			return sl, pollStateCmd()

		case "up", "k":
			if sl.cursor > 0 {
				sl.cursor--
			}

		case "down", "j":
			displaySessions := sl.getDisplaySessions()
			if sl.cursor < len(displaySessions)-1 {
				sl.cursor++
			}

		case "n":
			sl.RequestNewSession = true
			return sl, nil

		case "/":
			sl.isFiltering = true
			sl.filterInput.Focus()
			sl.filterInput.SetValue(sl.filterText) // Restore previous filter
			return sl, textinput.Blink

		case "enter":
			displaySessions := sl.getDisplaySessions()
			if len(displaySessions) > 0 && sl.cursor < len(displaySessions) {
				session := displaySessions[sl.cursor]

				// Ensure session exists (recreate if needed for race condition protection)
				if !sl.ensureSessionExists(session) {
					return sl, pollStateCmd()
				}

				sl.SelectedSession = session
				return sl, nil
			}

		case "x":
			displaySessions := sl.getDisplaySessions()
			if len(displaySessions) > 0 && sl.cursor < len(displaySessions) {
				session := displaySessions[sl.cursor]
				sl.SessionToKill = session
				return sl, nil
			}

		case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7":
			// Quick attach to session by number
			displaySessions := sl.getDisplaySessions()

			// Extract number from key (alt+1 -> '1' -> 1)
			numStr := msg.String()[4:] // Skip "alt+"
			num := int(numStr[0] - '0')
			index := num - 1 // Convert to 0-based index

			if index >= 0 && index < len(displaySessions) {
				session := displaySessions[index]

				// Ensure session exists (recreate if needed for race condition protection)
				if !sl.ensureSessionExists(session) {
					return sl, pollStateCmd()
				}

				sl.SelectedSession = session
				return sl, nil
			}
		}

	case tea.WindowSizeMsg:
		sl.width = msg.Width
		sl.height = msg.Height
	}

	return sl, pollStateCmd()
}

// updateFiltering handles messages when in filtering mode
func (sl *SessionList) updateFiltering(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case checkStateMsg:
		// Continue polling even in filtering mode
		newState, err := state.Load()
		if err != nil {
			return sl, tea.Batch(textinput.Blink, pollStateCmd())
		}

		// Only update if state actually changed
		if !newState.UpdatedAt.Equal(sl.sessionState.UpdatedAt) {
			sl.sessionState = newState
			sl.sessions = sessionsFromState(newState)
			sl.filteredSessions = sl.filterSessions()
			sl.adjustCursor(len(sl.filteredSessions))
		}

		return sl, tea.Batch(textinput.Blink, pollStateCmd())

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			sl.ShouldQuit = true
			return sl, nil

		case "esc":
			// Double-ESC detection
			now := time.Now()
			if now.Sub(sl.escPressTime) < escTimeout && sl.escPressCount >= 1 {
				// Second ESC - clear filter and exit
				sl.clearFilter()
				sl.isFiltering = false
				return sl, pollStateCmd()
			}
			// First ESC
			sl.escPressCount = 1
			sl.escPressTime = now
			return sl, textinput.Blink

		case "enter":
			// Apply filter and return to list
			sl.isFiltering = false
			return sl, pollStateCmd()

		case "up", "k":
			if sl.cursor > 0 {
				sl.cursor--
			}

		case "down", "j":
			if sl.cursor < len(sl.filteredSessions)-1 {
				sl.cursor++
			}

		case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7":
			// Quick attach to session by number
			numStr := msg.String()[4:] // Skip "alt+"
			num := int(numStr[0] - '0')
			index := num - 1 // Convert to 0-based index

			if index >= 0 && index < len(sl.filteredSessions) {
				session := sl.filteredSessions[index]

				// Ensure session exists
				if !sl.ensureSessionExists(session) {
					return sl, tea.Batch(textinput.Blink, pollStateCmd())
				}

				sl.SelectedSession = session
				return sl, nil
			}
		}

	case tea.WindowSizeMsg:
		sl.width = msg.Width
		sl.height = msg.Height
	}

	// Update filter input and refilter
	sl.filterInput, cmd = sl.filterInput.Update(msg)
	sl.updateFilterText(sl.filterInput.Value())

	return sl, tea.Batch(cmd, pollStateCmd())
}

// viewList renders the session list view
func (sl *SessionList) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rocha"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("Claude Code session manager"))
	b.WriteString("\n\n")

	displaySessions := sl.getDisplaySessions()

	if len(displaySessions) == 0 {
		if sl.filterText != "" {
			b.WriteString(normalStyle.Render("No sessions match filter. Press ESC twice to clear."))
		} else {
			b.WriteString(normalStyle.Render("No Claude Code sessions yet. Press 'n' to create one."))
		}
	} else {
		for i, session := range displaySessions {
			cursor := " "
			if i == sl.cursor {
				cursor = ">"
			}

			// Get display name from state, fallback to tmux name
			displayName := session.Name
			var gitRef string
			var sessionState string

			if sessionInfo, ok := sl.sessionState.Sessions[session.Name]; ok {
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
				b.WriteString(" " + workingIconStyle.Render(state.SymbolWorking))
			case state.StateIdle:
				b.WriteString(" " + idleIconStyle.Render(state.SymbolIdle))
			case state.StateWaitingUser:
				b.WriteString(" " + waitingIconStyle.Render(state.SymbolWaitingUser))
			}

			b.WriteString("\n")
		}
	}

	if sl.err != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", sl.err)))
		sl.err = nil // Clear error after showing
	}

	b.WriteString("\n\n")

	var helpText string
	if sl.filterText != "" {
		helpText = fmt.Sprintf("🔍 Filter: %s • ESC×2: clear\n", sl.filterText)
	}
	helpText += "↑/k: up • ↓/j: down • /: filter • n: new\n"
	helpText += "enter/Alt+1-7: attach (Ctrl+B D or Ctrl+Q to detach) • x: kill • q: quit\n\n"

	// Add legend for status symbols
	helpText += "Status: "
	helpText += workingIconStyle.Render(state.SymbolWorking) + " working • "
	helpText += idleIconStyle.Render(state.SymbolIdle) + " idle • "
	helpText += waitingIconStyle.Render(state.SymbolWaitingUser) + " waiting for input"

	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// viewFiltering renders the filtering view
func (sl *SessionList) viewFiltering() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rocha - Filter Sessions"))
	b.WriteString("\n\n")
	b.WriteString(sl.filterInput.View())
	b.WriteString("\n\n")

	resultCount := len(sl.filteredSessions)
	totalCount := len(sl.sessions)
	countText := fmt.Sprintf("Showing %d of %d sessions", resultCount, totalCount)
	b.WriteString(branchStyle.Render(countText))
	b.WriteString("\n\n")

	if resultCount == 0 {
		b.WriteString(normalStyle.Render("No sessions match filter."))
	} else {
		for i, session := range sl.filteredSessions {
			cursor := " "
			if i == sl.cursor {
				cursor = ">"
			}

			displayName := session.Name
			var gitRef string
			var sessionState string

			if sessionInfo, ok := sl.sessionState.Sessions[session.Name]; ok {
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
				b.WriteString(" " + workingIconStyle.Render(state.SymbolWorking))
			case state.StateIdle:
				b.WriteString(" " + idleIconStyle.Render(state.SymbolIdle))
			case state.StateWaitingUser:
				b.WriteString(" " + waitingIconStyle.Render(state.SymbolWaitingUser))
			}

			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")
	helpText := "Type to filter • ↑/↓: navigate • enter/Alt+1-7: apply/attach\n"
	helpText += "ESC×2: clear • Ctrl+C: quit\n\n"

	// Add legend for status symbols
	helpText += "Status: "
	helpText += workingIconStyle.Render(state.SymbolWorking) + " working • "
	helpText += idleIconStyle.Render(state.SymbolIdle) + " idle • "
	helpText += waitingIconStyle.Render(state.SymbolWaitingUser) + " waiting for input"

	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// Helper functions

// getDisplaySessions returns the appropriate session list based on filter state
func (sl *SessionList) getDisplaySessions() []*tmux.Session {
	if sl.filterText != "" {
		return sl.filteredSessions
	}
	return sl.sessions
}

// adjustCursor ensures cursor is within valid range
func (sl *SessionList) adjustCursor(maxLen int) {
	if maxLen == 0 {
		sl.cursor = 0
		return
	}
	if sl.cursor >= maxLen {
		sl.cursor = maxLen - 1
	}
	if sl.cursor < 0 {
		sl.cursor = 0
	}
}

// clearFilter clears the filter state
func (sl *SessionList) clearFilter() {
	sl.filterText = ""
	sl.filterInput.SetValue("")
	sl.filteredSessions = sl.sessions
	sl.cursor = 0
	sl.escPressCount = 0
}

// updateFilterText updates the filter text and recomputes filtered sessions
func (sl *SessionList) updateFilterText(newText string) {
	if sl.filterText != newText {
		sl.filterText = newText
		sl.filteredSessions = sl.filterSessions()

		// Reset and bound cursor
		sl.cursor = 0
		if len(sl.filteredSessions) > 0 && sl.cursor >= len(sl.filteredSessions) {
			sl.cursor = len(sl.filteredSessions) - 1
		}
		if sl.cursor < 0 {
			sl.cursor = 0
		}
	}
}

// filterSessions filters sessions based on filter text
func (sl *SessionList) filterSessions() []*tmux.Session {
	if sl.filterText == "" {
		return sl.sessions
	}

	filterLower := strings.ToLower(sl.filterText)
	var filtered []*tmux.Session

	for _, session := range sl.sessions {
		sessionInfo, ok := sl.sessionState.Sessions[session.Name]

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

// ensureSessionExists checks if a session exists and recreates it if needed
// Returns true if session is ready to attach, false if recreation failed
func (sl *SessionList) ensureSessionExists(session *tmux.Session) bool {
	if sl.tmuxClient.Exists(session.Name) {
		return true
	}

	logging.Logger.Info("Session no longer exists, recreating", "name", session.Name)

	// Try to get stored metadata to recreate with same worktree
	var worktreePath string
	if sessionInfo, ok := sl.sessionState.Sessions[session.Name]; ok {
		worktreePath = sessionInfo.WorktreePath
		logging.Logger.Info("Recreating session with stored worktree", "name", session.Name, "worktree", worktreePath)
	} else {
		logging.Logger.Warn("No stored metadata for session, creating without worktree", "name", session.Name)
	}

	// Recreate the session
	if _, err := sl.tmuxClient.Create(session.Name, worktreePath); err != nil {
		sl.err = fmt.Errorf("failed to recreate session: %w", err)
		return false
	}

	return true
}

// sessionsFromState rebuilds the session list from state.json (duplicated to avoid circular import)
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
