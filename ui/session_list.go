package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/storage"
	"rocha/tmux"
	"rocha/version"
	"sort"
	"strings"
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
	sessionState *storage.SessionState
}

func newSessionDelegate(sessionState *storage.SessionState) SessionDelegate {
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

		// Apply colors to +N and -N in the git ref
		styledGitRef := item.GitRef

		// Split by " | " to process each section
		parts := strings.Split(styledGitRef, " | ")
		for i, part := range parts {
			// Check if this part contains file stats (starts with +digit or has +digit in it)
			hasFileStats := false
			words := strings.Fields(part)
			for _, word := range words {
				if (strings.HasPrefix(word, "+") || strings.HasPrefix(word, "-")) && len(word) > 1 {
					// Check if the character after + or - is a digit
					if len(word) > 1 && word[1] >= '0' && word[1] <= '9' {
						hasFileStats = true
						break
					}
				}
			}

			if hasFileStats {
				// Color file stats: +N in green, -N in red
				for j, word := range words {
					if strings.HasPrefix(word, "+") && len(word) > 1 && word[1] >= '0' && word[1] <= '9' {
						words[j] = additionsStyle.Render(word)
					} else if strings.HasPrefix(word, "-") && len(word) > 1 && word[1] >= '0' && word[1] <= '9' {
						words[j] = deletionsStyle.Render(word)
					} else {
						// Other parts of stats section stay gray
						words[j] = branchStyle.Render(word)
					}
				}
				parts[i] = strings.Join(words, " ")
			} else {
				// Apply gray color to non-stat parts (branch name, PR, ahead/behind, etc)
				parts[i] = branchStyle.Render(part)
			}
		}
		styledGitRef = strings.Join(parts, branchStyle.Render(" | "))

		line2 = branchStyle.Render(indent) + styledGitRef
	}

	// Write both lines
	fmt.Fprint(w, line1+"\n"+line2)
}

// SessionList is a Bubble Tea component for displaying and managing sessions
type SessionList struct {
	list         list.Model
	tmuxClient   tmux.Client
	store        *storage.Store         // Storage for persistent state
	sessionState *storage.SessionState
	err          error
	editor       string // Editor to open sessions in

	// Window dimensions
	width  int
	height int

	// Escape handling for filter clearing
	escPressCount int
	escPressTime  time.Time

	// Git stats fetching
	fetchingGitStats bool // Prevent concurrent fetches

	// Result fields - set by component, read by Model
	SelectedSession      *tmux.Session // Session user wants to attach to
	SelectedShellSession *tmux.Session // Session user wants shell session for
	SessionToKill        *tmux.Session // Session user wants to kill
	SessionToRename      *tmux.Session // Session user wants to rename
	SessionToOpenEditor  *tmux.Session // Session user wants to open in editor
	RequestNewSession    bool          // User pressed 'n'
	RequestTestError     bool          // User pressed 'alt+e' (test command)
	ShouldQuit           bool          // User pressed 'q' or Ctrl+C
}

// NewSessionList creates a new session list component
func NewSessionList(tmuxClient tmux.Client, store *storage.Store, editor string) *SessionList {
	// Load session state
	sessionState, err := store.Load(context.Background())
	if err != nil {
		logging.Logger.Warn("Failed to load session state", "error", err)
		sessionState = &storage.SessionState{Sessions: make(map[string]storage.SessionInfo)}
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
		store:        store,
		sessionState: sessionState,
		err:          err,
		editor:       editor,
	}
}

// Init starts the session list component, including auto-refresh polling
func (sl *SessionList) Init() tea.Cmd {
	return pollStateCmd() // Start auto-refresh polling
}

// Update handles messages for the session list component
func (sl *SessionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case git.GitStatsReadyMsg:
		// Git stats successfully fetched
		if info, exists := sl.sessionState.Sessions[msg.SessionName]; exists {
			info.GitStats = msg.Stats
			sl.sessionState.Sessions[msg.SessionName] = info
		}

		// Rebuild items with updated stats
		delegate := newSessionDelegate(sl.sessionState)
		sl.list.SetDelegate(delegate)
		items := buildListItems(sl.sessionState, sl.tmuxClient)
		cmd := sl.list.SetItems(items)

		// Mark fetching as done and continue polling
		sl.fetchingGitStats = false
		return sl, tea.Batch(cmd, pollStateCmd())

	case git.GitStatsErrorMsg:
		// Git stats fetch failed - log and continue
		logging.Logger.Debug("Git stats fetch failed", "session", msg.SessionName, "error", msg.Err)
		sl.fetchingGitStats = false
		return sl, pollStateCmd()

	case checkStateMsg:
		// Auto-refresh: Check if state has changed
		newState, err := sl.store.Load(context.Background())
		if err != nil {
			// Continue polling even on error
			return sl, pollStateCmd()
		}

		// Preserve GitStats cache from old state
		for name, newInfo := range newState.Sessions {
			if oldInfo, exists := sl.sessionState.Sessions[name]; exists {
				newInfo.GitStats = oldInfo.GitStats
				newState.Sessions[name] = newInfo
			}
		}

		sl.sessionState = newState

		// Update delegate with new state
		delegate := newSessionDelegate(newState)
		sl.list.SetDelegate(delegate)

		// Rebuild items
		items := buildListItems(newState, sl.tmuxClient)
		cmd := sl.list.SetItems(items)

		// Request git stats for visible sessions
		gitStatsCmd := sl.requestGitStatsForVisible()
		return sl, tea.Batch(cmd, pollStateCmd(), gitStatsCmd)

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

		case "o":
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToOpenEditor = item.Session
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

		case "alt+e":
			// Hidden test command: Request Model to generate test error
			sl.RequestTestError = true
			return sl, nil

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
		// Store dimensions
		sl.width = msg.Width
		sl.height = msg.Height

		// Calculate list size - reserve space for:
		// - Header: 2 lines (title + tagline)
		// - Spacing after header: 2 lines
		// - Help text: 6 lines (legend + keys)
		// - Spacing before help: 2 lines
		// - Error space: 2 lines (always reserved)
		// Total reserved: 14 lines
		sl.list.SetSize(msg.Width, msg.Height-14)
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

	// Show SessionList error if any (transient, limited to 2 lines)
	if sl.err != nil {
		errorText := formatErrorForDisplay(sl.err, sl.width)
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(errorText)
		sl.err = nil
	}

	// Add custom help (status legend first, then keys)
	s += "\n\n"
	helpText := sl.renderStatusLegend() + "\n\n"
	helpText += "↑/k: up • ↓/j: down • shift+↑/k: move up • shift+↓/j: move down • /: filter • n: new • r: rename • x: kill\n"
	helpText += "enter/alt+1-7: attach • ctrl+q: detach • alt+enter: shell (⌨) • o: open editor • q: quit"

	s += helpStyle.Render(helpText)

	return s
}

// RefreshFromState reloads the session list from state
func (sl *SessionList) RefreshFromState() {
	sessionState, err := sl.store.Load(context.Background())
	if err != nil {
		sl.err = fmt.Errorf("failed to refresh sessions: %w", err)
		logging.Logger.Error("Failed to refresh session state", "error", err)
		return
	}

	// Preserve GitStats cache from old state
	for name, newInfo := range sessionState.Sessions {
		if oldInfo, exists := sl.sessionState.Sessions[name]; exists {
			newInfo.GitStats = oldInfo.GitStats
			sessionState.Sessions[name] = newInfo
		}
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
func buildListItems(sessionState *storage.SessionState, tmuxClient tmux.Client) []list.Item {
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

	// Build ordered sessions list using OrderedSessionNames from database
	var sessions []*tmux.Session
	orderedSet := make(map[string]bool)

	// Add sessions in database order
	for _, name := range sessionState.OrderedSessionNames {
		if session, exists := sessionsMap[name]; exists {
			sessions = append(sessions, session)
			orderedSet[name] = true
		}
	}

	// Add any new sessions not yet in database order (alphabetically at end)
	var unordered []string
	for name := range sessionsMap {
		if !orderedSet[name] {
			unordered = append(unordered, name)
		}
	}
	sort.Strings(unordered)
	for _, name := range unordered {
		sessions = append(sessions, sessionsMap[name])
	}

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

		// Append git stats if available
		if info.GitStats != nil {
			if stats, ok := info.GitStats.(*git.GitStats); ok {
				if stats.Error != nil {
					// Log the error to help debug why some repos don't show info
					logging.Logger.Debug("Git stats error for session",
						"session", session.Name,
						"error", stats.Error)
				} else {
					// Add PR number (if available)
					if stats.PRNumber != "" {
						gitRef += fmt.Sprintf(" | #%s", stats.PRNumber)
					}

					// Add ahead/behind (if non-zero)
					if stats.Ahead > 0 || stats.Behind > 0 {
						gitRef += fmt.Sprintf(" | ↑%d ↓%d", stats.Ahead, stats.Behind)
					}

					// Add file stats (if non-zero)
					if stats.Additions > 0 || stats.Deletions > 0 {
						gitRef += fmt.Sprintf(" | +%d -%d", stats.Additions, stats.Deletions)
					}
				}
			}
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

	// Get previous item
	items := sl.list.Items()
	prevItem, ok := items[currentIndex-1].(SessionItem)
	if !ok {
		return nil
	}

	// Swap positions in database
	if err := sl.store.SwapPositions(context.Background(), item.Session.Name, prevItem.Session.Name); err != nil {
		logging.Logger.Warn("Failed to swap session positions", "error", err)
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

	// Get next item
	nextItem, ok := items[currentIndex+1].(SessionItem)
	if !ok {
		return nil
	}

	// Swap positions in database
	if err := sl.store.SwapPositions(context.Background(), item.Session.Name, nextItem.Session.Name); err != nil {
		logging.Logger.Warn("Failed to swap session positions", "error", err)
	}

	// Reload state and rebuild list
	sl.RefreshFromState()

	// Adjust cursor to follow moved item
	sl.list.Select(currentIndex + 1)

	return pollStateCmd()
}

// requestGitStatsForVisible fetches git stats for visible sessions
// Returns a tea.Cmd that will fetch stats asynchronously
func (sl *SessionList) requestGitStatsForVisible() tea.Cmd {
	// Don't start a new fetch if one is already in progress
	if sl.fetchingGitStats {
		return nil
	}

	// Get visible items
	visibleItems := sl.list.VisibleItems()
	if len(visibleItems) == 0 {
		return nil
	}

	// Collect requests for sessions that need stats
	var requests []git.GitStatsRequest
	selectedIndex := sl.list.Index()

	for i, item := range visibleItems {
		sessionItem, ok := item.(SessionItem)
		if !ok {
			continue
		}

		// Get session info
		info, exists := sl.sessionState.Sessions[sessionItem.Session.Name]
		if !exists {
			logging.Logger.Debug("Session not in state, skipping git stats",
				"session", sessionItem.Session.Name)
			continue
		}

		// Determine which path to use for git stats
		// Prefer worktree path, fall back to repo path
		gitPath := info.WorktreePath
		if gitPath == "" {
			gitPath = info.RepoPath
		}

		if gitPath == "" {
			logging.Logger.Debug("No git path for session, skipping git stats",
				"session", sessionItem.Session.Name)
			continue
		}

		// Check if git path exists
		if _, err := os.Stat(gitPath); os.IsNotExist(err) {
			logging.Logger.Debug("Git path does not exist, skipping git stats",
				"session", sessionItem.Session.Name,
				"path", gitPath)
			continue
		}

		// Check if stats are fresh (< 5 seconds old)
		if info.GitStats != nil {
			if stats, ok := info.GitStats.(*git.GitStats); ok {
				if time.Since(stats.FetchedAt) < 5*time.Second {
					continue // Stats are fresh, skip
				}
			}
		}

		// Add request with priority (selected = 1, others = 0)
		priority := 0
		if i == selectedIndex {
			priority = 1
		}

		requests = append(requests, git.GitStatsRequest{
			SessionName:  sessionItem.Session.Name,
			WorktreePath: gitPath, // Use gitPath which can be either worktree or repo path
			Priority:     priority,
		})

		// Limit to 5 requests per cycle to fetch faster while preventing git storms
		if len(requests) >= 5 {
			break
		}
	}

	if len(requests) == 0 {
		return nil
	}

	// Sort by priority (higher first)
	if len(requests) > 1 {
		for i := 0; i < len(requests)-1; i++ {
			for j := i + 1; j < len(requests); j++ {
				if requests[j].Priority > requests[i].Priority {
					requests[i], requests[j] = requests[j], requests[i]
				}
			}
		}
	}

	// Mark as fetching
	sl.fetchingGitStats = true

	// Start fetchers for all requests in parallel
	var cmds []tea.Cmd
	for _, req := range requests {
		cmds = append(cmds, git.StartGitStatsFetcher(req))
	}

	return tea.Batch(cmds...)
}
