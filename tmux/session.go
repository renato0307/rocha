package tmux

import (
	"time"

	"rocha/ports"
)

const (
	// DefaultStatusPosition is the default tmux status bar position
	DefaultStatusPosition = "bottom"
)

// Session is a local type for backward compatibility
// New code should use ports.TmuxSession directly
type Session struct {
	CreatedAt time.Time
	Name      string
}

// Package-level default client for backward compatibility
var defaultClient = NewClient()

// NewSession creates a new tmux session with the given name
// If worktreePath is provided (non-empty), the session will start in that directory
// This is a backward-compatible wrapper around DefaultClient.CreateSession
func NewSession(name string, worktreePath string) (*Session, error) {
	portsSess, err := defaultClient.CreateSession(name, worktreePath, "", DefaultStatusPosition)
	if err != nil {
		return nil, err
	}
	return toLocalSession(portsSess), nil
}

// List returns all active tmux sessions
// This is a backward-compatible wrapper around DefaultClient.ListSessions
func List() ([]*Session, error) {
	portsSessions, err := defaultClient.ListSessions()
	if err != nil {
		return nil, err
	}
	sessions := make([]*Session, len(portsSessions))
	for i, ps := range portsSessions {
		sessions[i] = toLocalSession(ps)
	}
	return sessions, nil
}

// toLocalSession converts a ports.TmuxSession to a local Session
func toLocalSession(ps *ports.TmuxSession) *Session {
	if ps == nil {
		return nil
	}
	return &Session{
		CreatedAt: ps.CreatedAt,
		Name:      ps.Name,
	}
}
