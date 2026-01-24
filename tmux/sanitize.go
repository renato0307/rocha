package tmux

import (
	"strings"
	"unicode"
)

// SanitizeSessionName converts a display name to a tmux-compatible session name.
// - Alphanumeric, underscores, hyphens, and periods are kept
// - Spaces, parentheses, and slashes become underscores (consecutive ones collapsed)
// - Special characters like []{}:;,!@#$%^&*+=|\/'"<>? are removed
func SanitizeSessionName(displayName string) string {
	var result strings.Builder
	lastWasUnderscore := false

	for _, r := range displayName {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '.' {
			// Keep alphanumeric, hyphens, and periods
			result.WriteRune(r)
			lastWasUnderscore = false
		} else if r == '_' {
			// Keep explicit underscores
			result.WriteRune('_')
			lastWasUnderscore = true
		} else if unicode.IsSpace(r) || r == '(' || r == ')' || r == '/' {
			// Replace spaces, parentheses, and slashes with underscore (avoid consecutive)
			if !lastWasUnderscore && result.Len() > 0 {
				result.WriteRune('_')
				lastWasUnderscore = true
			}
		}
		// All other special characters are removed
	}

	// Trim trailing underscore if any
	str := result.String()
	return strings.TrimRight(str, "_")
}
