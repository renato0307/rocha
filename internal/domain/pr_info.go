package domain

import "time"

// PRInfo represents GitHub PR information for a session
type PRInfo struct {
	CheckedAt time.Time // When PR was last checked
	Number    int       // PR number (0 if no PR)
	State     string    // open, closed, merged
	URL       string    // PR URL for browser
}
