package ui

import (
	"fmt"
	"io"
	"rocha/logging"
	"rocha/state"
	"rocha/tmux"
	"rocha/version"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const escTimeout = 500 * time.Millisecond

// Internal messages for SessionList
type checkStateMsg struct{}          // Triggers state file check
type sessionListDetachedMsg struct{} // Session list returned from attached state

// SessionItem implements list.Item and list.DefaultItem
type SessionItem struct {
	Session     *tmux.Session
	DisplayName string
	GitRef      string
	State       string
}

// FilterValue implements list.Item
func (i SessionItem) FilterValue() string {
	return i.DisplayName + " " + i.GitRef
}

// Title implements list.DefaultItem
func (i SessionItem) Title() string {
	return i.DisplayName
}

// Description implements list.DefaultItem
func (i SessionItem) Description() string {
	return i.GitRef
}

// SessionDelegate is a custom delegate for rendering session items
type SessionDelegate struct {
	sessionState *state.SessionState
}

func newSessionDelegate(sessionState *state.SessionState) SessionDelegate {
	return SessionDelegate{sessionState: sessionState}
}

// Height implements list.ItemDelegate
func (d SessionDelegate) Height() int {
	return 2 // Two lines per item (name + git ref)
}

// Spacing implements list.ItemDelegate
func (d SessionDelegate) Spacing() int {
	return 0
}

// Update implements list.ItemDelegate
func (d SessionDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render implements list.ItemDelegate
func (d SessionDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(SessionItem)
	if !ok {
		return
	}

	// Get current index and check if selected
	isSelected := index == m.Index()
	cursor := " "
	if isSelected {
		cursor = ">"
	}

	// Get session state
	sessionState := item.State

	// Render status icon
	var statusIcon string
	switch sessionState {
	case state.StateWorking:
		statusIcon = workingIconStyle.Render(state.SymbolWorking)
	case state.StateIdle:
		statusIcon = idleIconStyle.Render(state.SymbolIdle)
	case state.StateWaitingUser:
		statusIcon = waitingIconStyle.Render(state.SymbolWaitingUser)
	case state.StateExited:
		statusIcon = exitedIconStyle.Render(state.SymbolExited)
	}

	// Build first line: cursor + zero-padded number + status + name
	line1 := fmt.Sprintf("%s %02d. %s %s", cursor, index+1, statusIcon, item.DisplayName)
	line1 = normalStyle.Render(line1)

	// Build second line: git ref (indented to align with session name)
	var line2 string
	if item.GitRef != "" {
		indent := "        " // 8 spaces to align with session name (> 01. ● name)
		line2 = branchStyle.Render(fmt.Sprintf("%s%s", indent, item.GitRef))
	}

	// Write both lines
	fmt.Fprint(w, line1+"\n"+line2)
}

// SessionList is a Bubble Tea component for displaying and managing sessions
type SessionList struct {
	list         list.Model
	tmuxClient   tmux.Client
	sessionState *state.SessionState
	err          error

	// Escape handling for filter clearing
	escPressCount int
	escPressTime  time.Time

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

	// Build items from state
	items := buildListItems(sessionState)

	// Create delegate
	delegate := newSessionDelegate(sessionState)

	// Create list with reasonable default size (will be resized on WindowSizeMsg)
	// Initial height: assume 40 line terminal - 12 lines for header/help = 28
	l := list.New(items, delegate, 80, 28)
	l.SetShowTitle(false) // We'll render our own title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // We'll render our own help

	return &SessionList{
		list:         l,
		tmuxClient:   tmuxClient,
		sessionState: sessionState,
		err:          err,
	}
}

// Init starts the session list component, including auto-refresh polling
func (sl *SessionList) Init() tea.Cmd {
	return pollStateCmd() // Start auto-refresh polling
}

// Update handles messages for the session list component
func (sl *SessionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

			// Update delegate with new state
			delegate := newSessionDelegate(newState)
			sl.list.SetDelegate(delegate)

			// Rebuild items
			items := buildListItems(newState)
			cmd := sl.list.SetItems(items)
			return sl, tea.Batch(cmd, pollStateCmd())
		}

		return sl, pollStateCmd()

	case error:
		sl.err = msg
		return sl, pollStateCmd()

	case tea.KeyMsg:
		// Handle custom keys before delegating to list
		switch msg.String() {
		case "ctrl+c", "q":
			sl.ShouldQuit = true
			return sl, nil

		case "n":
			sl.RequestNewSession = true
			return sl, nil

		case "enter":
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				// Ensure session exists
				if !sl.ensureSessionExists(item.Session) {
					return sl, pollStateCmd()
				}
				sl.SelectedSession = item.Session
				return sl, nil
			}

		case "x":
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToKill = item.Session
				return sl, nil
			}

		case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7":
			// Quick attach to session by number
			numStr := msg.String()[4:] // Skip "alt+"
			num := int(numStr[0] - '0')
			index := num - 1

			items := sl.list.VisibleItems()
			if index >= 0 && index < len(items) {
				if item, ok := items[index].(SessionItem); ok {
					// Update list's internal selection state
					sl.list.Select(index)

					if !sl.ensureSessionExists(item.Session) {
						return sl, pollStateCmd()
					}
					sl.SelectedSession = item.Session
					return sl, nil
				}
			}

		case "esc":
			// Handle double-ESC for filter clearing
			if sl.list.FilterState() != list.Unfiltered {
				now := time.Now()
				if now.Sub(sl.escPressTime) < escTimeout && sl.escPressCount >= 1 {
					// Second ESC - clear filter
					sl.list.ResetFilter()
					sl.escPressCount = 0
					return sl, pollStateCmd()
				}
				// First ESC
				sl.escPressCount = 1
				sl.escPressTime = now
			}
		}

	case tea.WindowSizeMsg:
		// Update list size - reserve space for:
		// - Header: 2 lines (title + tagline)
		// - Spacing after header: 2 lines
		// - Help text: 6 lines
		// - Spacing before help: 2 lines
		sl.list.SetSize(msg.Width, msg.Height-12)
	}

	// Delegate to list for normal handling
	var cmd tea.Cmd
	sl.list, cmd = sl.list.Update(msg)
	return sl, tea.Batch(cmd, pollStateCmd())
}

// View renders the session list component
func (sl *SessionList) View() string {
	var s string

	// Add custom header
	titleText := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Render("Rocha")
	s += titleText + "\n"
	s += normalStyle.Render(version.Tagline) + "\n\n"

	// Render list
	s += sl.list.View()

	// Show error if any
	if sl.err != nil {
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", sl.err))
		sl.err = nil
	}

	// Add custom help (status legend first, then keys)
	s += "\n\n"
	helpText := sl.renderStatusLegend() + "\n\n"
	helpText += "↑/k: up • ↓/j: down • /: filter • n: new\n"
	helpText += "enter/Alt+1-7: attach (Ctrl+B D or Ctrl+Q to detach) • x: kill • q: quit"

	s += helpStyle.Render(helpText)

	return s
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

	// Update delegate
	delegate := newSessionDelegate(sessionState)
	sl.list.SetDelegate(delegate)

	// Rebuild items
	items := buildListItems(sessionState)
	sl.list.SetItems(items)
}

// pollStateCmd returns a command that waits 2 seconds then sends checkStateMsg
func pollStateCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return checkStateMsg{}
	})
}

// buildListItems converts SessionState to list items
func buildListItems(sessionState *state.SessionState) []list.Item {
	var items []list.Item
	var sessions []*tmux.Session

	// Build sessions from state
	for name, info := range sessionState.Sessions {
		sessions = append(sessions, &tmux.Session{
			Name:      name,
			CreatedAt: info.LastUpdated,
		})
	}

	// Sort by name
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Name < sessions[j].Name
	})

	// Convert to list items
	for _, session := range sessions {
		info := sessionState.Sessions[session.Name]
		displayName := session.Name
		if info.DisplayName != "" {
			displayName = info.DisplayName
		}

		// Build git reference
		var gitRef string
		if info.RepoInfo != "" && info.BranchName != "" {
			gitRef = fmt.Sprintf("%s:%s", info.RepoInfo, info.BranchName)
		} else if info.BranchName != "" {
			gitRef = info.BranchName
		}

		items = append(items, SessionItem{
			Session:     session,
			DisplayName: displayName,
			GitRef:      gitRef,
			State:       info.State,
		})
	}

	return items
}

// renderStatusLegend renders the status legend with counts
func (sl *SessionList) renderStatusLegend() string {
	workingCount, idleCount, waitingCount, exitedCount := sl.countSessionsByState()

	legend := "Status: "
	legend += workingIconStyle.Render(state.SymbolWorking) + fmt.Sprintf(" %d working • ", workingCount)
	legend += idleIconStyle.Render(state.SymbolIdle) + fmt.Sprintf(" %d idle • ", idleCount)
	legend += waitingIconStyle.Render(state.SymbolWaitingUser) + fmt.Sprintf(" %d waiting • ", waitingCount)
	legend += exitedIconStyle.Render(state.SymbolExited) + fmt.Sprintf(" %d exited", exitedCount)

	return legend
}

// countSessionsByState counts sessions by their state
func (sl *SessionList) countSessionsByState() (working, idle, waiting, exited int) {
	for _, sessionInfo := range sl.sessionState.Sessions {
		switch sessionInfo.State {
		case state.StateWorking:
			working++
		case state.StateIdle:
			idle++
		case state.StateWaitingUser:
			waiting++
		case state.StateExited:
			exited++
		}
	}
	return
}

// ensureSessionExists checks if a session exists and recreates it if needed
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
