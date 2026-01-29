package ui

import (
	"fmt"
	"strings"

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
	Completed     bool
	Result        CommandPaletteResult
	actions       []domain.Action // Filtered actions
	allActions    []domain.Action // All available actions for context
	filterInput   textinput.Model
	height        int
	lastQuery     string // Previous filter query (to detect changes)
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
func NewCommandPalette(session *ports.TmuxSession, sessionName string) *CommandPalette {
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
		switch msg.String() {
		case "esc", "ctrl+c":
			cp.Completed = true
			cp.Result.Cancelled = true
			return cp, nil

		case "enter":
			if len(cp.actions) > 0 && cp.selectedIndex < len(cp.actions) {
				cp.Completed = true
				cp.Result.Action = &cp.actions[cp.selectedIndex]
			}
			return cp, nil

		case "up", "ctrl+p":
			if cp.selectedIndex > 0 {
				cp.selectedIndex--
			}
			return cp, nil

		case "down", "ctrl+n":
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

	// Top border
	topBorder := theme.PaletteHeaderStyle.Render(strings.Repeat("─", width))

	// Filter input line
	filterLine := theme.PaletteFilterStyle.Render("> ") + cp.filterInput.View()

	// Build visible action list
	var items []string
	maxNameLen := cp.maxActionNameLen()
	start, end := cp.visibleRange()

	for i := start; i < end; i++ {
		action := cp.actions[i]
		name := padRight(action.DisplayName, maxNameLen)
		desc := action.Description

		if i == cp.selectedIndex {
			line := theme.PaletteItemSelectedStyle.Render("> "+name) +
				theme.PaletteDescSelectedStyle.Render("  "+desc)
			items = append(items, line)
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

	// Scroll indicator
	scrollInfo := ""
	if len(cp.actions) > maxVisibleItems {
		scrollInfo = theme.PaletteDescStyle.Render(
			fmt.Sprintf(" [%d/%d]", cp.selectedIndex+1, len(cp.actions)))
	}

	actionList := strings.Join(items, "\n")

	// Combine: border, filter, actions
	content := topBorder + "\n" +
		filterLine + scrollInfo + "\n" +
		actionList

	return content
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
	qi := 0
	for _, c := range target {
		if qi < len(query) && c == rune(query[qi]) {
			qi++
		}
	}
	return qi == len(query)
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
const maxVisibleItems = 5

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
