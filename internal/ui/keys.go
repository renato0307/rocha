package ui

import (
	"github.com/charmbracelet/bubbles/key"

	"github.com/renato0307/rocha/internal/config"
)

// KeyMap contains all keyboard shortcuts organized by context
type KeyMap struct {
	Application       ApplicationKeys
	Navigation        NavigationKeys
	SessionActions    SessionActionsKeys
	SessionManagement SessionManagementKeys
	SessionMetadata   SessionMetadataKeys
}

// NewKeyMap creates a new KeyMap with all key bindings initialized.
// Pass nil for customKeys to use default bindings.
func NewKeyMap(customKeys config.KeyBindingsConfig) KeyMap {
	defaults := GetDefaultKeyBindings()
	return KeyMap{
		Application:       newApplicationKeys(defaults, customKeys),
		Navigation:        newNavigationKeys(defaults, customKeys),
		SessionActions:    newSessionActionsKeys(defaults, customKeys),
		SessionManagement: newSessionManagementKeys(defaults, customKeys),
		SessionMetadata:   newSessionMetadataKeys(defaults, customKeys),
	}
}

// ShortHelp returns a curated list of key bindings for the bottom bar
// Note: ctrl+q is intentionally excluded as it only works when attached to a session
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.SessionActions.Open.Binding,
		k.SessionManagement.New.Binding,
		k.SessionManagement.Rename.Binding,
		k.SessionManagement.Archive.Binding,
		k.SessionManagement.Kill.Binding,
		k.SessionMetadata.Comment.Binding,
		k.SessionMetadata.Flag.Binding,
		k.Navigation.Filter.Binding,
		k.Application.Help.Binding,
		k.Application.Quit.Binding,
	}
}
