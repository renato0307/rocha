package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
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
	Completed   bool
	content     string         // Pre-built help content
	height      int            // Terminal height
	initialized bool           // Track if viewport has been sized
	keys        *KeyMap        // Key bindings to display
	viewport    viewport.Model // Scrollable viewport
	width       int            // Terminal width
}

// renderShortcut renders a single shortcut line with key and description
func renderShortcut(key, description string) string {
	return helpKeyStyle.Render(key) + helpDescStyle.Render(description) + "\n"
}

// buildHelpContent builds the complete help text content using key bindings
func buildHelpContent(keys *KeyMap) string {
	var content string

	// Navigation
	content += helpGroupStyle.Render("Navigation") + "\n"
	content += renderBinding(keys.Navigation.Up)
	content += renderBinding(keys.Navigation.Down)
	content += renderBinding(keys.Navigation.MoveUp)
	content += renderBinding(keys.Navigation.MoveDown)
	content += renderBinding(keys.Navigation.Filter)
	content += renderBinding(keys.Navigation.ClearFilter)

	// Session Management
	content += "\n" + helpGroupStyle.Render("Session Management") + "\n"
	content += renderBinding(keys.SessionManagement.New)
	content += renderBinding(keys.SessionManagement.Rename)
	content += renderBinding(keys.SessionManagement.Archive)
	content += renderBinding(keys.SessionManagement.Kill)

	// Session Metadata
	content += "\n" + helpGroupStyle.Render("Session Metadata") + "\n"
	content += renderBinding(keys.SessionMetadata.Comment)
	content += renderBinding(keys.SessionMetadata.Flag)
	content += renderBinding(keys.SessionMetadata.StatusCycle)
	content += renderBinding(keys.SessionMetadata.StatusSetForm)

	// Session Actions
	content += "\n" + helpGroupStyle.Render("Session Actions") + "\n"
	content += renderBinding(keys.SessionActions.Open)
	content += renderBinding(keys.SessionActions.Detach)
	content += renderBinding(keys.SessionActions.QuickOpen)
	content += renderBinding(keys.SessionActions.OpenShell)
	content += renderBinding(keys.SessionActions.OpenEditor)

	// Application
	content += "\n" + helpGroupStyle.Render("Application") + "\n"
	content += renderBinding(keys.Application.Timestamps)
	content += renderBinding(keys.Application.Help)
	content += renderBinding(keys.Application.Quit)
	content += renderBinding(keys.Application.ForceQuit)

	// State Indicators
	content += "\n" + helpGroupStyle.Render("State Indicators (read-only)") + "\n"
	content += renderShortcut("●", "Session is working")
	content += renderShortcut("○", "Session is idle")
	content += renderShortcut("◐", "Session is waiting")
	content += renderShortcut("■", "Session has exited")
	content += renderShortcut("⚑", "Session has flag set")
	content += renderShortcut("⌨", "Session has comment")
	content += renderShortcut(">_", "Shell session active")
	content += renderShortcut("[spec], [plan], etc.", "Implementation status")

	return content
}

// NewHelpScreen creates a new help screen component
func NewHelpScreen(keys *KeyMap) *HelpScreen {
	content := buildHelpContent(keys)
	return &HelpScreen{
		Completed:   false,
		content:     content,
		initialized: false,
		keys:        keys,
		viewport:    viewport.New(0, 0),
	}
}

// Init implements tea.Model
func (h *HelpScreen) Init() tea.Cmd {
	h.viewport.KeyMap.Up.SetKeys("up", "k")
	h.viewport.KeyMap.Down.SetKeys("down", "j")
	return nil
}

// Update implements tea.Model
func (h *HelpScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height

		// Dialog header: 4 lines, Footer: 2 lines
		viewportHeight := msg.Height - 6
		if viewportHeight < 5 {
			viewportHeight = 5
		}

		h.viewport.Width = msg.Width
		h.viewport.Height = viewportHeight
		h.viewport.SetContent(h.content)
		h.initialized = true
		return h, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "h", "?":
			h.Completed = true
			return h, nil
		}
	}

	var cmd tea.Cmd
	h.viewport, cmd = h.viewport.Update(msg)
	return h, cmd
}

// View implements tea.Model
func (h *HelpScreen) View() string {
	if !h.initialized {
		return "Loading help..."
	}

	footer := helpStyle.Render("Press esc, q, h, or ? to close • ↑↓/jk/PgUp/PgDn to scroll")
	return h.viewport.View() + "\n\n" + footer
}

// renderBinding renders a single shortcut line from a key binding
func renderBinding(binding key.Binding) string {
	help := binding.Help()
	return renderShortcut(help.Key, help.Desc)
}
