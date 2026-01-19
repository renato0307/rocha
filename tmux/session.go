package tmux

import (
	"time"
)

const (
	// DefaultStatusPosition is the default tmux status bar position
	DefaultStatusPosition = "bottom"
)

// Session represents a tmux session (data-only struct)
type Session struct {
	Name      string
	CreatedAt time.Time
}

// Package-level default client for backward compatibility
var defaultClient = NewClient()

// NewSession creates a new tmux session with the given name
// If worktreePath is provided (non-empty), the session will start in that directory
// This is a backward-compatible wrapper around DefaultClient.Create
func NewSession(name string, worktreePath string) (*Session, error) {
	return defaultClient.Create(name, worktreePath, DefaultStatusPosition)
}

// List returns all active tmux sessions
// This is a backward-compatible wrapper around DefaultClient.List
func List() ([]*Session, error) {
	return defaultClient.List()
}

// Exists checks if the tmux session exists
func (s *Session) Exists() bool {
	return defaultClient.Exists(s.Name)
}

// Kill terminates the tmux session
func (s *Session) Kill() error {
	return defaultClient.Kill(s.Name)
}

// Attach attaches to the tmux session. Returns a channel that will be closed when detached.
func (s *Session) Attach() (chan struct{}, error) {
	return defaultClient.Attach(s.Name)
}

// Detach detaches from the tmux session
func (s *Session) Detach() error {
	return defaultClient.Detach(s.Name)
}
