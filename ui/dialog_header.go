package ui

import (
	"fmt"
	"rocha/version"

	"github.com/charmbracelet/lipgloss"
)

// renderHeader creates a consistent header used across the entire application.
// It displays the app name with optional version info (in dev mode) and tagline.
// If subtitle is provided, it's rendered below the tagline (used for dialog form titles).
func renderHeader(devMode bool, subtitle string, _ string) string {
	// Title style - matches session list (color 99, bold)
	appNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))

	// Version info style - grey, shown next to title in dev mode
	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// Tagline style - matches session list (color 250)
	taglineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))

	// Subtitle style - used for dialog form titles
	subtitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	// Build app name line (with optional version info)
	appNameLine := appNameStyle.Render("Rocha")
	if devMode {
		versionInfo := fmt.Sprintf(" %s | %s | %s | %s",
			version.Version,
			version.Commit[:7], // Short commit hash
			version.Date,
			version.GoVersion)
		appNameLine += versionStyle.Render(versionInfo)
	}

	// Build tagline
	result := appNameLine + "\n"
	result += taglineStyle.Render(version.Tagline)

	// Add subtitle if provided (e.g., dialog form title)
	if subtitle != "" {
		result += "\n\n" + subtitleStyle.Render(subtitle)
	}

	result += "\n"
	return result
}

// renderDialogHeader creates a header for dialogs with a form title.
// This is a convenience wrapper around renderHeader for backward compatibility.
//
// NOTE: This function should ONLY be called by the Dialog wrapper in dialog.go.
// Do not call this function directly from form components. Instead, wrap your
// form component in a Dialog using NewDialog(), which will automatically add
// the header. This enforces consistent headers "by design" across all dialogs.
func renderDialogHeader(devMode bool, formTitle string) string {
	return renderHeader(devMode, formTitle, "") // No tip for dialogs
}
