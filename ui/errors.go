package ui

import (
	"strings"
	"unicode/utf8"
)

const (
	maxErrorLines  = 2
	errorPrefix    = "Error: "
	truncationMark = "..."
)

// formatErrorForDisplay formats an error message for TUI display.
// It limits the error to maxErrorLines (2 lines) and wraps text based on terminal width.
// The function accounts for the "Error: " prefix when calculating line width.
// If the error message is too long, it truncates with "..." at the end.
func formatErrorForDisplay(err error, maxWidth int) string {
	if err == nil {
		return ""
	}

	// Get the error message
	message := err.Error()
	if message == "" {
		return errorPrefix + "unknown error"
	}

	// Calculate available width per line (accounting for "Error: " prefix on first line)
	firstLineWidth := maxWidth - utf8.RuneCountInString(errorPrefix)
	if firstLineWidth < 10 {
		firstLineWidth = 10 // Minimum width to prevent edge cases
	}

	otherLineWidth := maxWidth
	if otherLineWidth < 10 {
		otherLineWidth = 10
	}

	// Split message into words
	words := strings.Fields(message)
	if len(words) == 0 {
		return errorPrefix + message
	}

	// Build lines
	var lines []string
	var currentLine strings.Builder
	currentLineWidth := firstLineWidth

	for _, word := range words {
		wordLen := utf8.RuneCountInString(word)
		currentLen := utf8.RuneCountInString(currentLine.String())

		// Check if adding this word would exceed the current line width
		if currentLen > 0 && currentLen+1+wordLen > currentLineWidth {
			// Save current line and start a new one
			lines = append(lines, currentLine.String())
			currentLine.Reset()

			// Check if we've reached max lines
			if len(lines) >= maxErrorLines {
				break
			}

			// Switch to other line width after first line
			currentLineWidth = otherLineWidth
		}

		// Add word to current line
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}

	// Add the last line if there's content and we haven't exceeded max lines
	if currentLine.Len() > 0 && len(lines) < maxErrorLines {
		lines = append(lines, currentLine.String())
	}

	// If we have exactly maxErrorLines and there are more words, add truncation mark
	if len(lines) == maxErrorLines && len(words) > 0 {
		lastLine := lines[maxErrorLines-1]
		truncLen := utf8.RuneCountInString(truncationMark)

		// If the last line is too long, truncate it to make room for "..."
		if utf8.RuneCountInString(lastLine)+truncLen > otherLineWidth {
			// Calculate how many runes we can keep
			maxRunes := otherLineWidth - truncLen
			if maxRunes > 0 {
				runes := []rune(lastLine)
				if len(runes) > maxRunes {
					lastLine = string(runes[:maxRunes])
				}
			}
		}

		lines[maxErrorLines-1] = lastLine + truncationMark
	}

	// Combine lines with newlines
	if len(lines) == 0 {
		return errorPrefix
	}

	// Add "Error: " prefix to first line
	result := errorPrefix + lines[0]
	if len(lines) > 1 {
		result += "\n" + strings.Join(lines[1:], "\n")
	}

	return result
}
