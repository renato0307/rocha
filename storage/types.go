package storage

import "time"

// SessionState represents the complete state (compatible with old state package)
type SessionState struct {
	Sessions            map[string]SessionInfo
	OrderedSessionNames []string // Populated from database ORDER BY position
}

// SessionInfo represents a session (compatible with old state package)
type SessionInfo struct {
	AllowDangerouslySkipPermissions bool
	BranchName                      string
	ClaudeDir                       string
	Comment                         string
	DisplayName                     string
	ExecutionID                     string
	GitStats                        interface{}
	IsArchived                      bool
	IsFlagged                       bool
	LastUpdated                     time.Time
	Name                            string
	RepoInfo                        string
	RepoPath                        string
	RepoSource                      string
	ShellSession                    *SessionInfo
	State                           string
	Status                          *string // Implementation status (nil = no status set)
	WorktreePath                    string
}
