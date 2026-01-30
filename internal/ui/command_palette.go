package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/theme"
)

// paletteExcludedActions lists actions that shouldn't appear in the command palette.
// These are "quick" actions with dedicated shortcuts that don't benefit from palette discovery.
var paletteExcludedActions = map[string]bool{
	"cycle_status": true, // Use 's' for quick cycling, 'S' for form
	"open":         true, // Use Enter directly
	"open_shell":   true, // Use dedicated shortcut
}

// CommandPalette is a searchable action palette overlay.
type CommandPalette struct {
	actions       []domain.Action    // Filtered actions
	allActions    []domain.Action    // All available actions for context
	Completed     bool
	filterInput   textinput.Model
	height        int
	keys          KeyMap             // Key bindings for navigation
	lastQuery     string             // Previous filter query (to detect changes)
	Result        CommandPaletteResult
	selectedIndex int
	session       *ports.TmuxSession // Selected session (can be nil)
	sessionName   string             // Display name for header
	width         int
}

// CommandPaletteResult contains the result of the command palette interaction.
type CommandPaletteResult struct {
	Action    *domain.Action
	Cancelled bool
}

// NewCommandPalette creates a new command palette.
// session can be nil if no session is selected.
// sessionName is the display name to show in the header.
// keys provides the key bindings for navigation.
func NewCommandPalette(session *ports.TmuxSession, sessionName string, keys KeyMap) *CommandPalette {
	hasSession := session != nil
	actions := filterActionsForPalette(domain.GetActionsForContext(hasSession))

	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40

	return &CommandPalette{
		actions:       actions,
		allActions:    actions,
		filterInput:   ti,
		keys:          keys,
		selectedIndex: 0,
		session:       session,
		sessionName:   sessionName,
	}
}

// Init initializes the command palette.
func (cp *CommandPalette) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the command palette.
func (cp *CommandPalette) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cp.width = msg.Width
		cp.height = msg.Height
		return cp, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, cp.keys.Navigation.ClearFilter.Binding) ||
			key.Matches(msg, cp.keys.Application.ForceQuit.Binding):
			cp.Completed = true
			cp.Result.Cancelled = true
			return cp, nil

		case key.Matches(msg, cp.keys.SessionActions.Open.Binding):
			if len(cp.actions) > 0 && cp.selectedIndex < len(cp.actions) {
				cp.Completed = true
				cp.Result.Action = &cp.actions[cp.selectedIndex]
			}
			return cp, nil

		case key.Matches(msg, cp.keys.Navigation.Up.Binding):
			if cp.selectedIndex > 0 {
				cp.selectedIndex--
			}
			return cp, nil

		case key.Matches(msg, cp.keys.Navigation.Down.Binding):
			if cp.selectedIndex < len(cp.actions)-1 {
				cp.selectedIndex++
			}
			return cp, nil
		}
	}

	// Update filter input
	var cmd tea.Cmd
	cp.filterInput, cmd = cp.filterInput.Update(msg)

	// Re-filter actions based on input
	cp.filterActions()

	return cp, cmd
}

// View renders the command palette as a full-width bottom panel.
func (cp *CommandPalette) View() string {
	width := cp.paletteWidth()
	// Inner content width (accounting for border and padding: 2 border + 2 padding = 4)
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Header with session name
	var header string
	if cp.sessionName != "" {
		header = theme.PaletteHeaderStyle.Render("Session: " + cp.sessionName)
	} else {
		header = theme.PaletteHeaderStyle.Render("Command Palette")
	}

	// Separator line (fits inner width)
	separator := theme.PaletteDescStyle.Render(strings.Repeat("─", innerWidth))

	// Build visible action list
	var items []string
	maxNameLen := cp.maxActionNameLen()
	start, end := cp.visibleRange()

	for i := start; i < end; i++ {
		action := cp.actions[i]
		name := padRight(action.DisplayName, maxNameLen)
		desc := action.Description

		if i == cp.selectedIndex {
			// Selected: full-width highlight (inner width)
			lineContent := "> " + name + "  " + desc
			lineContent = padRight(lineContent, innerWidth)
			items = append(items, theme.PaletteItemSelectedStyle.Render(lineContent))
		} else {
			line := theme.PaletteItemStyle.Render("  "+name) +
				theme.PaletteDescStyle.Render("  "+desc)
			items = append(items, line)
		}
	}

	// If no matches
	if len(items) == 0 {
		items = append(items, theme.PaletteDescStyle.Render("  No matching actions"))
	}

	actionList := strings.Join(items, "\n")

	// Filter input line (textinput already has "> " prompt)
	filterLine := cp.filterInput.View()

	// Combine inner content (no gap between header and separator, no footer)
	innerContent := header +
		separator + "\n" +
		actionList + "\n" +
		"\n" +
		filterLine

	// Apply border around entire palette
	bordered := theme.PaletteBorderStyle.Width(width - 2).Render(innerContent)

	return bordered
}

// filterActions filters the action list based on the current input.
func (cp *CommandPalette) filterActions() {
	query := strings.ToLower(cp.filterInput.Value())

	// Only refilter if query changed
	if query == cp.lastQuery {
		return
	}
	cp.lastQuery = query

	if query == "" {
		cp.actions = cp.allActions
		cp.selectedIndex = 0
		return
	}

	var filtered []domain.Action
	for _, action := range cp.allActions {
		if fuzzyMatch(query, action.Name) || fuzzyMatch(query, action.Description) {
			filtered = append(filtered, action)
		}
	}

	cp.actions = filtered

	// Reset selection if out of bounds
	if cp.selectedIndex >= len(cp.actions) {
		cp.selectedIndex = 0
	}
}

// filterActionsForPalette removes actions that shouldn't appear in the palette.
func filterActionsForPalette(actions []domain.Action) []domain.Action {
	var filtered []domain.Action
	for _, action := range actions {
		if !paletteExcludedActions[action.Name] {
			filtered = append(filtered, action)
		}
	}
	return filtered
}

// fuzzyMatch checks if all characters in query appear in order in target.
func fuzzyMatch(query, target string) bool {
	target = strings.ToLower(target)
	queryRunes := []rune(query)
	qi := 0
	for _, c := range target {
		if qi < len(queryRunes) && c == queryRunes[qi] {
			qi++
		}
	}
	return qi == len(queryRunes)
}

// maxActionNameLen returns the maximum display name length for alignment.
func (cp *CommandPalette) maxActionNameLen() int {
	maxLen := 0
	for _, action := range cp.actions {
		if len(action.DisplayName) > maxLen {
			maxLen = len(action.DisplayName)
		}
	}
	return maxLen
}

// paletteWidth returns the full terminal width.
func (cp *CommandPalette) paletteWidth() int {
	if cp.width > 0 {
		return cp.width
	}
	return 80 // fallback
}

// maxVisibleItems returns the maximum number of items to show at once.
const maxVisibleItems = 6

// visibleRange returns the start and end indices for visible items.
func (cp *CommandPalette) visibleRange() (int, int) {
	total := len(cp.actions)
	if total <= maxVisibleItems {
		return 0, total
	}

	// Keep selected item visible with some context
	start := cp.selectedIndex - maxVisibleItems/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisibleItems
	if end > total {
		end = total
		start = end - maxVisibleItems
	}
	return start, end
}

// padRight pads a string to the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
