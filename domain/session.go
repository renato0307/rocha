package domain

import "time"

// SessionState represents the state of a Claude session
type SessionState string

const (
	StateExited  SessionState = "exited"
	StateIdle    SessionState = "idle"
	StateWaiting SessionState = "waiting"
	StateWorking SessionState = "working"
)

// Session represents a rocha session (domain entity)
type Session struct {
	AllowDangerouslySkipPermissions bool
	BranchName                      string
	ClaudeDir                       string
	Comment                         string
	DisplayName                     string
	ExecutionID                     string
	GitStats                        *GitStats
	IsArchived                      bool
	IsFlagged                       bool
	LastUpdated                     time.Time
	Name                            string
	RepoInfo                        string
	RepoPath                        string
	RepoSource                      string
	ShellSession                    *Session
	State                           SessionState
	Status                          *string
	WorktreePath                    string
}

// SessionCollection represents a collection of sessions with ordering
type SessionCollection struct {
	OrderedNames []string
	Sessions     map[string]Session
}
