package theme

import "github.com/charmbracelet/lipgloss"

// Main UI styles
var (
	BranchStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	HelpLabelStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	HelpShortcutStyle = lipgloss.NewStyle().
				Foreground(ColorHighlight).
				Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(1, 0)

	NormalStyle = lipgloss.NewStyle().
			Foreground(ColorNormal)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(1, 0)
)

// State icon styles
var (
	ExitedIconStyle = lipgloss.NewStyle().
			Foreground(ColorExited)

	IdleIconStyle = lipgloss.NewStyle().
			Foreground(ColorIdle)

	WaitingIconStyle = lipgloss.NewStyle().
				Foreground(ColorWaiting)

	WorkingIconStyle = lipgloss.NewStyle().
				Foreground(ColorWorking)
)

// Git diff styles
var (
	AdditionsStyle = lipgloss.NewStyle().
			Foreground(ColorAdditions)

	DeletionsStyle = lipgloss.NewStyle().
			Foreground(ColorDeletions)
)

// Dialog header styles
var (
	AppNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	SubtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	TaglineStyle = lipgloss.NewStyle().
			Foreground(ColorNormal)

	VersionStyle = lipgloss.NewStyle().
			Foreground(ColorVersion)
)

// Help screen styles
var (
	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	HelpGroupStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHelpGroup).
			MarginTop(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true).
			Width(25)
)

// Tip styles
var (
	TipKeyStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	TipTextStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)
)

// First-session hint styles
var (
	HintKeyStyle = lipgloss.NewStyle().
			Foreground(ColorHintKey).
			Bold(true)

	HintLabelStyle = lipgloss.NewStyle().
			Foreground(ColorHintLabel)
)

// Spinner style
var SpinnerStyle = lipgloss.NewStyle().
	Foreground(ColorSpinner)

// Error style
var ErrorStyle = lipgloss.NewStyle().
	Foreground(ColorError).
	Bold(true)

// StatusStyle returns a style for a given status color string
func StatusStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

// TimestampStyle returns a style for a given timestamp color string
func TimestampStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

// Token chart styles
var (
	TokenInputStyle = lipgloss.NewStyle().
			Foreground(ColorTokenInput)

	TokenOutputStyle = lipgloss.NewStyle().
				Foreground(ColorTokenOutput)

	TokenChartLegendStyle = lipgloss.NewStyle().
				Foreground(ColorSubtle)
)
