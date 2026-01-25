package ui

import (
	"fmt"
	"strings"

	"rocha/internal/theme"

	"github.com/charmbracelet/bubbles/key"
)

// Tip holds a tip format string and the keys to highlight
type Tip struct {
	Format string
	Keys   []string
}

// tips is the private collection of all tips, populated by newTip()
var tips []Tip

// newTip registers a tip with format string and keys to highlight
// Format uses %s placeholders for keys, e.g. newTip("press %s to filter", "/")
func newTip(format string, keys ...string) string {
	tips = append(tips, Tip{Format: format, Keys: keys})
	// Return plain text for Tip field (used for filtering, etc.)
	args := make([]any, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	return fmt.Sprintf(format, args...)
}

// GetTips returns all registered tips
func GetTips() []Tip {
	return tips
}

// RenderTip formats a tip with highlighted keys and gray text
func RenderTip(tip Tip) string {
	// Split format by %s to get text segments
	parts := strings.Split(tip.Format, "%s")
	var result string
	result += theme.TipTextStyle.Render("ℹ  tip: ")
	for i, part := range parts {
		result += theme.TipTextStyle.Render(part)
		if i < len(tip.Keys) {
			result += theme.TipKeyStyle.Render(tip.Keys[i])
		}
	}
	return result
}

// KeyWithTip wraps a key.Binding with an optional tip for rotating tips display.
type KeyWithTip struct {
	Binding key.Binding
	Tip     string
}

// TipsConfig holds configuration for the tips feature
type TipsConfig struct {
	DisplayDurationSeconds int
	Enabled                bool
	ShowIntervalSeconds    int
}
