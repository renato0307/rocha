package domain

import (
	"strings"
	"time"
	"unicode"
)

// SessionState represents the state of a Claude session
type SessionState string

const (
	StateExited  SessionState = "exited"
	StateIdle    SessionState = "idle"
	StateWaiting SessionState = "waiting"
	StateWorking SessionState = "working"
)

// Status symbols (Unicode)
const (
	SymbolExited  = "■" // Gray - Claude has exited
	SymbolIdle    = "○" // Yellow - finished/idle
	SymbolWaiting = "◐" // Red - waiting for user input/prompt
	SymbolWorking = "●" // Green - actively working
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
	InitialPrompt                   string
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

// SanitizeSessionName converts a display name to a tmux-compatible session name.
// - Alphanumeric, underscores, hyphens, and periods are kept
// - Spaces, parentheses, and slashes become underscores (consecutive ones collapsed)
// - Special characters like []{}:;,!@#$%^&*+=|\/'"<>? are removed
func SanitizeSessionName(displayName string) string {
	var result strings.Builder
	lastWasUnderscore := false

	for _, r := range displayName {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '.' {
			// Keep alphanumeric, hyphens, and periods
			result.WriteRune(r)
			lastWasUnderscore = false
		} else if r == '_' {
			// Keep explicit underscores
			result.WriteRune('_')
			lastWasUnderscore = true
		} else if unicode.IsSpace(r) || r == '(' || r == ')' || r == '/' {
			// Replace spaces, parentheses, and slashes with underscore (avoid consecutive)
			if !lastWasUnderscore && result.Len() > 0 {
				result.WriteRune('_')
				lastWasUnderscore = true
			}
		}
		// All other special characters are removed
	}

	// Trim trailing underscore if any
	str := result.String()
	return strings.TrimRight(str, "_")
}
