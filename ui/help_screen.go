package ui

import (
	"github.com/charmbracelet/bubbles/key"
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
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Width(25)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

// HelpScreen displays keyboard shortcuts organized by category
type HelpScreen struct {
	Completed bool
	keys      *KeyMap
}

// NewHelpScreen creates a new help screen component
func NewHelpScreen(keys *KeyMap) *HelpScreen {
	return &HelpScreen{
		Completed: false,
		keys:      keys,
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
	content += h.renderBinding(h.keys.Navigation.Up)
	content += h.renderBinding(h.keys.Navigation.Down)
	content += h.renderBinding(h.keys.Navigation.MoveUp)
	content += h.renderBinding(h.keys.Navigation.MoveDown)
	content += h.renderBinding(h.keys.Navigation.Filter)
	content += h.renderBinding(h.keys.Navigation.ClearFilter)

	// Session Management
	content += "\n" + helpGroupStyle.Render("Session Management") + "\n"
	content += h.renderBinding(h.keys.SessionManagement.New)
	content += h.renderBinding(h.keys.SessionManagement.Rename)
	content += h.renderBinding(h.keys.SessionManagement.Archive)
	content += h.renderBinding(h.keys.SessionManagement.Kill)

	// Session Metadata
	content += "\n" + helpGroupStyle.Render("Session Metadata") + "\n"
	content += h.renderBinding(h.keys.SessionMetadata.Comment)
	content += h.renderBinding(h.keys.SessionMetadata.Flag)
	content += h.renderBinding(h.keys.SessionMetadata.StatusCycle)
	content += h.renderBinding(h.keys.SessionMetadata.StatusSetForm)

	// Session Actions
	content += "\n" + helpGroupStyle.Render("Session Actions") + "\n"
	content += h.renderBinding(h.keys.SessionActions.Open)
	content += h.renderBinding(h.keys.SessionActions.Detach)
	content += h.renderBinding(h.keys.SessionActions.QuickOpen)
	content += h.renderBinding(h.keys.SessionActions.OpenShell)
	content += h.renderBinding(h.keys.SessionActions.OpenEditor)

	// Application
	content += "\n" + helpGroupStyle.Render("Application") + "\n"
	content += h.renderBinding(h.keys.Application.Timestamps)
	content += h.renderBinding(h.keys.Application.Help)
	content += h.renderBinding(h.keys.Application.Quit)
	content += h.renderBinding(h.keys.Application.ForceQuit)

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

// renderBinding renders a single shortcut line from a key binding
func (h *HelpScreen) renderBinding(binding key.Binding) string {
	help := binding.Help()
	return h.renderShortcut(help.Key, help.Desc)
}

// renderShortcut renders a single shortcut line with key and description
func (h *HelpScreen) renderShortcut(key, description string) string {
	return helpKeyStyle.Render(key) + helpDescStyle.Render(description) + "\n"
}
