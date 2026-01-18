package storage

import "time"

// SessionState represents the complete state (compatible with old state package)
type SessionState struct {
	Sessions            map[string]SessionInfo
	OrderedSessionNames []string // Populated from database ORDER BY position
}

// SessionInfo represents a session (compatible with old state package)
type SessionInfo struct {
	BranchName   string
	DisplayName  string
	ExecutionID  string
	GitStats     interface{}
	IsFlagged    bool
	LastUpdated  time.Time
	Name         string
	RepoInfo     string
	RepoPath     string
	ShellSession *SessionInfo
	State        string
	WorktreePath string
}
