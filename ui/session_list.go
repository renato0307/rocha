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
	Session         *tmux.Session
	DisplayName     string
	GitRef          string
	State           string
	HasShellSession bool // Track if shell session exists
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

	// Add shell session indicator at the end
	if item.HasShellSession {
		line1 += " " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("22")).
			Render("⌨")
	}

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
	SelectedSession      *tmux.Session // Session user wants to attach to
	SelectedShellSession *tmux.Session // Session user wants shell session for
	SessionToKill        *tmux.Session // Session user wants to kill
	SessionToRename      *tmux.Session // Session user wants to rename
	RequestNewSession    bool          // User pressed 'n'
	ShouldQuit           bool          // User pressed 'q' or Ctrl+C
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
	items := buildListItems(sessionState, tmuxClient)

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
			items := buildListItems(newState, sl.tmuxClient)
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

		case "r":
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToRename = item.Session
				return sl, nil
			}

		case "shift+up", "shift+k":
			return sl, sl.moveSelectedUp()

		case "shift+down", "shift+j":
			return sl, sl.moveSelectedDown()

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

		case "alt+enter":
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				// Ensure session exists
				if !sl.ensureSessionExists(item.Session) {
					return sl, pollStateCmd()
				}
				sl.SelectedShellSession = item.Session
				return sl, nil
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
	helpText += "↑/k: up • ↓/j: down • Shift+↑/K: move up • Shift+↓/J: move down • /: filter • n: new • r: rename • x: kill • q: quit\n"
	helpText += "enter/alt+1-7: attach • ctrl+q: detach • alt+enter: shell (⌨)"

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
	items := buildListItems(sessionState, sl.tmuxClient)
	sl.list.SetItems(items)
}

// pollStateCmd returns a command that waits 2 seconds then sends checkStateMsg
func pollStateCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return checkStateMsg{}
	})
}

// buildListItems converts SessionState to list items
func buildListItems(sessionState *state.SessionState, tmuxClient tmux.Client) []list.Item {
	var items []list.Item

	// Build sessions from state
	// No need to filter - shell sessions won't have top-level entries with nested structure!
	sessionsMap := make(map[string]*tmux.Session)
	for name, info := range sessionState.Sessions {
		sessionsMap[name] = &tmux.Session{
			Name:      name,
			CreatedAt: info.LastUpdated,
		}
	}

	// Initialize OrderedSessionNames if empty (first time)
	if len(sessionState.OrderedSessionNames) == 0 && len(sessionsMap) > 0 {
		// Get all session names and sort alphabetically for initial order
		var names []string
		for name := range sessionsMap {
			names = append(names, name)
		}
		sort.Strings(names)
		sessionState.OrderedSessionNames = names
		sessionState.Save() // Persist the initial order
	}

	// Apply manual order
	var sessions []*tmux.Session
	sessions = applyManualOrder(sessionsMap, sessionState.OrderedSessionNames)

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

		// Check if shell session exists (check nested object)
		hasShell := false
		if info.ShellSession != nil {
			hasShell = tmuxClient.Exists(info.ShellSession.Name)
		}

		items = append(items, SessionItem{
			Session:         session,
			DisplayName:     displayName,
			GitRef:          gitRef,
			State:           info.State,
			HasShellSession: hasShell,
		})
	}

	return items
}

// applyManualOrder orders sessions according to OrderedSessionNames
// Sessions in the order list come first, followed by unordered sessions alphabetically
func applyManualOrder(sessionsMap map[string]*tmux.Session, orderedNames []string) []*tmux.Session {
	var ordered []*tmux.Session
	orderedSet := make(map[string]bool)

	// Add sessions in manual order
	for _, name := range orderedNames {
		if session, exists := sessionsMap[name]; exists {
			ordered = append(ordered, session)
			orderedSet[name] = true
		}
	}

	// Find sessions not in order list and add them alphabetically at end
	var unordered []string
	for name := range sessionsMap {
		if !orderedSet[name] {
			unordered = append(unordered, name)
		}
	}
	sort.Strings(unordered)

	// Append unordered sessions
	for _, name := range unordered {
		ordered = append(ordered, sessionsMap[name])
	}

	return ordered
}

// renderStatusLegend renders the status legend with counts
func (sl *SessionList) renderStatusLegend() string {
	workingCount, idleCount, waitingCount, exitedCount := sl.countSessionsByState()

	legend := workingIconStyle.Render(state.SymbolWorking) + fmt.Sprintf(" %d working • ", workingCount)
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

// moveSelectedUp moves the currently selected session up one position in the order
func (sl *SessionList) moveSelectedUp() tea.Cmd {
	item, ok := sl.list.SelectedItem().(SessionItem)
	if !ok {
		return nil
	}

	currentIndex := sl.list.Index()
	if currentIndex <= 0 {
		return nil // Already at top
	}

	// Move session up in state
	if err := sl.sessionState.MoveSessionUp(item.Session.Name); err != nil {
		logging.Logger.Warn("Failed to move session up", "session", item.Session.Name, "error", err)
		return nil
	}

	// Reload state and rebuild list
	sl.RefreshFromState()

	// Adjust cursor to follow moved item
	sl.list.Select(currentIndex - 1)

	return pollStateCmd()
}

// moveSelectedDown moves the currently selected session down one position in the order
func (sl *SessionList) moveSelectedDown() tea.Cmd {
	item, ok := sl.list.SelectedItem().(SessionItem)
	if !ok {
		return nil
	}

	currentIndex := sl.list.Index()
	items := sl.list.Items()
	if currentIndex >= len(items)-1 {
		return nil // Already at bottom
	}

	// Move session down in state
	if err := sl.sessionState.MoveSessionDown(item.Session.Name); err != nil {
		logging.Logger.Warn("Failed to move session down", "session", item.Session.Name, "error", err)
		return nil
	}

	// Reload state and rebuild list
	sl.RefreshFromState()

	// Adjust cursor to follow moved item
	sl.list.Select(currentIndex + 1)

	return pollStateCmd()
}
