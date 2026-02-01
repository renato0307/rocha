package theme

import "github.com/charmbracelet/lipgloss"

// Color is an alias for lipgloss.Color for convenience
type Color = lipgloss.Color

// Brand colors
const (
	ColorPrimary   Color = "99" // Purple - app name, titles
	ColorSecondary Color = "86" // Cyan - subtitles
)

// Session state colors
const (
	ColorExited  Color = "8" // Gray - exited
	ColorIdle    Color = "3" // Yellow - idle
	ColorWaiting Color = "1" // Red - waiting for user
	ColorWorking Color = "2" // Green - working
)

// UI semantic colors
const (
	ColorDimmed          Color = "240" // Dark gray - dimmed background content
	ColorError           Color = "196" // Bright red
	ColorHighlight       Color = "255" // White - emphasis
	ColorMuted           Color = "241" // Gray - secondary text
	ColorNormal          Color = "250" // Default text
	ColorPaletteSelected Color = "62"  // Purple - selected item background
	ColorScrollIndicator Color = "236" // Very dark gray - scroll arrows
	ColorSubtle          Color = "245" // Light gray - labels
	ColorVersion         Color = "240" // Dark gray
)

// Accent colors
const (
	ColorHelpGroup Color = "141" // Purple
	ColorHintKey   Color = "226" // Yellow - first session hint keys
	ColorHintLabel Color = "178" // Gold - first session hint labels
	ColorSpinner   Color = "205" // Pink
)

// Git colors
const (
	ColorAdditions Color = "2" // Green
	ColorDeletions Color = "1" // Red
)

// Token chart colors
const (
	ColorTokenInput  Color = "2"  // Green - input tokens
	ColorTokenOutput Color = "33" // Blue - output tokens
)

// DefaultStatusColors is the default color palette for implementation statuses
var DefaultStatusColors = []string{"141", "33", "214", "226", "46"}
