package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Dim style for background when overlay is shown
var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// compositeOverlay renders an overlay centered on top of a dimmed background.
// The background content is visible but dimmed, with the overlay rendered on top.
func compositeOverlay(background, overlay string, width, height int) string {
	// Split both into lines
	bgLines := strings.Split(background, "\n")
	overlayLines := strings.Split(overlay, "\n")

	// Use actual background height, but ensure minimum of terminal height
	actualHeight := len(bgLines)
	if height > actualHeight {
		actualHeight = height
	}

	// Ensure background has enough lines to fill the screen
	for len(bgLines) < actualHeight {
		bgLines = append(bgLines, "")
	}

	// Dim each background line and pad to full width
	for i := range bgLines {
		// Strip existing ANSI codes and apply dim style
		plainText := stripAnsi(bgLines[i])
		dimmedLine := dimStyle.Render(plainText)

		// Pad to full width if needed
		visibleWidth := lipgloss.Width(dimmedLine)
		if visibleWidth < width {
			dimmedLine = dimmedLine + strings.Repeat(" ", width-visibleWidth)
		}
		bgLines[i] = dimmedLine
	}

	// Calculate overlay dimensions
	overlayWidth := 0
	for _, line := range overlayLines {
		w := lipgloss.Width(line)
		if w > overlayWidth {
			overlayWidth = w
		}
	}
	overlayHeight := len(overlayLines)

	// Calculate centered position (use terminal height for centering)
	startX := (width - overlayWidth) / 2
	startY := (height - overlayHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	// Composite the overlay onto the dimmed background
	result := make([]string, len(bgLines))
	for y := 0; y < len(bgLines); y++ {
		if y >= startY && y < startY+overlayHeight {
			overlayLineIdx := y - startY
			if overlayLineIdx < len(overlayLines) {
				overlayLine := overlayLines[overlayLineIdx]
				overlayLineWidth := lipgloss.Width(overlayLine)

				// Build the line: dimmed left bg + overlay + dimmed right bg
				leftPad := dimStyle.Render(strings.Repeat(" ", startX))

				rightPadWidth := width - startX - overlayLineWidth
				if rightPadWidth < 0 {
					rightPadWidth = 0
				}
				rightPad := dimStyle.Render(strings.Repeat(" ", rightPadWidth))

				result[y] = leftPad + overlayLine + rightPad
			} else {
				result[y] = bgLines[y]
			}
		} else {
			result[y] = bgLines[y]
		}
	}

	return strings.Join(result, "\n")
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			// End of escape sequence at 'm' (SGR) or other terminator
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}

// bottomAnchoredOverlay renders an overlay anchored to the bottom of a dimmed background.
// The background content is visible but dimmed, with the overlay rendered at the bottom.
func bottomAnchoredOverlay(background, overlay string, width, height, overlayHeight int) string {
	// Split both into lines
	bgLines := strings.Split(background, "\n")
	overlayLines := strings.Split(overlay, "\n")

	// Use actual background height, but ensure minimum of terminal height
	actualHeight := len(bgLines)
	if height > actualHeight {
		actualHeight = height
	}

	// Ensure background has enough lines to fill the screen
	for len(bgLines) < actualHeight {
		bgLines = append(bgLines, "")
	}

	// Dim each background line and pad to full width
	for i := range bgLines {
		plainText := stripAnsi(bgLines[i])
		dimmedLine := dimStyle.Render(plainText)

		visibleWidth := lipgloss.Width(dimmedLine)
		if visibleWidth < width {
			dimmedLine = dimmedLine + strings.Repeat(" ", width-visibleWidth)
		}
		bgLines[i] = dimmedLine
	}

	// Calculate bottom-anchored position
	startY := height - overlayHeight
	if startY < 0 {
		startY = 0
	}

	// Composite the overlay onto the dimmed background at the bottom
	result := make([]string, len(bgLines))
	for y := 0; y < len(bgLines); y++ {
		if y >= startY && y < startY+len(overlayLines) {
			overlayLineIdx := y - startY
			if overlayLineIdx < len(overlayLines) {
				overlayLine := overlayLines[overlayLineIdx]
				overlayLineWidth := lipgloss.Width(overlayLine)

				// Pad overlay line to full width
				if overlayLineWidth < width {
					overlayLine = overlayLine + strings.Repeat(" ", width-overlayLineWidth)
				}
				result[y] = overlayLine
			} else {
				result[y] = bgLines[y]
			}
		} else {
			result[y] = bgLines[y]
		}
	}

	return strings.Join(result, "\n")
}
