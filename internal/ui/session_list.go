package ui

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/services"
	"github.com/renato0307/rocha/internal/theme"
)

const escTimeout = 500 * time.Millisecond

// Messages for SessionList (exported for Model integration)
type checkStateMsg struct{} // Triggers state file check - used by Model for token chart refresh
type hideTipMsg struct{}             // Time to hide the current tip
type sessionListDetachedMsg struct{} // Session list returned from attached state
type showTipMsg struct{}             // Time to show a new random tip

// SessionItem implements list.Item and list.DefaultItem
type SessionItem struct {
	Comment         string
	DisplayName     string
	GitRef          string
	HasShellSession bool // Track if shell session exists
	IsFlagged       bool
	LastUpdated     time.Time
	Session         *ports.TmuxSession
	State           string
	Status          *string // Implementation status
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
	sessionState    *domain.SessionCollection
	statusConfig    *config.StatusConfig
	timestampConfig *config.TimestampColorConfig
	timestampMode   TimestampMode
}

func newSessionDelegate(sessionState *domain.SessionCollection, statusConfig *config.StatusConfig, timestampConfig *config.TimestampColorConfig, timestampMode TimestampMode) SessionDelegate {
	return SessionDelegate{
		sessionState:    sessionState,
		statusConfig:    statusConfig,
		timestampConfig: timestampConfig,
		timestampMode:   timestampMode,
	}
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
	sessionState := domain.SessionState(item.State)

	// Render status icon
	var statusIcon string
	switch sessionState {
	case domain.StateWorking:
		statusIcon = theme.WorkingIconStyle.Render(domain.SymbolWorking)
	case domain.StateIdle:
		statusIcon = theme.IdleIconStyle.Render(domain.SymbolIdle)
	case domain.StateWaiting:
		statusIcon = theme.WaitingIconStyle.Render(domain.SymbolWaiting)
	case domain.StateExited:
		statusIcon = theme.ExitedIconStyle.Render(domain.SymbolExited)
	}

	// Build first line: cursor + zero-padded number + status + name
	line1 := fmt.Sprintf("%s %02d. %s %s", cursor, index+1, statusIcon, item.DisplayName)
	line1 = theme.NormalStyle.Render(line1)

	// Add flag indicator if flagged
	if item.IsFlagged {
		line1 += " ⚑"
	}

	// Add comment indicator if there's a comment
	if item.Comment != "" {
		line1 += " ⌨"
	}

	// Add shell session indicator at the end
	if item.HasShellSession {
		line1 += " >_"
	}

	// Add implementation status if set (with color-coded brackets)
	if item.Status != nil && *item.Status != "" {
		statusColor := d.statusConfig.GetColor(*item.Status)
		line1 += " " + theme.StatusStyle(statusColor).Render("["+*item.Status+"]")
	}

	// Add timestamp at the end with color based on age
	if !item.LastUpdated.IsZero() {
		var timeStr string
		switch d.timestampMode {
		case TimestampRelative:
			timeStr = formatRelativeTime(item.LastUpdated)
		case TimestampAbsolute:
			timeStr = formatAbsoluteTime(item.LastUpdated)
		case TimestampHidden:
			// Don't show timestamp
		}

		if timeStr != "" {
			color := getTimestampColor(item.LastUpdated, d.timestampConfig)
			line1 += " " + theme.TimestampStyle(color).Render("["+timeStr+"]")
		}
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
						words[j] = theme.AdditionsStyle.Render(word)
					} else if strings.HasPrefix(word, "-") && len(word) > 1 && word[1] >= '0' && word[1] <= '9' {
						words[j] = theme.DeletionsStyle.Render(word)
					} else {
						// Other parts of stats section stay gray
						words[j] = theme.BranchStyle.Render(word)
					}
				}
				parts[i] = strings.Join(words, " ")
			} else {
				// Apply gray color to non-stat parts (branch name, PR, ahead/behind, etc)
				parts[i] = theme.BranchStyle.Render(part)
			}
		}
		styledGitRef = strings.Join(parts, theme.BranchStyle.Render(" | "))

		line2 = theme.BranchStyle.Render(indent) + styledGitRef
	}

	// Write both lines
	fmt.Fprint(w, line1+"\n"+line2)
}

// SessionList is a Bubble Tea component for displaying and managing sessions
type SessionList struct {
	devMode            bool
	editor             string // Editor to open sessions in
	err                error
	fetchingGitStats   bool // Prevent concurrent fetches
	gitService         *services.GitService // Git operations service
	keys               KeyMap
	list               list.Model
	sessionService     *services.SessionService // Session service
	sessionState       *domain.SessionCollection
	statusConfig       *config.StatusConfig
	timestampMode      TimestampMode
	tmuxStatusPosition string

	// Tips feature
	currentTip *Tip       // Currently displayed tip (nil = hidden)
	tipsConfig TipsConfig // Tips display configuration

	// Escape handling for filter clearing
	escPressCount int
	escPressTime  time.Time

	// Window dimensions
	height     int
	listHeight int // Height available for the list component
	width      int

	// Timestamp configuration
	timestampConfig *config.TimestampColorConfig

	// Result fields - set by component, read by Model
	RequestHelp           bool               // User pressed 'h' or '?'
	RequestNewSession     bool               // User pressed 'n'
	RequestNewSessionFrom bool               // User pressed 'shift+n' (new from same repo)
	RequestTestError      bool               // User pressed 'alt+e' (test command)
	SelectedSession       *ports.TmuxSession // Session user wants to attach to
	SessionForTemplate    *ports.TmuxSession // Session to use as template for new session
	SelectedShellSession  *ports.TmuxSession // Session user wants shell session for
	SessionToArchive      *ports.TmuxSession // Session user wants to archive
	SessionToComment      *ports.TmuxSession // Session user wants to comment
	SessionToKill         *ports.TmuxSession // Session user wants to kill
	SessionToOpenEditor   *ports.TmuxSession // Session user wants to open in editor
	SessionToRename       *ports.TmuxSession // Session user wants to rename
	SessionToSendText     *ports.TmuxSession // Session user wants to send text to
	SessionToSetStatus    *ports.TmuxSession // Session user wants to set status for
	SessionToToggleFlag   *ports.TmuxSession // Session user wants to toggle flag
	ShouldQuit            bool               // User pressed 'q' or Ctrl+C
}

// NewSessionList creates a new session list component
func NewSessionList(sessionService *services.SessionService, gitService *services.GitService, editor string, statusConfig *config.StatusConfig, timestampConfig *config.TimestampColorConfig, devMode bool, timestampMode TimestampMode, keys KeyMap, tmuxStatusPosition string, tipsConfig TipsConfig) *SessionList {
	// Load session state (showArchived=false - TUI never shows archived sessions)
	sessionState, err := sessionService.LoadState(context.Background(), false)
	if err != nil {
		logging.Logger.Warn("Failed to load session state", "error", err)
		sessionState = &domain.SessionCollection{Sessions: make(map[string]domain.Session)}
	}

	// Build items from state
	items := buildListItems(sessionState, sessionService, statusConfig)

	// Create delegate
	delegate := newSessionDelegate(sessionState, statusConfig, timestampConfig, timestampMode)

	// Create list with reasonable default size (will be resized on WindowSizeMsg)
	// Initial height: assume 40 line terminal - 12 lines for header/help = 28
	l := list.New(items, delegate, 80, 28)
	l.SetShowTitle(false)      // We'll render our own title
	l.SetShowStatusBar(false)  // No status bar
	l.SetShowPagination(false) // No pagination dots
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // We'll render our own help

	// Show a tip immediately at startup if tips are enabled
	var initialTip *Tip
	allTips := GetTips()
	if tipsConfig.Enabled && len(allTips) > 0 {
		initialTip = &allTips[rand.Intn(len(allTips))]
	}

	return &SessionList{
		currentTip:         initialTip,
		devMode:            devMode,
		editor:             editor,
		err:                err,
		gitService:         gitService,
		keys:               keys,
		list:               l,
		sessionService:     sessionService,
		sessionState:       sessionState,
		statusConfig:       statusConfig,
		timestampConfig:    timestampConfig,
		timestampMode:      timestampMode,
		tipsConfig:         tipsConfig,
		tmuxStatusPosition: tmuxStatusPosition,
	}
}

// Init starts the session list component, including auto-refresh polling
func (sl *SessionList) Init() tea.Cmd {
	cmds := []tea.Cmd{pollStateCmd()}

	// Schedule hide for the initial tip (already shown at startup)
	if sl.tipsConfig.Enabled && sl.currentTip != nil {
		cmds = append(cmds, tea.Tick(time.Duration(sl.tipsConfig.DisplayDurationSeconds)*time.Second, func(time.Time) tea.Msg {
			return hideTipMsg{}
		}))
	}

	return tea.Batch(cmds...)
}

// Update handles messages for the session list component
func (sl *SessionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case GitStatsReadyMsg:
		// Git stats successfully fetched - convert to domain type
		if info, exists := sl.sessionState.Sessions[msg.SessionName]; exists {
			if msg.Stats != nil {
				info.GitStats = &domain.GitStats{
					Additions:    msg.Stats.Additions,
					Ahead:        msg.Stats.Ahead,
					Behind:       msg.Stats.Behind,
					ChangedFiles: msg.Stats.ChangedFiles,
					Deletions:    msg.Stats.Deletions,
					Error:        msg.Stats.Error,
					FetchedAt:    msg.Stats.FetchedAt,
				}
			}
			sl.sessionState.Sessions[msg.SessionName] = info
		}

		// Mark fetching as done
		sl.fetchingGitStats = false

		// Skip list rebuild when user is actively filtering to prevent flickering
		// Don't schedule new poll - let existing poll timer continue
		if sl.list.FilterState() == list.Filtering {
			return sl, nil
		}

		// Rebuild items with updated stats
		delegate := newSessionDelegate(sl.sessionState, sl.statusConfig, sl.timestampConfig, sl.timestampMode)
		sl.list.SetDelegate(delegate)
		items := buildListItems(sl.sessionState, sl.sessionService, sl.statusConfig)
		cmd := sl.list.SetItems(items)

		// Don't schedule new poll - one is already running
		return sl, cmd

	case GitStatsErrorMsg:
		// Git stats fetch failed - log and continue
		logging.Logger.Debug("Git stats fetch failed", "session", msg.SessionName, "error", msg.Err)
		sl.fetchingGitStats = false

		// Don't schedule new poll - one is already running
		return sl, nil

	case checkStateMsg:
		// This message is sent by the poll timer every 2 seconds
		// We schedule exactly ONE new poll at the end to maintain the loop

		// Skip refresh when user is actively filtering to prevent flickering
		if sl.list.FilterState() == list.Filtering {
			// Still schedule next poll to maintain the loop
			return sl, pollStateCmd()
		}

		// Auto-refresh: Check if state has changed (showArchived=false - TUI never shows archived)
		newState, err := sl.sessionService.LoadState(context.Background(), false)
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
		delegate := newSessionDelegate(newState, sl.statusConfig, sl.timestampConfig, sl.timestampMode)
		sl.list.SetDelegate(delegate)

		// Rebuild items
		items := buildListItems(newState, sl.sessionService, sl.statusConfig)
		cmd := sl.list.SetItems(items)

		// Request git stats for visible sessions
		gitStatsCmd := sl.requestGitStatsForVisible()

		// Schedule next poll to maintain the 2-second loop (exactly one poll)
		return sl, tea.Batch(cmd, pollStateCmd(), gitStatsCmd)

	case showTipMsg:
		// Don't show tip if there's an error - reschedule for later
		if sl.err != nil {
			return sl, tea.Tick(time.Duration(sl.tipsConfig.ShowIntervalSeconds)*time.Second, func(time.Time) tea.Msg {
				return showTipMsg{}
			})
		}
		// Time to show a new random tip
		allTips := GetTips()
		if len(allTips) > 0 {
			sl.currentTip = &allTips[rand.Intn(len(allTips))]
			return sl, tea.Tick(time.Duration(sl.tipsConfig.DisplayDurationSeconds)*time.Second, func(time.Time) tea.Msg {
				return hideTipMsg{}
			})
		}
		return sl, nil

	case hideTipMsg:
		// Hide the current tip and schedule the next one
		sl.currentTip = nil
		if sl.tipsConfig.Enabled {
			return sl, tea.Tick(time.Duration(sl.tipsConfig.ShowIntervalSeconds)*time.Second, func(time.Time) tea.Msg {
				return showTipMsg{}
			})
		}
		return sl, nil

	case error:
		sl.err = msg
		// Don't schedule new poll - one is already running
		return sl, nil

	// Each time you add something here, don't forget to add it to the help screen
	case tea.KeyMsg:
		// Guard clause: When actively filtering, bypass shortcuts to allow typing
		if sl.list.FilterState() == list.Filtering {
			// ESC is the only key we handle specially during filtering
			if msg.String() == "esc" {
				now := time.Now()
				if now.Sub(sl.escPressTime) < escTimeout && sl.escPressCount >= 1 {
					// Second ESC - clear filter
					sl.list.ResetFilter()
					sl.escPressCount = 0
					// Don't schedule new poll - one is already running
					return sl, nil
				}
				// First ESC - start counting
				sl.escPressCount = 1
				sl.escPressTime = now
			}

			// For all other keys during filtering, delegate to list immediately
			// Don't schedule new polls - one is already running
			var cmd tea.Cmd
			sl.list, cmd = sl.list.Update(msg)
			return sl, cmd
		}

		// Normal shortcut processing when NOT filtering
		switch {
		case key.Matches(msg, sl.keys.Application.Quit.Binding, sl.keys.Application.ForceQuit.Binding):
			sl.ShouldQuit = true
			return sl, nil

		case key.Matches(msg, sl.keys.Application.Help.Binding):
			sl.RequestHelp = true
			return sl, nil

		case key.Matches(msg, sl.keys.SessionManagement.New.Binding):
			sl.RequestNewSession = true
			return sl, nil

		case key.Matches(msg, sl.keys.SessionManagement.NewFromRepo.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.RequestNewSessionFrom = true
				sl.SessionForTemplate = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionActions.Open.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				// Ensure session exists
				if !sl.ensureSessionExists(item.Session) {
					// Don't schedule new poll - one is already running
					return sl, nil
				}
				sl.SelectedSession = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionManagement.Kill.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToKill = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionManagement.Rename.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToRename = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionMetadata.Comment.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToComment = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionMetadata.SendText.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToSendText = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionActions.OpenEditor.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToOpenEditor = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionMetadata.Flag.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToToggleFlag = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionManagement.Archive.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToArchive = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.SessionMetadata.StatusCycle.Binding):
			// s: Cycle through statuses (default/quick action)
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				return sl, sl.cycleSessionStatus(item.Session.Name)
			}

		case key.Matches(msg, sl.keys.SessionMetadata.StatusSetForm.Binding):
			// Shift+S: Open status form (edit action)
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				sl.SessionToSetStatus = item.Session
				return sl, nil
			}

		case key.Matches(msg, sl.keys.Navigation.MoveUp.Binding):
			return sl, sl.moveSelectedUp()

		case key.Matches(msg, sl.keys.Navigation.MoveDown.Binding):
			return sl, sl.moveSelectedDown()

		case key.Matches(msg, sl.keys.SessionActions.QuickOpen.Binding):
			// Quick attach to session by number
			numStr := msg.String()
			num := int(numStr[0] - '0')
			index := num - 1

			items := sl.list.VisibleItems()
			if index >= 0 && index < len(items) {
				if item, ok := items[index].(SessionItem); ok {
					// Update list's internal selection state
					sl.list.Select(index)

					if !sl.ensureSessionExists(item.Session) {
						// Don't schedule new poll - one is already running
						return sl, nil
					}
					sl.SelectedSession = item.Session
					return sl, nil
				}
			}

		case key.Matches(msg, sl.keys.SessionActions.OpenShell.Binding):
			if item, ok := sl.list.SelectedItem().(SessionItem); ok {
				// Ensure session exists
				if !sl.ensureSessionExists(item.Session) {
					// Don't schedule new poll - one is already running
					return sl, nil
				}
				sl.SelectedShellSession = item.Session
				return sl, nil
			}

		case msg.String() == "alt+e":
			// Hidden test command: Request Model to generate test error
			sl.RequestTestError = true
			return sl, nil

		case key.Matches(msg, sl.keys.Navigation.ClearFilter.Binding):
			// Handle double-ESC for filter clearing (only when filtering)
			if sl.list.FilterState() != list.Unfiltered {
				now := time.Now()
				if now.Sub(sl.escPressTime) < escTimeout && sl.escPressCount >= 1 {
					// Second ESC - clear filter
					sl.list.ResetFilter()
					sl.escPressCount = 0
					// Don't schedule new poll - one is already running
					return sl, nil
				}
				// First ESC
				sl.escPressCount = 1
				sl.escPressTime = now
			}
			// When not filtering, ESC does nothing (only q and ctrl+c exit)
			return sl, nil
		}

	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		switch msg.Type {
		case tea.MouseWheelUp:
			sl.list.CursorUp()
			return sl, nil
		case tea.MouseWheelDown:
			sl.list.CursorDown()
			return sl, nil
		}

	case tea.WindowSizeMsg:
		// Store dimensions - actual sizing is done by Model via SetSize()
		sl.width = msg.Width
		sl.height = msg.Height
	}

	// Delegate to list for normal handling
	var cmd tea.Cmd
	sl.list, cmd = sl.list.Update(msg)

	// IMPORTANT: Don't schedule new polls here!
	// The poll loop is maintained by checkStateMsg scheduling exactly one new poll.
	// Scheduling polls here would cause exponential accumulation.
	return sl, cmd
}

// View renders the session list component
func (sl *SessionList) View() string {
	var s string

	// Title + Tagline
	s += renderHeader(sl.devMode, "", "")

	// Legend + Shortcuts (moved to top, below header)
	helpText := sl.renderStatusLegend() + "  " + theme.HelpShortcutStyle.Render("?") + theme.HelpLabelStyle.Render(" shortcuts")

	// Add first-session hint when there's exactly 1 session (highlighted for first-timers)
	if len(sl.list.Items()) == 1 {
		helpText += "  " + theme.HintKeyStyle.Render(sl.keys.SessionActions.Open.Binding.Help().Key) + theme.HintLabelStyle.Render(" open Claude ") +
			theme.HintKeyStyle.Render(sl.keys.SessionActions.Detach.Binding.Help().Key) + theme.HintLabelStyle.Render(" return here")
	}

	s += theme.HelpStyle.Render(helpText) + "\n"

	// Session List
	if len(sl.list.Items()) == 0 {
		s += theme.HelpLabelStyle.Render("No sessions. Press ") + theme.HelpShortcutStyle.Render("n") + theme.HelpLabelStyle.Render(" to create a session.") + "\n"
	} else {
		s += sl.list.View()
	}

	// Show SessionList error if any (transient, limited to 2 lines)
	if sl.err != nil {
		errorText := formatErrorForDisplay(sl.err, sl.width)
		s += "\n" + theme.ErrorStyle.Render(errorText)
		sl.currentTip = nil // Clear tip when error is shown
		sl.err = nil
	}

	// Ensure output is exactly the expected height (4 lines header/legend/spacing + listHeight)
	// This prevents layout shifts regardless of list content
	expectedHeight := 4 + sl.listHeight
	actualHeight := lipgloss.Height(s)
	if actualHeight < expectedHeight {
		s += strings.Repeat("\n", expectedHeight-actualHeight)
	}

	return s
}

// GetCurrentTip returns the current tip text with highlighted keys (empty if no tip to show)
func (sl *SessionList) GetCurrentTip() string {
	if sl.currentTip == nil {
		return ""
	}
	return RenderTip(*sl.currentTip)
}

// ClearCurrentTip clears the current tip (called when error is shown)
func (sl *SessionList) ClearCurrentTip() {
	sl.currentTip = nil
}

// SetSize sets the available size for the session list
// width/height are the full terminal dimensions
// listHeight is the calculated height available for the list component
func (sl *SessionList) SetSize(width, height, listHeight int) {
	sl.width = width
	sl.height = height
	sl.listHeight = listHeight
	sl.list.SetSize(width, listHeight)
}

// RefreshFromState reloads the session list from state
func (sl *SessionList) RefreshFromState() {
	sessionState, err := sl.sessionService.LoadState(context.Background(), false)
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
	delegate := newSessionDelegate(sessionState, sl.statusConfig, sl.timestampConfig, sl.timestampMode)
	sl.list.SetDelegate(delegate)

	// Rebuild items
	items := buildListItems(sessionState, sl.sessionService, sl.statusConfig)
	sl.list.SetItems(items)
}

// pollStateCmd returns a command that waits 2 seconds then sends checkStateMsg
func pollStateCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return checkStateMsg{}
	})
}

// buildListItems converts SessionCollection to list items
func buildListItems(sessionState *domain.SessionCollection, sessionService *services.SessionService, statusConfig *config.StatusConfig) []list.Item {
	var items []list.Item

	// Build sessions from state
	sessionsMap := make(map[string]*ports.TmuxSession)
	for name, info := range sessionState.Sessions {
		sessionsMap[name] = &ports.TmuxSession{
			Name:      name,
			CreatedAt: info.LastUpdated,
		}
	}

	// Build ordered sessions list using OrderedNames from database
	var sessions []*ports.TmuxSession
	orderedSet := make(map[string]bool)

	// Add sessions in database order
	for _, name := range sessionState.OrderedNames {
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
			hasShell = sessionService.SessionExists(info.ShellSession.Name)
		}

		// Append git stats if available
		if info.GitStats != nil {
			stats := info.GitStats
			if stats.Error != nil {
				// Log the error to help debug why some repos don't show info
				logging.Logger.Debug("Git stats error for session",
					"session", session.Name,
					"error", stats.Error)
			} else {
				// Add ahead/behind (if non-zero)
				if stats.Ahead > 0 || stats.Behind > 0 {
					gitRef += fmt.Sprintf(" | ↑%d ↓%d", stats.Ahead, stats.Behind)
				}

				// Add file stats (if non-zero)
				if stats.ChangedFiles > 0 || stats.Additions > 0 || stats.Deletions > 0 {
					gitRef += fmt.Sprintf(" | %d files +%d -%d", stats.ChangedFiles, stats.Additions, stats.Deletions)
				}
			}
		}

		items = append(items, SessionItem{
			Comment:         info.Comment,
			DisplayName:     displayName,
			GitRef:          gitRef,
			HasShellSession: hasShell,
			IsFlagged:       info.IsFlagged,
			LastUpdated:     info.LastUpdated,
			Session:         session,
			State:           string(info.State),
			Status:          info.Status,
		})
	}

	return items
}

// renderStatusLegend renders the status legend with counts
func (sl *SessionList) renderStatusLegend() string {
	workingCount, idleCount, waitingCount, exitedCount := sl.countSessionsByState()

	legend := theme.WorkingIconStyle.Render(domain.SymbolWorking) + fmt.Sprintf(" %d working • ", workingCount)
	legend += theme.IdleIconStyle.Render(domain.SymbolIdle) + fmt.Sprintf(" %d idle • ", idleCount)
	legend += theme.WaitingIconStyle.Render(domain.SymbolWaiting) + fmt.Sprintf(" %d waiting • ", waitingCount)
	legend += theme.ExitedIconStyle.Render(domain.SymbolExited) + fmt.Sprintf(" %d exited", exitedCount)

	return legend
}

// countSessionsByState counts sessions by their state
func (sl *SessionList) countSessionsByState() (working, idle, waiting, exited int) {
	for _, sessionInfo := range sl.sessionState.Sessions {
		switch sessionInfo.State {
		case domain.StateWorking:
			working++
		case domain.StateIdle:
			idle++
		case domain.StateWaiting:
			waiting++
		case domain.StateExited:
			exited++
		}
	}
	return
}

// formatHelpLine formats a help line with styled shortcuts and labels
func formatHelpLine(line string) string {
	// Split by bullet separator
	parts := strings.Split(line, " • ")
	var formatted []string

	for _, part := range parts {
		// Split by colon to separate shortcut from label
		colonIdx := strings.Index(part, ": ")
		if colonIdx == -1 {
			// No colon found, just render as-is
			formatted = append(formatted, part)
			continue
		}

		shortcut := part[:colonIdx]
		label := part[colonIdx+2:] // +2 to skip ": "

		// Style shortcut and label separately
		styledShortcut := theme.HelpShortcutStyle.Render(shortcut)
		styledLabel := theme.HelpLabelStyle.Render(": " + label)

		formatted = append(formatted, styledShortcut+styledLabel)
	}

	// Join back with bullet separator
	return strings.Join(formatted, " • ")
}

// ensureSessionExists checks if a session exists and recreates it if needed
func (sl *SessionList) ensureSessionExists(session *ports.TmuxSession) bool {
	if sl.sessionService.SessionExists(session.Name) {
		return true
	}

	logging.Logger.Info("Session no longer exists, recreating", "name", session.Name)

	// Try to get stored metadata to recreate with same worktree and ClaudeDir
	var claudeDir string
	var worktreePath string
	if sessionInfo, ok := sl.sessionState.Sessions[session.Name]; ok {
		claudeDir = sessionInfo.ClaudeDir
		worktreePath = sessionInfo.WorktreePath
		logging.Logger.Info("Recreating session with stored worktree", "name", session.Name, "worktree", worktreePath, "claude_dir", claudeDir)
	} else {
		logging.Logger.Warn("No stored metadata for session, creating without worktree", "name", session.Name)
	}

	// Recreate the session
	if err := sl.sessionService.RecreateSession(session.Name, worktreePath, claudeDir, sl.tmuxStatusPosition); err != nil {
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

	// Store the name of the item we're moving (to find it after refresh)
	movedItemName := item.Session.Name

	// Debug: Log state before swap
	logging.Logger.Debug("Move up - before swap",
		"current_item", movedItemName,
		"current_index", currentIndex,
		"prev_item", prevItem.Session.Name,
		"prev_index", currentIndex-1,
		"total_items", len(items))

	// Swap positions in database
	if err := sl.sessionService.SwapPositions(context.Background(), movedItemName, prevItem.Session.Name); err != nil {
		logging.Logger.Warn("Failed to swap session positions", "error", err)
		return nil
	}

	// Reload state and rebuild list
	sl.RefreshFromState()

	// Find the new index of the moved item by name
	newItems := sl.list.Items()
	newIndex := -1
	for i, it := range newItems {
		if sessionItem, ok := it.(SessionItem); ok {
			logging.Logger.Debug("Move up - item order", "index", i, "name", sessionItem.Session.Name)
			if sessionItem.Session.Name == movedItemName {
				newIndex = i
			}
		}
	}

	// Debug: Log state after refresh
	logging.Logger.Debug("Move up - after refresh",
		"new_total_items", len(newItems),
		"moved_item", movedItemName,
		"new_index", newIndex,
		"expected_index", currentIndex-1)

	// Select the item at its new position
	if newIndex >= 0 {
		sl.list.Select(newIndex)
	}

	// Debug: Verify cursor position
	logging.Logger.Debug("Move up - after select",
		"selected_index", sl.list.Index(),
		"expected_name", movedItemName)

	// Don't schedule new poll - one is already running
	return nil
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

	// Store the name of the item we're moving (to find it after refresh)
	movedItemName := item.Session.Name

	// Debug: Log state before swap
	logging.Logger.Debug("Move down - before swap",
		"current_item", movedItemName,
		"current_index", currentIndex,
		"next_item", nextItem.Session.Name,
		"next_index", currentIndex+1,
		"total_items", len(items))

	// Swap positions in database
	if err := sl.sessionService.SwapPositions(context.Background(), movedItemName, nextItem.Session.Name); err != nil {
		logging.Logger.Warn("Failed to swap session positions", "error", err)
		return nil
	}

	// Reload state and rebuild list
	sl.RefreshFromState()

	// Find the new index of the moved item by name
	newItems := sl.list.Items()
	newIndex := -1
	for i, it := range newItems {
		if sessionItem, ok := it.(SessionItem); ok {
			logging.Logger.Debug("Move down - item order", "index", i, "name", sessionItem.Session.Name)
			if sessionItem.Session.Name == movedItemName {
				newIndex = i
			}
		}
	}

	// Debug: Log state after refresh
	logging.Logger.Debug("Move down - after refresh",
		"new_total_items", len(newItems),
		"moved_item", movedItemName,
		"new_index", newIndex,
		"expected_index", currentIndex+1)

	// Select the item at its new position
	if newIndex >= 0 {
		sl.list.Select(newIndex)
	}

	// Debug: Verify cursor position
	logging.Logger.Debug("Move down - after select",
		"selected_index", sl.list.Index(),
		"expected_name", movedItemName)

	// Don't schedule new poll - one is already running
	return nil
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
	var requests []GitStatsRequest
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
			if time.Since(info.GitStats.FetchedAt) < 5*time.Second {
				continue // Stats are fresh, skip
			}
		}

		// Add request with priority (selected = 1, others = 0)
		priority := 0
		if i == selectedIndex {
			priority = 1
		}

		requests = append(requests, GitStatsRequest{
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
		cmds = append(cmds, StartGitStatsFetcher(sl.gitService, req))
	}

	return tea.Batch(cmds...)
}

// cycleSessionStatus cycles the status of a session to the next value
func (sl *SessionList) cycleSessionStatus(sessionName string) tea.Cmd {
	// Get current status from session state
	var currentStatus *string
	if sessionInfo, ok := sl.sessionState.Sessions[sessionName]; ok {
		currentStatus = sessionInfo.Status
	}

	// Get next status in cycle
	nextStatus := sl.statusConfig.GetNextStatus(currentStatus)

	// Update in database
	if err := sl.sessionService.UpdateStatus(context.Background(), sessionName, nextStatus); err != nil {
		logging.Logger.Error("Failed to cycle session status", "error", err, "session", sessionName)
		return nil
	}

	// Log the change
	currentStr := "nil"
	if currentStatus != nil {
		currentStr = *currentStatus
	}
	nextStr := "nil"
	if nextStatus != nil {
		nextStr = *nextStatus
	}
	logging.Logger.Info("Cycled session status", "session", sessionName, "from", currentStr, "to", nextStr)

	// Refresh list immediately to show new status
	return func() tea.Msg {
		return checkStateMsg{}
	}
}
