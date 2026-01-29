package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/theme"
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
	return theme.HelpKeyStyle.Render(key) + theme.HelpDescStyle.Render(description) + "\n"
}

// buildHelpContent builds the complete help text content using key bindings
func buildHelpContent(keys *KeyMap) string {
	var content string

	// Navigation
	content += theme.HelpGroupStyle.Render("Navigation") + "\n"
	content += renderBinding(keys.Navigation.Up.Binding)
	content += renderBinding(keys.Navigation.Down.Binding)
	content += renderBinding(keys.Navigation.MoveUp.Binding)
	content += renderBinding(keys.Navigation.MoveDown.Binding)
	content += renderBinding(keys.Navigation.Filter.Binding)
	content += renderBinding(keys.Navigation.ClearFilter.Binding)

	// Session Management
	content += "\n" + theme.HelpGroupStyle.Render("Session Management") + "\n"
	content += renderBinding(keys.SessionManagement.New.Binding)
	content += renderBinding(keys.SessionManagement.NewFromRepo.Binding)
	content += renderBinding(keys.SessionManagement.Rename.Binding)
	content += renderBinding(keys.SessionManagement.Archive.Binding)
	content += renderBinding(keys.SessionManagement.Kill.Binding)

	// Session Metadata
	content += "\n" + theme.HelpGroupStyle.Render("Session Metadata") + "\n"
	content += renderBinding(keys.SessionMetadata.Comment.Binding)
	content += renderBinding(keys.SessionMetadata.Flag.Binding)
	content += renderBinding(keys.SessionMetadata.StatusCycle.Binding)
	content += renderBinding(keys.SessionMetadata.StatusSetForm.Binding)

	// Experimental Features
	content += "\n" + theme.HelpGroupStyle.Render("Experimental Features") + "\n"
	content += renderShortcut(keys.SessionMetadata.SendText.Binding.Help().Key, keys.SessionMetadata.SendText.Binding.Help().Desc + " (experimental)")

	// Session Actions
	content += "\n" + theme.HelpGroupStyle.Render("Session Actions") + "\n"
	content += renderBinding(keys.SessionActions.Open.Binding)
	content += renderBinding(keys.SessionActions.Detach.Binding)
	content += renderBinding(keys.SessionActions.QuickOpen.Binding)
	content += renderBinding(keys.SessionActions.OpenShell.Binding)
	content += renderBinding(keys.SessionActions.OpenEditor.Binding)

	// Inside Session Shortcuts (tmux-level)
	content += "\n" + theme.HelpGroupStyle.Render("Inside Session Shortcuts") + "\n"
	content += renderShortcut(keys.SessionActions.Detach.Binding.Help().Key, "quick return to list")
	content += renderShortcut("ctrl+]", "swap between claude and shell sessions")
	content += renderShortcut("ctrl+b then d", "standard tmux detach (also works)")

	// Application
	content += "\n" + theme.HelpGroupStyle.Render("Application") + "\n"
	content += renderBinding(keys.Application.CommandPalette.Binding)
	content += renderBinding(keys.Application.Timestamps.Binding)
	content += renderBinding(keys.Application.TokenChart.Binding)
	content += renderBinding(keys.Application.Help.Binding)
	content += renderBinding(keys.Application.Quit.Binding)
	content += renderBinding(keys.Application.ForceQuit.Binding)

	// State Indicators
	content += "\n" + theme.HelpGroupStyle.Render("State Indicators (read-only)") + "\n"
	content += renderShortcut("●", "session is working")
	content += renderShortcut("○", "session is idle")
	content += renderShortcut("◐", "session is waiting")
	content += renderShortcut("■", "session has exited")
	content += renderShortcut("⚑", "session has flag set")
	content += renderShortcut("⌨", "session has comment")
	content += renderShortcut(">_", "shell session active")
	content += renderShortcut("[spec], [plan], etc.", "implementation status")

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
		if msg.String() == "esc" || key.Matches(msg, h.keys.Application.Quit.Binding, h.keys.Application.Help.Binding) {
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

	footer := theme.HelpStyle.Render("Press esc, q, h, or ? to close • ↑↓/jk/PgUp/PgDn to scroll")
	return h.viewport.View() + "\n\n" + footer
}

// renderBinding renders a single shortcut line from a key binding
func renderBinding(binding key.Binding) string {
	help := binding.Help()
	return renderShortcut(help.Key, help.Desc)
}
