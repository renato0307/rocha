package ui

import (
	"fmt"

	"rocha/internal/theme"
)

// VersionInfo holds version information for display in UI headers.
// Populated by main.go from ldflags-injected values.
type VersionInfo struct {
	Commit    string
	Date      string
	GoVersion string
	Tagline   string
	Version   string
}

// DefaultVersionInfo provides default values when version info is not available
var DefaultVersionInfo = VersionInfo{
	Commit:    "unknown",
	Date:      "unknown",
	GoVersion: "unknown",
	Tagline:   "I'm Rocha, and I manage coding agents",
	Version:   "dev",
}

// versionInfo holds the global version info set by SetVersionInfo
var versionInfo = DefaultVersionInfo

// SetVersionInfo sets the global version info (called from main.go)
func SetVersionInfo(info VersionInfo) {
	versionInfo = info
}

// renderHeader creates a consistent header used across the entire application.
// It displays the app name with optional version info (in dev mode) and tagline.
// If subtitle is provided, it's rendered below the tagline (used for dialog form titles).
func renderHeader(devMode bool, subtitle string, _ string) string {
	// Build app name line (with optional version info)
	appNameLine := theme.AppNameStyle.Render("Rocha")
	if devMode {
		commit := versionInfo.Commit
		if len(commit) > 7 {
			commit = commit[:7] // Short commit hash
		}
		versionInfoStr := fmt.Sprintf(" %s | %s | %s | %s",
			versionInfo.Version,
			commit,
			versionInfo.Date,
			versionInfo.GoVersion)
		appNameLine += theme.VersionStyle.Render(versionInfoStr)
	}

	// Build tagline
	result := appNameLine + "\n"
	result += theme.TaglineStyle.Render(versionInfo.Tagline)

	// Add subtitle if provided (e.g., dialog form title)
	if subtitle != "" {
		result += "\n\n" + theme.SubtitleStyle.Render(subtitle)
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
