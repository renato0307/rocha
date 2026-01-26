package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/theme"
)

// Command palette actions (reusing from options_menu for consistency)
const (
	PaletteActionArchive  = "archive"
	PaletteActionCopyInfo = "copy_info"
	PaletteActionOpenPR   = "open_pr"
	PaletteActionRebase   = "rebase"
)

// CommandItem represents a single command in the palette
type CommandItem struct {
	Action      string
	Description string
	Name        string
}

// CommandPalette is a bottom-anchored command palette with fuzzy filtering
type CommandPalette struct {
	Cancelled      bool
	Completed      bool
	filteredItems  []CommandItem
	height         int
	items          []CommandItem
	SelectedAction string
	selectedIndex  int
	sessionName    string
	textInput      textinput.Model
	width          int
}

// NewCommandPalette creates a new command palette for a session
func NewCommandPalette(sessionName string) *CommandPalette {
	items := []CommandItem{
		{Action: PaletteActionRebase, Name: "Rebase on main", Description: "Rebase your branch onto main"},
		{Action: PaletteActionOpenPR, Name: "Open Pull Request", Description: "Create a PR on GitHub"},
		{Action: PaletteActionCopyInfo, Name: "Copy Session Info", Description: "Copy session details to clipboard"},
		{Action: PaletteActionArchive, Name: "Archive Session", Description: "Hide session from the list"},
	}

	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40

	cp := &CommandPalette{
		filteredItems: items, // Start with all items visible
		items:         items,
		sessionName:   sessionName,
		textInput:     ti,
	}

	return cp
}

// Init implements tea.Model
func (cp *CommandPalette) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (cp *CommandPalette) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cp.width = msg.Width
		cp.height = msg.Height
		return cp, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			logging.Logger.Debug("Command palette cancelled")
			cp.Cancelled = true
			cp.Completed = true
			return cp, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if len(cp.filteredItems) > 0 && cp.selectedIndex < len(cp.filteredItems) {
				cp.SelectedAction = cp.filteredItems[cp.selectedIndex].Action
				logging.Logger.Debug("Command palette selected", "action", cp.SelectedAction)
				cp.Completed = true
			}
			return cp, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "ctrl+p"))):
			cp.moveUp()
			return cp, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "ctrl+n"))):
			cp.moveDown()
			return cp, nil
		}
	}

	// Update text input and filter items
	var cmd tea.Cmd
	cp.textInput, cmd = cp.textInput.Update(msg)
	cp.filterItems()

	return cp, cmd
}

// moveUp moves selection up
func (cp *CommandPalette) moveUp() {
	if len(cp.filteredItems) == 0 {
		return
	}
	cp.selectedIndex--
	if cp.selectedIndex < 0 {
		cp.selectedIndex = len(cp.filteredItems) - 1
	}
}

// moveDown moves selection down
func (cp *CommandPalette) moveDown() {
	if len(cp.filteredItems) == 0 {
		return
	}
	cp.selectedIndex++
	if cp.selectedIndex >= len(cp.filteredItems) {
		cp.selectedIndex = 0
	}
}

// filterItems applies fuzzy matching to filter commands
func (cp *CommandPalette) filterItems() {
	query := cp.textInput.Value()

	if query == "" {
		cp.filteredItems = cp.items
		// Don't reset selectedIndex - preserve user's navigation
		if cp.selectedIndex >= len(cp.filteredItems) {
			cp.selectedIndex = 0
		}
		return
	}

	// Build list of names for fuzzy matching
	names := make([]string, len(cp.items))
	for i, item := range cp.items {
		names[i] = item.Name
	}

	// Perform fuzzy match
	matches := fuzzy.Find(query, names)

	// Build filtered items from matches
	cp.filteredItems = make([]CommandItem, len(matches))
	for i, match := range matches {
		cp.filteredItems[i] = cp.items[match.Index]
	}

	// Reset selection if out of bounds
	if cp.selectedIndex >= len(cp.filteredItems) {
		cp.selectedIndex = 0
	}
}

// View implements tea.Model
func (cp *CommandPalette) View() string {
	var content strings.Builder

	// Header with session name
	content.WriteString(theme.PaletteHeaderStyle.Render("Session: " + cp.sessionName))
	content.WriteString("\n")
	separatorWidth := max(cp.width-4, 20)
	content.WriteString(theme.PaletteSeparatorStyle.Render(strings.Repeat("─", separatorWidth)))
	content.WriteString("\n")

	// Command list (rendered above the input)
	nameColWidth := 24 // Fixed width for name column
	for i, item := range cp.filteredItems {
		prefix := "  "
		nameStyle := theme.PaletteItemStyle
		descStyle := theme.PaletteDescStyle
		if i == cp.selectedIndex {
			prefix = "> "
			nameStyle = theme.PaletteSelectedStyle
			descStyle = theme.PaletteSelectedStyle
		}

		// Pad name to fixed width for alignment
		name := item.Name
		nameLen := lipgloss.Width(name)
		if nameLen < nameColWidth {
			name = name + strings.Repeat(" ", nameColWidth-nameLen)
		}

		line := nameStyle.Render(prefix+name) + descStyle.Render(item.Description)

		// Pad to full width for selected highlight
		if i == cp.selectedIndex {
			lineWidth := lipgloss.Width(line)
			if lineWidth < cp.width-6 { // Account for border and padding
				line = line + descStyle.Render(strings.Repeat(" ", cp.width-6-lineWidth))
			}
		}
		content.WriteString(line)
		content.WriteString("\n")
	}

	// Empty state
	if len(cp.filteredItems) == 0 {
		content.WriteString(theme.PaletteItemStyle.Render("  No matching commands"))
		content.WriteString("\n")
	}

	// Separator line before input
	content.WriteString(theme.PaletteSeparatorStyle.Render(strings.Repeat("─", separatorWidth)))
	content.WriteString("\n")

	// Input field
	content.WriteString(cp.textInput.View())
	content.WriteString("\n")

	// Footer with navigation hints
	content.WriteString(theme.PaletteFooterStyle.Render("↑↓ navigate • ⏎ select • esc cancel"))

	// Wrap in border
	return theme.PaletteBorderStyle.Render(content.String())
}

// GetHeight returns the rendered height of the palette (for overlay positioning)
func (cp *CommandPalette) GetHeight() int {
	// Header (2: title + separator) + items + separator + input + footer + padding/border
	itemCount := len(cp.filteredItems)
	if itemCount == 0 {
		itemCount = 1 // "No matching commands" line
	}
	return itemCount + 6 // header(2) + separator + input + footer + border
}

// SetSize sets the width and height for the palette
func (cp *CommandPalette) SetSize(width, height int) {
	cp.width = width
	cp.height = height
	cp.textInput.Width = width - 8 // Account for border, padding, and prompt
}
