package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/theme"
)

// CommandPalette is a searchable action palette overlay.
type CommandPalette struct {
	actions       []KeyDefinition    // Filtered actions
	allActions    []KeyDefinition    // All available actions for context
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
	Action    *KeyDefinition
	Cancelled bool
}

// NewCommandPalette creates a new command palette.
// session can be nil if no session is selected.
// sessionName is the display name to show in the header.
// keys provides the key bindings for navigation.
func NewCommandPalette(session *ports.TmuxSession, sessionName string, keys KeyMap) *CommandPalette {
	actions := GetPaletteActions()

	ti := textinput.New()
	ti.Prompt = "Filter: "
	ti.PromptStyle = theme.FilterPromptStyle
	ti.Cursor.Style = theme.FilterCursorStyle
	ti.Placeholder = "type to filter"
	ti.PlaceholderStyle = theme.DimmedStyle
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

		case msg.Type == tea.KeyUp:
			if cp.selectedIndex > 0 {
				cp.selectedIndex--
			}
			return cp, nil

		case msg.Type == tea.KeyDown:
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

	// Header with session name (use inline styles to keep on same line)
	var header string
	titlePart := theme.PaletteTitleStyle.Render("⌘ Command Palette")
	if cp.sessionName != "" {
		header = titlePart + " " + theme.DimmedStyle.Render("(selected session: "+cp.sessionName+")")
	} else {
		header = titlePart
	}

	// Build visible action list
	var items []string
	maxHelpLen := cp.maxHelpLen()
	start, end := cp.visibleRange()
	total := len(cp.actions)
	hasMoreAbove := start > 0
	hasMoreBelow := end < total

	for i := start; i < end; i++ {
		def := cp.actions[i]
		helpText := padRight(capitalizeFirst(def.Help), maxHelpLen)
		shortcut := def.Defaults[0]

		// Determine prefix: selection indicator or scroll arrow
		var prefix string
		if i == cp.selectedIndex {
			prefix = "> "
		} else if i == start && hasMoreAbove {
			prefix = theme.ScrollIndicatorStyle.Render("↑ ")
		} else if i == end-1 && hasMoreBelow {
			prefix = theme.ScrollIndicatorStyle.Render("↓ ")
		} else {
			prefix = "  "
		}

		line := prefix +
			theme.PaletteItemStyle.Render(helpText) +
			theme.PaletteShortcutStyle.Render("  "+shortcut)
		items = append(items, line)
	}

	// If no matches
	if len(items) == 0 {
		items = append(items, theme.PaletteDescStyle.Render("  No matching actions"))
	}

	// Pad to fixed height
	for len(items) < maxVisibleItems {
		items = append(items, "")
	}

	actionList := strings.Join(items, "\n")

	// Filter input line
	filterLine := cp.filterInput.View()

	// Combine inner content
	innerContent := header + "\n" +
		"\n" +
		filterLine + "\n" +
		"\n" +
		actionList

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

	var filtered []KeyDefinition
	for _, def := range cp.allActions {
		if fuzzyMatch(query, def.Help) {
			filtered = append(filtered, def)
		}
	}

	cp.actions = filtered

	// Reset selection if out of bounds
	if cp.selectedIndex >= len(cp.actions) {
		cp.selectedIndex = 0
	}
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

// maxHelpLen returns the maximum help text length for alignment.
// Uses allActions to keep alignment stable during filtering.
func (cp *CommandPalette) maxHelpLen() int {
	maxLen := 0
	for _, def := range cp.allActions {
		if len(def.Help) > maxLen {
			maxLen = len(def.Help)
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

// capitalizeFirst returns the string with the first letter uppercased.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
