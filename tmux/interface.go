package tmux

import (
	"errors"
	"os/exec"
)

// Error sentinels for consistent error handling
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExists   = errors.New("session already exists")
	ErrNotAttached     = errors.New("not attached to session")
	ErrAlreadyAttached = errors.New("already attached to session")
)

// SessionManager handles session lifecycle operations
type SessionManager interface {
	Create(name string, worktreePath string) (*Session, error)
	CreateShellSession(name string, worktreePath string) (*Session, error)
	Exists(name string) bool
	List() ([]*Session, error)
	Kill(name string) error
	Rename(oldName, newName string) error
}

// Attacher handles session attachment operations
type Attacher interface {
	Attach(sessionName string) (chan struct{}, error)
	Detach(sessionName string) error
	// GetAttachCommand returns an exec.Cmd configured for attaching to a session.
	// This is useful for integration with frameworks that need the command directly
	// (like Bubble Tea's tea.ExecProcess).
	GetAttachCommand(sessionName string) *exec.Cmd
}

// PaneOperations handles pane-level operations
type PaneOperations interface {
	SendKeys(sessionName string, keys ...string) error
	CapturePane(sessionName string, startLine int) (string, error)
}

// Configurator handles tmux configuration operations
type Configurator interface {
	SourceFile(configPath string) error
	BindKey(table, key, command string) error
}

// Client is a composite interface for commands that need multiple operations
type Client interface {
	SessionManager
	Attacher
	PaneOperations
	Configurator
}
