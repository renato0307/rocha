package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/ports"
)

// ActionDispatcher maps key definitions to UI messages.
// This keeps the command palette decoupled from specific message types.
type ActionDispatcher struct {
	session *ports.TmuxSession
}

// NewActionDispatcher creates a new action dispatcher.
// session can be nil if no session is selected.
func NewActionDispatcher(session *ports.TmuxSession) *ActionDispatcher {
	return &ActionDispatcher{session: session}
}

// Dispatch returns the appropriate tea.Msg for the given key definition.
// Returns nil if the action cannot be dispatched.
func (d *ActionDispatcher) Dispatch(def KeyDefinition) tea.Msg {
	if def.Msg == nil {
		return nil
	}

	// If message needs session context, call WithSession
	if sessionMsg, ok := def.Msg.(SessionAwareMsg); ok {
		if d.session == nil {
			return nil
		}
		return sessionMsg.WithSession(d.session)
	}

	// Otherwise return the prototype as-is
	return def.Msg
}
