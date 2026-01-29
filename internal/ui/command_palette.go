package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/theme"
)

// CommandPalette is a searchable action palette overlay.
type CommandPalette struct {
	Completed     bool
	Result        CommandPaletteResult
	actions       []domain.Action // Filtered actions
	allActions    []domain.Action // All available actions for context
	filterInput   textinput.Model
	height        int
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
	actions := domain.GetActionsForContext(hasSession)

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

// View renders the command palette.
func (cp *CommandPalette) View() string {
	// Build header
	var header string
	if cp.sessionName != "" {
		header = theme.PaletteHeaderStyle.Render("Session: " + cp.sessionName)
	} else {
		header = theme.PaletteHeaderStyle.Render("Command Palette")
	}

	// Build action list
	var items []string
	maxNameLen := cp.maxActionNameLen()

	for i, action := range cp.actions {
		name := padRight(action.Name, maxNameLen)
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

	actionList := strings.Join(items, "\n")

	// Build filter input
	filterLine := "> " + cp.filterInput.View()

	// Build footer
	footer := theme.PaletteFooterStyle.Render("↑↓ navigate • ⏎ select • esc cancel")

	// Combine all parts
	content := header + "\n" +
		strings.Repeat("─", cp.paletteWidth()-2) + "\n" +
		actionList + "\n" +
		strings.Repeat("─", cp.paletteWidth()-2) + "\n" +
		filterLine + "\n" +
		footer

	// Apply border
	bordered := theme.PaletteBorderStyle.Render(content)

	return bordered
}

// filterActions filters the action list based on the current input.
func (cp *CommandPalette) filterActions() {
	query := strings.ToLower(cp.filterInput.Value())
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

// maxActionNameLen returns the maximum action name length for alignment.
func (cp *CommandPalette) maxActionNameLen() int {
	maxLen := 0
	for _, action := range cp.actions {
		if len(action.Name) > maxLen {
			maxLen = len(action.Name)
		}
	}
	return maxLen
}

// paletteWidth returns the width of the palette content.
func (cp *CommandPalette) paletteWidth() int {
	// Fixed width or responsive based on terminal
	width := 60
	if cp.width > 0 && cp.width < 80 {
		width = cp.width - 10
	}
	if width < 40 {
		width = 40
	}
	return width
}

// padRight pads a string to the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
