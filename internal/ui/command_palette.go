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

// Command palette actions define available operations in the palette.
// These are displayed to users and processed in handlePaletteAction.
const (
	PaletteActionArchive  = "archive"   // Archive the selected session
	PaletteActionCopyInfo = "copy_info" // Copy session details to clipboard
	PaletteActionOpenPR   = "open_pr"   // Open a pull request on GitHub
	PaletteActionRebase   = "rebase"    // Rebase branch onto main
)

// Layout constants for command palette rendering
const (
	// paletteBorderPadding accounts for border (2) + internal padding (2)
	paletteBorderPadding = 4
	// paletteFixedLines is header(2) + separator + input + footer + border
	paletteFixedLines = 6
	// paletteInputPadding accounts for border, padding, and prompt
	paletteInputPadding = 8
	// paletteLineWidthPadding accounts for border (2) + padding (2) + prefix (2)
	paletteLineWidthPadding = 6
	// paletteMinSeparatorWidth is minimum width for visual separator
	paletteMinSeparatorWidth = 20
	// paletteNameColumnWidth is fixed width for command name alignment
	paletteNameColumnWidth = 24
)

// Key bindings for command palette navigation
var (
	paletteDownBinding  = key.NewBinding(key.WithKeys("down", "ctrl+n"))
	paletteEnterBinding = key.NewBinding(key.WithKeys("enter"))
	paletteEscBinding   = key.NewBinding(key.WithKeys("esc"))
	paletteUpBinding    = key.NewBinding(key.WithKeys("up", "ctrl+p"))
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
		case key.Matches(msg, paletteEscBinding):
			logging.Logger.Debug("Command palette cancelled")
			cp.Cancelled = true
			cp.Completed = true
			return cp, nil

		case key.Matches(msg, paletteEnterBinding):
			if len(cp.filteredItems) > 0 && cp.selectedIndex < len(cp.filteredItems) {
				cp.SelectedAction = cp.filteredItems[cp.selectedIndex].Action
				logging.Logger.Debug("Command palette selected", "action", cp.SelectedAction)
				cp.Completed = true
			}
			return cp, nil

		case key.Matches(msg, paletteUpBinding):
			cp.moveUp()
			return cp, nil

		case key.Matches(msg, paletteDownBinding):
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
		logging.Logger.Debug("Command palette filter cleared", "item_count", len(cp.filteredItems))
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

	logging.Logger.Debug("Command palette filtered", "query", query, "matches", len(cp.filteredItems))
}

// View implements tea.Model
func (cp *CommandPalette) View() string {
	var content strings.Builder

	cp.renderHeader(&content)
	cp.renderCommandList(&content)
	cp.renderFooter(&content)

	return theme.PaletteBorderStyle.Render(content.String())
}

// renderHeader renders the palette header with session name
func (cp *CommandPalette) renderHeader(content *strings.Builder) {
	content.WriteString(theme.PaletteHeaderStyle.Render("Session: " + cp.sessionName))
	content.WriteString("\n")
	separatorWidth := max(cp.width-paletteBorderPadding, paletteMinSeparatorWidth)
	content.WriteString(theme.PaletteSeparatorStyle.Render(strings.Repeat("─", separatorWidth)))
	content.WriteString("\n")
}

// renderCommandList renders the filtered command items
func (cp *CommandPalette) renderCommandList(content *strings.Builder) {
	for i, item := range cp.filteredItems {
		cp.renderCommandItem(content, i, item)
	}

	// Empty state
	if len(cp.filteredItems) == 0 {
		content.WriteString(theme.PaletteItemStyle.Render("  No matching commands"))
		content.WriteString("\n")
	}
}

// renderCommandItem renders a single command item
func (cp *CommandPalette) renderCommandItem(content *strings.Builder, index int, item CommandItem) {
	prefix := "  "
	nameStyle := theme.PaletteItemStyle
	descStyle := theme.PaletteDescriptionStyle
	isSelected := index == cp.selectedIndex

	if isSelected {
		prefix = "> "
		nameStyle = theme.PaletteSelectedStyle
		descStyle = theme.PaletteSelectedStyle
	}

	// Pad name to fixed width for alignment
	name := item.Name
	nameLen := lipgloss.Width(name)
	if nameLen < paletteNameColumnWidth {
		name = name + strings.Repeat(" ", paletteNameColumnWidth-nameLen)
	}

	line := nameStyle.Render(prefix+name) + descStyle.Render(item.Description)

	// Pad to full width for selected highlight
	if isSelected {
		lineWidth := lipgloss.Width(line)
		targetWidth := cp.width - paletteLineWidthPadding
		if lineWidth < targetWidth {
			line = line + descStyle.Render(strings.Repeat(" ", targetWidth-lineWidth))
		}
	}

	content.WriteString(line)
	content.WriteString("\n")
}

// renderFooter renders the input field and navigation hints
func (cp *CommandPalette) renderFooter(content *strings.Builder) {
	// Separator line before input
	separatorWidth := max(cp.width-paletteBorderPadding, paletteMinSeparatorWidth)
	content.WriteString(theme.PaletteSeparatorStyle.Render(strings.Repeat("─", separatorWidth)))
	content.WriteString("\n")

	// Input field
	content.WriteString(cp.textInput.View())
	content.WriteString("\n")

	// Footer with navigation hints
	content.WriteString(theme.PaletteFooterStyle.Render("↑↓ navigate • ⏎ select • esc cancel"))
}

// GetHeight returns the rendered height of the palette (for overlay positioning)
func (cp *CommandPalette) GetHeight() int {
	itemCount := len(cp.filteredItems)
	if itemCount == 0 {
		itemCount = 1 // "No matching commands" line
	}
	return itemCount + paletteFixedLines
}

// SetSize sets the width and height for the palette
func (cp *CommandPalette) SetSize(width, height int) {
	cp.width = width
	cp.height = height
	cp.textInput.Width = width - paletteInputPadding
}
