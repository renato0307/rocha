package ui

import (
	"github.com/renato0307/rocha/internal/config"
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
	Detach      KeyWithTip
	Open        KeyWithTip
	OpenEditor  KeyWithTip
	OpenShell   KeyWithTip
	OptionsMenu KeyWithTip
	QuickOpen   KeyWithTip
}

// newSessionManagementKeys creates session management key bindings
func newSessionManagementKeys(defaults map[string][]string, customKeys config.KeyBindingsConfig) SessionManagementKeys {
	return SessionManagementKeys{
		Archive:     buildBinding("archive", defaults, customKeys),
		Kill:        buildBinding("kill", defaults, customKeys),
		New:         buildBinding("new", defaults, customKeys),
		NewFromRepo: buildBinding("new_from_repo", defaults, customKeys),
		Rename:      buildBinding("rename", defaults, customKeys),
	}
}

// newSessionMetadataKeys creates session metadata key bindings
func newSessionMetadataKeys(defaults map[string][]string, customKeys config.KeyBindingsConfig) SessionMetadataKeys {
	return SessionMetadataKeys{
		Comment:       buildBinding("comment", defaults, customKeys),
		Flag:          buildBinding("flag", defaults, customKeys),
		SendText:      buildBinding("send_text", defaults, customKeys),
		StatusCycle:   buildBinding("cycle_status", defaults, customKeys),
		StatusSetForm: buildBinding("set_status", defaults, customKeys),
	}
}

// newSessionActionsKeys creates session action key bindings
func newSessionActionsKeys(defaults map[string][]string, customKeys config.KeyBindingsConfig) SessionActionsKeys {
	return SessionActionsKeys{
		Detach:      buildBinding("detach", defaults, customKeys),
		Open:        buildBinding("open", defaults, customKeys),
		OpenEditor:  buildBinding("open_editor", defaults, customKeys),
		OpenShell:   buildBinding("open_shell", defaults, customKeys),
		OptionsMenu: buildBinding("options_menu", defaults, customKeys),
		QuickOpen:   buildBinding("quick_open", defaults, customKeys),
	}
}
