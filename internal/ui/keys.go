package ui

import (
	"github.com/charmbracelet/bubbles/key"

	"rocha/internal/config"
)

// KeyMap contains all keyboard shortcuts organized by context
type KeyMap struct {
	Application       ApplicationKeys
	Navigation        NavigationKeys
	SessionActions    SessionActionsKeys
	SessionManagement SessionManagementKeys
	SessionMetadata   SessionMetadataKeys
}

// NewKeyMap creates a new KeyMap with all key bindings initialized
// Pass nil for keysConfig to use default bindings
func NewKeyMap(keysConfig *config.KeyBindingsConfig) KeyMap {
	return KeyMap{
		Application:       newApplicationKeys(keysConfig),
		Navigation:        newNavigationKeys(keysConfig),
		SessionActions:    newSessionActionsKeys(keysConfig),
		SessionManagement: newSessionManagementKeys(keysConfig),
		SessionMetadata:   newSessionMetadataKeys(keysConfig),
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
