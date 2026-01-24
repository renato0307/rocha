package ui

import "github.com/charmbracelet/bubbles/key"

// SessionManagementKeys defines key bindings for managing sessions (create, rename, archive, kill)
type SessionManagementKeys struct {
	Archive     KeyWithTip
	Kill        KeyWithTip
	New         KeyWithTip
	NewFromRepo KeyWithTip
	Rename      KeyWithTip
}

// SessionMetadataKeys defines key bindings for session metadata (comment, flag, status)
type SessionMetadataKeys struct {
	Comment       KeyWithTip
	Flag          KeyWithTip
	SendText      KeyWithTip
	StatusCycle   KeyWithTip
	StatusSetForm KeyWithTip
}

// SessionActionsKeys defines key bindings for session actions (open, shell, editor, quick open)
type SessionActionsKeys struct {
	Detach     KeyWithTip
	Open       KeyWithTip
	OpenEditor KeyWithTip
	OpenShell  KeyWithTip
	QuickOpen  KeyWithTip
}

// newSessionManagementKeys creates session management key bindings
func newSessionManagementKeys() SessionManagementKeys {
	return SessionManagementKeys{
		Archive: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("a"),
				key.WithHelp("a", "archive"),
			),
			Tip: newTip("press 'a' to archive a session (hidden from list)"),
		},
		Kill: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("x"),
				key.WithHelp("x", "kill"),
			),
			Tip: newTip("press 'x' to kill a session and optionally remove its worktree"),
		},
		New: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("n"),
				key.WithHelp("n", "new"),
			),
			Tip: newTip("press 'n' to create a new session"),
		},
		NewFromRepo: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("N"),
				key.WithHelp("shift+n", "new from same repo"),
			),
			Tip: newTip("press 'shift+N' to create a new session based on the selected session"),
		},
		Rename: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "rename"),
			),
			Tip: newTip("press 'r' to rename a session"),
		},
	}
}

// newSessionMetadataKeys creates session metadata key bindings
func newSessionMetadataKeys() SessionMetadataKeys {
	return SessionMetadataKeys{
		Comment: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "comment (⌨)"),
			),
			Tip: newTip("press 'c' to add a comment to a session"),
		},
		Flag: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "flag (⚑)"),
			),
			Tip: newTip("press 'f' to flag a session for attention"),
		},
		SendText: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("p"),
				key.WithHelp("p", "send text (prompt)"),
			),
			Tip: newTip("press 'p' to send text to a session (experimental)"),
		},
		StatusCycle: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "cycle status"),
			),
			Tip: newTip("press 's' to cycle through implementation statuses"),
		},
		StatusSetForm: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("S"),
				key.WithHelp("shift+s", "set status"),
			),
			Tip: newTip("press 'shift+S' to pick a specific status"),
		},
	}
}

// newSessionActionsKeys creates session action key bindings
func newSessionActionsKeys() SessionActionsKeys {
	return SessionActionsKeys{
		Detach: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("ctrl+q"),
				key.WithHelp("ctrl+q", "detach from session (return to list)"),
			),
			Tip: newTip("press 'ctrl+q' inside a session to return to the list"),
		},
		Open: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "open"),
			),
		},
		OpenEditor: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("o"),
				key.WithHelp("o", "editor"),
			),
			Tip: newTip("press 'o' to open the session's folder in your editor"),
		},
		OpenShell: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("alt+enter"),
				key.WithHelp("alt+enter", "shell (>_)"),
			),
			Tip: newTip("press 'alt+enter' to open a shell session alongside claude"),
		},
		QuickOpen: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7"),
				key.WithHelp("alt+1-7", "quick open"),
			),
			Tip: newTip("press 'alt+1-7' to quickly open sessions by their number"),
		},
	}
}
