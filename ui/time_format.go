package ui

import (
	"fmt"
	"time"
)

// TimestampMode represents the display mode for timestamps
type TimestampMode int

const (
	TimestampHidden   TimestampMode = 0 // Don't show timestamps
	TimestampRelative TimestampMode = 1 // Show relative time (e.g., "5m ago")
	TimestampAbsolute TimestampMode = 2 // Show absolute time (e.g., "Jan 19 14:30")
)

// formatRelativeTime converts a timestamp to a human-readable relative time string.
// Returns empty string for zero times.
//
// Format:
//   - < 1 min: "just now"
//   - < 1 hour: "Xm ago"
//   - < 24 hours: "Xh ago"
//   - < 7 days: "Xd ago"
//   - < 30 days: "Xw ago"
//   - < 365 days: "Xmo ago"
//   - >= 365 days: "Xy ago"
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	elapsed := time.Since(t)

	// Less than 1 minute
	if elapsed < time.Minute {
		return "just now"
	}

	// Less than 1 hour
	if elapsed < time.Hour {
		minutes := int(elapsed.Minutes())
		return formatWithUnit(minutes, "m")
	}

	// Less than 24 hours
	if elapsed < 24*time.Hour {
		hours := int(elapsed.Hours())
		return formatWithUnit(hours, "h")
	}

	// Less than 7 days
	if elapsed < 7*24*time.Hour {
		days := int(elapsed.Hours() / 24)
		return formatWithUnit(days, "d")
	}

	// Less than 30 days
	if elapsed < 30*24*time.Hour {
		weeks := int(elapsed.Hours() / (24 * 7))
		return formatWithUnit(weeks, "w")
	}

	// Less than 365 days
	if elapsed < 365*24*time.Hour {
		months := int(elapsed.Hours() / (24 * 30))
		return formatWithUnit(months, "mo")
	}

	// 365 days or more
	years := int(elapsed.Hours() / (24 * 365))
	return formatWithUnit(years, "y")
}

// formatWithUnit creates a formatted string with value and unit followed by "ago"
func formatWithUnit(value int, unit string) string {
	return fmt.Sprintf("%d%s ago", value, unit)
}

// formatAbsoluteTime converts a timestamp to absolute date/time format.
// Returns empty string for zero times.
//
// Format: "YYYY-MM-DD HH:MM" (e.g., "2024-01-19 14:30")
func formatAbsoluteTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format("2006-01-02 15:04")
}

// getTimestampColor determines the color code based on how long ago the timestamp was.
// Recent updates use one color, older updates use warning color, very old use stale color.
func getTimestampColor(t time.Time, config *TimestampColorConfig) string {
	if t.IsZero() {
		return config.RecentColor
	}

	elapsed := time.Since(t).Minutes()

	if elapsed < float64(config.RecentMinutes) {
		return config.RecentColor
	} else if elapsed < float64(config.WarningMinutes) {
		return config.WarningColor
	}
	return config.StaleColor
}
