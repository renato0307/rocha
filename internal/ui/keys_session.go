package ui

import (
	"rocha/internal/config"
)

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
func newSessionManagementKeys(keysConfig *config.KeyBindingsConfig) SessionManagementKeys {
	defaults := config.GetDefaultKeyBindings()

	return SessionManagementKeys{
		Archive:     buildBinding(defaults["archive"], keysConfig.GetBindingByName("archive"), "archive", "press %s to archive a session (hidden from list)"),
		Kill:        buildBinding(defaults["kill"], keysConfig.GetBindingByName("kill"), "kill", "press %s to kill a session and optionally remove its worktree"),
		New:         buildBinding(defaults["new"], keysConfig.GetBindingByName("new"), "new", "press %s to create a new session"),
		NewFromRepo: buildBinding(defaults["new_from_repo"], keysConfig.GetBindingByName("new_from_repo"), "new from same repo", "press %s to create a new session based on the selected session"),
		Rename:      buildBinding(defaults["rename"], keysConfig.GetBindingByName("rename"), "rename", "press %s to rename a session"),
	}
}

// newSessionMetadataKeys creates session metadata key bindings
func newSessionMetadataKeys(keysConfig *config.KeyBindingsConfig) SessionMetadataKeys {
	defaults := config.GetDefaultKeyBindings()

	return SessionMetadataKeys{
		Comment:       buildBinding(defaults["comment"], keysConfig.GetBindingByName("comment"), "comment (⌨)", "press %s to add a comment to a session"),
		Flag:          buildBinding(defaults["flag"], keysConfig.GetBindingByName("flag"), "flag (⚑)", "press %s to flag a session for attention"),
		SendText:      buildBinding(defaults["send_text"], keysConfig.GetBindingByName("send_text"), "send text (prompt)", "press %s to send text to a session (experimental)"),
		StatusCycle:   buildBinding(defaults["cycle_status"], keysConfig.GetBindingByName("cycle_status"), "cycle status", "press %s to cycle through implementation statuses"),
		StatusSetForm: buildBinding(defaults["set_status"], keysConfig.GetBindingByName("set_status"), "set status", "press %s to pick a specific status"),
	}
}

// newSessionActionsKeys creates session action key bindings
func newSessionActionsKeys(keysConfig *config.KeyBindingsConfig) SessionActionsKeys {
	defaults := config.GetDefaultKeyBindings()

	return SessionActionsKeys{
		Detach:     buildBinding(defaults["detach"], keysConfig.GetBindingByName("detach"), "detach from session (return to list)", "press %s inside a session to return to the list"),
		Open:       buildBinding(defaults["open"], keysConfig.GetBindingByName("open"), "open", ""),
		OpenEditor: buildBinding(defaults["open_editor"], keysConfig.GetBindingByName("open_editor"), "editor", "press %s to open the session's folder in your editor"),
		OpenShell:  buildBinding(defaults["open_shell"], keysConfig.GetBindingByName("open_shell"), "shell (>_)", "press %s to open a shell session alongside claude"),
		QuickOpen:  buildBinding(defaults["quick_open"], keysConfig.GetBindingByName("quick_open"), "quick open (0=10th)", "press %s to quickly open sessions by their number"),
	}
}
