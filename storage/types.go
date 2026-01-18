package storage

import "time"

// SessionState represents the complete state (compatible with old state package)
type SessionState struct {
	Sessions            map[string]SessionInfo
	OrderedSessionNames []string // Populated from database ORDER BY position
}

// SessionInfo represents a session (compatible with old state package)
type SessionInfo struct {
	Name         string
	ShellSession *SessionInfo
	DisplayName  string
	State        string
	ExecutionID  string
	LastUpdated  time.Time
	RepoPath     string
	RepoInfo     string
	BranchName   string
	WorktreePath string
	GitStats     interface{}
}
