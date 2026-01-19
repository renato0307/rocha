package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap contains all keyboard shortcuts organized by context
type KeyMap struct {
	Application       ApplicationKeys
	Navigation        NavigationKeys
	SessionActions    SessionActionsKeys
	SessionManagement SessionManagementKeys
	SessionMetadata   SessionMetadataKeys
}

// NewKeyMap creates a new KeyMap with all key bindings initialized
func NewKeyMap() KeyMap {
	return KeyMap{
		Application:       newApplicationKeys(),
		Navigation:        newNavigationKeys(),
		SessionActions:    newSessionActionsKeys(),
		SessionManagement: newSessionManagementKeys(),
		SessionMetadata:   newSessionMetadataKeys(),
	}
}

// ShortHelp returns a curated list of key bindings for the bottom bar
// Note: ctrl+q is intentionally excluded as it only works when attached to a session
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.SessionActions.Open,
		k.SessionManagement.New,
		k.SessionManagement.Rename,
		k.SessionManagement.Archive,
		k.SessionManagement.Kill,
		k.SessionMetadata.Comment,
		k.SessionMetadata.Flag,
		k.Navigation.Filter,
		k.Application.Help,
		k.Application.Quit,
	}
}
