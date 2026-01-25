package domain

import "time"

// GitStats holds detailed git statistics for a worktree
type GitStats struct {
	Additions    int       // Lines added in working directory
	Ahead        int       // Commits ahead of tracking branch
	Behind       int       // Commits behind tracking branch
	ChangedFiles int       // Number of changed files in working directory
	Deletions    int       // Lines deleted in working directory
	Error        error     // Error during fetching (if any)
	FetchedAt    time.Time // When these stats were fetched
}
