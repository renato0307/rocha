package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Help screen styles
	helpGroupStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("141")).
			MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Width(25)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))
)

// HelpScreen displays keyboard shortcuts organized by category
type HelpScreen struct {
	Completed bool
}

// NewHelpScreen creates a new help screen component
func NewHelpScreen() *HelpScreen {
	return &HelpScreen{
		Completed: false,
	}
}

// Init implements tea.Model
func (h *HelpScreen) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (h *HelpScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle key press
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc", "q", "h", "?":
			h.Completed = true
			return h, nil
		}
	}

	return h, nil
}

// View implements tea.Model
func (h *HelpScreen) View() string {
	var content string

	// Navigation
	content += helpGroupStyle.Render("Navigation") + "\n"
	content += h.renderShortcut("↑ / k", "Move up")
	content += h.renderShortcut("↓ / j", "Move down")
	content += h.renderShortcut("shift+K", "Move session up in list")
	content += h.renderShortcut("shift+J", "Move session down in list")
	content += h.renderShortcut("/", "Enter filter mode")
	content += h.renderShortcut("esc", "Clear filter (press twice within 500ms)")

	// Session Management
	content += "\n" + helpGroupStyle.Render("Session Management") + "\n"
	content += h.renderShortcut("n", "Create new session")
	content += h.renderShortcut("r", "Rename session")
	content += h.renderShortcut("x", "Kill session")

	// Session Metadata
	content += "\n" + helpGroupStyle.Render("Session Metadata") + "\n"
	content += h.renderShortcut("c", "Add/edit comment (shows ⌨ indicator)")
	content += h.renderShortcut("f", "Toggle flag (shows ⚑ indicator)")
	content += h.renderShortcut("s", "Quick cycle through statuses")
	content += h.renderShortcut("S (shift+S)", "Open status selection form")

	// Session Actions
	content += "\n" + helpGroupStyle.Render("Session Actions") + "\n"
	content += h.renderShortcut("enter", "Open/attach to session")
	content += h.renderShortcut("alt+1 through alt+7", "Quick open session by position")
	content += h.renderShortcut("alt+enter", "Open shell session (shows >_ indicator)")
	content += h.renderShortcut("o", "Open session in editor (requires worktree)")

	// Application
	content += "\n" + helpGroupStyle.Render("Application") + "\n"
	content += h.renderShortcut("h / ?", "Show this help screen")
	content += h.renderShortcut("q / ctrl+c", "Quit application")

	// State Indicators
	content += "\n" + helpGroupStyle.Render("State Indicators (read-only)") + "\n"
	content += h.renderShortcut("●", "Session is working")
	content += h.renderShortcut("○", "Session is idle")
	content += h.renderShortcut("◐", "Session is waiting")
	content += h.renderShortcut("■", "Session has exited")
	content += h.renderShortcut("⚑", "Session has flag set")
	content += h.renderShortcut("⌨", "Session has comment")
	content += h.renderShortcut(">_", "Shell session active")
	content += h.renderShortcut("[spec], [plan], etc.", "Implementation status")

	// Footer instruction
	content += "\n\n" + helpStyle.Render("Press esc, q, h, or ? to close")

	return content
}

// renderShortcut renders a single shortcut line with key and description
func (h *HelpScreen) renderShortcut(key, description string) string {
	return helpKeyStyle.Render(key) + helpDescStyle.Render(description) + "\n"
}
