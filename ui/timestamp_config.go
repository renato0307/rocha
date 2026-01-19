package ui

// TimestampColorConfig holds configuration for timestamp display colors.
// Colors change based on how recently the session state was updated.
type TimestampColorConfig struct {
	RecentColor    string // Color for recent updates (< RecentMinutes)
	RecentMinutes  int    // Threshold in minutes for recent color
	StaleColor     string // Color for very old updates (>= WarningMinutes)
	WarningColor   string // Color for moderately old updates (>= RecentMinutes, < WarningMinutes)
	WarningMinutes int    // Threshold in minutes for warning color
}

// NewTimestampColorConfig creates a new TimestampColorConfig with the provided values.
func NewTimestampColorConfig(recentMin, warningMin int, recentColor, warningColor, staleColor string) *TimestampColorConfig {
	return &TimestampColorConfig{
		RecentMinutes:  recentMin,
		WarningMinutes: warningMin,
		RecentColor:    recentColor,
		WarningColor:   warningColor,
		StaleColor:     staleColor,
	}
}
