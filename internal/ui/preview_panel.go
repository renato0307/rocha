package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
	previewBorderColor = lipgloss.Color("63")

	previewLineStyle = lipgloss.NewStyle().
				Foreground(previewBorderColor)

	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(previewBorderColor).
				PaddingLeft(1)
)

// PreviewPanel displays tmux pane content for the selected session
type PreviewPanel struct {
	height      int
	initialized bool
	sessionName string
	viewport    viewport.Model
	width       int
}

// NewPreviewPanel creates a new preview panel component
func NewPreviewPanel() *PreviewPanel {
	return &PreviewPanel{
		initialized: false,
		viewport:    viewport.New(0, 0),
	}
}

// SetContent updates the displayed content, keeping only the last N lines
// to create an auto-scroll effect where new content appears at the bottom
func (p *PreviewPanel) SetContent(content string) {
	if !p.initialized {
		return
	}

	// Split into lines and trim empty lines from bottom
	lines := strings.Split(content, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// Keep only the last viewportHeight lines for auto-scroll effect
	if len(lines) > p.viewport.Height {
		lines = lines[len(lines)-p.viewport.Height:]
	}
	content = strings.Join(lines, "\n")

	p.viewport.SetContent(content)
}

// SetSession updates the session name for the header
func (p *PreviewPanel) SetSession(name string) {
	p.sessionName = name
}

// SetSize handles resize and marks as initialized
func (p *PreviewPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Account for top line (1) + header (1) + bottom line (1) = 3 lines overhead
	viewportWidth := width
	viewportHeight := height - 3
	if viewportWidth < 1 {
		viewportWidth = 1
	}
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	p.viewport.Width = viewportWidth
	p.viewport.Height = viewportHeight
	p.initialized = true
}

// Height returns the current height of the preview panel
func (p *PreviewPanel) Height() int {
	return p.height
}

// Initialized returns whether the preview panel has been sized
func (p *PreviewPanel) Initialized() bool {
	return p.initialized
}

// View renders the preview panel
func (p *PreviewPanel) View() string {
	if !p.initialized {
		return ""
	}

	// Build horizontal line (double border character)
	line := previewLineStyle.Render(strings.Repeat("═", p.width))

	// Build header
	header := previewHeaderStyle.Render("Preview: " + p.sessionName)

	// Get content
	content := p.viewport.View()

	// Combine: top line + header + content + bottom line
	return line + "\n" + header + "\n" + content + "\n" + line
}
