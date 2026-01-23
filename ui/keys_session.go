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
			Tip: newTip("press %s to archive a session (hidden from list)", "a"),
		},
		Kill: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("x"),
				key.WithHelp("x", "kill"),
			),
			Tip: newTip("press %s to kill a session and optionally remove its worktree", "x"),
		},
		New: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("n"),
				key.WithHelp("n", "new"),
			),
			Tip: newTip("press %s to create a new session", "n"),
		},
		NewFromRepo: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("N"),
				key.WithHelp("shift+n", "new from same repo"),
			),
			Tip: newTip("press %s to create a new session based on the selected session", "shift+n"),
		},
		Rename: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "rename"),
			),
			Tip: newTip("press %s to rename a session", "r"),
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
			Tip: newTip("press %s to add a comment to a session", "c"),
		},
		Flag: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "flag (⚑)"),
			),
			Tip: newTip("press %s to flag a session for attention", "f"),
		},
		SendText: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("p"),
				key.WithHelp("p", "send text (prompt)"),
			),
			Tip: newTip("press %s to send text to a session (experimental)", "p"),
		},
		StatusCycle: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "cycle status"),
			),
			Tip: newTip("press %s to cycle through implementation statuses", "s"),
		},
		StatusSetForm: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("S"),
				key.WithHelp("shift+s", "set status"),
			),
			Tip: newTip("press %s to pick a specific status", "shift+s"),
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
			Tip: newTip("press %s inside a session to return to the list", "ctrl+q"),
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
			Tip: newTip("press %s to open the session's folder in your editor", "o"),
		},
		OpenShell: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("ctrl+s"),
				key.WithHelp("ctrl+s", "shell (>_)"),
			),
			Tip: newTip("press %s to open a shell session alongside claude", "ctrl+s"),
		},
		QuickOpen: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9", "0"),
				key.WithHelp("1-9,0", "quick open (0=10th)"),
			),
			Tip: newTip("press %s to quickly open sessions by their number", "1-9,0"),
		},
	}
}
