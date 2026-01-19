package ui

import "github.com/charmbracelet/bubbles/key"

// SessionManagementKeys defines key bindings for managing sessions (create, rename, archive, kill)
type SessionManagementKeys struct {
	Archive key.Binding
	Kill    key.Binding
	New     key.Binding
	Rename  key.Binding
}

// SessionMetadataKeys defines key bindings for session metadata (comment, flag, status)
type SessionMetadataKeys struct {
	Comment         key.Binding
	Flag            key.Binding
	StatusCycle     key.Binding
	StatusSetForm   key.Binding
}

// SessionActionsKeys defines key bindings for session actions (open, shell, editor, quick open)
type SessionActionsKeys struct {
	Detach     key.Binding
	Open       key.Binding
	OpenEditor key.Binding
	OpenShell  key.Binding
	QuickOpen  key.Binding
}

// newSessionManagementKeys creates session management key bindings
func newSessionManagementKeys() SessionManagementKeys {
	return SessionManagementKeys{
		Archive: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "archive session"),
		),
		Kill: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "kill session"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new session"),
		),
		Rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename session"),
		),
	}
}

// newSessionMetadataKeys creates session metadata key bindings
func newSessionMetadataKeys() SessionMetadataKeys {
	return SessionMetadataKeys{
		Comment: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "comment (⌨)"),
		),
		Flag: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "flag (⚑)"),
		),
		StatusCycle: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "cycle status"),
		),
		StatusSetForm: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("shift+s", "set status"),
		),
	}
}

// newSessionActionsKeys creates session action key bindings
func newSessionActionsKeys() SessionActionsKeys {
	return SessionActionsKeys{
		Detach: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("ctrl+q", "detach from session (return to list)"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		OpenEditor: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "editor"),
		),
		OpenShell: key.NewBinding(
			key.WithKeys("alt+enter"),
			key.WithHelp("alt+enter", "shell (>_)"),
		),
		QuickOpen: key.NewBinding(
			key.WithKeys("alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7"),
			key.WithHelp("alt+1-7", "quick open"),
		),
	}
}
