package ports

import (
	"errors"
	"os/exec"
	"time"
)

// Error sentinels for tmux operations
var (
	ErrTmuxAlreadyAttached = errors.New("already attached to tmux session")
	ErrTmuxNotAttached     = errors.New("not attached to tmux session")
	ErrTmuxSessionExists   = errors.New("tmux session already exists")
	ErrTmuxSessionNotFound = errors.New("tmux session not found")
)

// TmuxSession represents a tmux session
type TmuxSession struct {
	CreatedAt time.Time
	Name      string
}

// TmuxSessionLifecycle handles tmux session lifecycle operations
type TmuxSessionLifecycle interface {
	CreateSession(name, worktreePath, claudeDir, statusPosition string) (*TmuxSession, error)
	CreateShellSession(name, worktreePath, statusPosition string) (*TmuxSession, error)
	KillSession(name string) error
	ListSessions() ([]*TmuxSession, error)
	RenameSession(oldName, newName string) error
	SessionExists(name string) bool
}

// TmuxSessionAttacher handles tmux session attachment
type TmuxSessionAttacher interface {
	Attach(sessionName string) (chan struct{}, error)
	Detach(sessionName string) error
	GetAttachCommand(sessionName string) *exec.Cmd
}

// TmuxPaneController handles tmux pane operations
type TmuxPaneController interface {
	CapturePane(sessionName string, startLine int) (string, error)
	SendKeys(sessionName string, keys ...string) error
}

// TmuxConfigurator handles tmux configuration
type TmuxConfigurator interface {
	BindKey(table, key, command string) error
	SetOption(sessionName, option, value string) error
	SourceFile(configPath string) error
}

// TmuxClient is the composite tmux interface
type TmuxClient interface {
	TmuxConfigurator
	TmuxPaneController
	TmuxSessionAttacher
	TmuxSessionLifecycle
}
