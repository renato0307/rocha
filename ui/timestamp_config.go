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
// If values are zero/empty, defaults are applied:
//   - RecentMinutes: 5
//   - WarningMinutes: 20
//   - RecentColor: "241" (gray)
//   - WarningColor: "136" (amber/yellow)
//   - StaleColor: "208" (orange)
func NewTimestampColorConfig(recentMin, warningMin int, recentColor, warningColor, staleColor string) *TimestampColorConfig {
	config := &TimestampColorConfig{
		RecentMinutes:  recentMin,
		WarningMinutes: warningMin,
		RecentColor:    recentColor,
		WarningColor:   warningColor,
		StaleColor:     staleColor,
	}

	// Apply defaults if not set
	if config.RecentMinutes == 0 {
		config.RecentMinutes = 5
	}
	if config.WarningMinutes == 0 {
		config.WarningMinutes = 20
	}
	if config.RecentColor == "" {
		config.RecentColor = "241"
	}
	if config.WarningColor == "" {
		config.WarningColor = "136"
	}
	if config.StaleColor == "" {
		config.StaleColor = "208"
	}

	return config
}
