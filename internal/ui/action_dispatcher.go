package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/ports"
)

// ActionDispatcher maps domain actions to UI messages.
// This keeps the command palette decoupled from specific message types.
type ActionDispatcher struct {
	session *ports.TmuxSession
}

// NewActionDispatcher creates a new action dispatcher.
// session can be nil if no session is selected.
func NewActionDispatcher(session *ports.TmuxSession) *ActionDispatcher {
	return &ActionDispatcher{session: session}
}

// Dispatch returns the appropriate tea.Msg for the given action.
// Returns nil if the action cannot be dispatched (e.g., requires session but none selected).
func (d *ActionDispatcher) Dispatch(action domain.Action) tea.Msg {
	// Safety check: action requires session but none selected
	if action.RequiresSession && d.session == nil {
		return nil
	}

	switch action.Name {
	case "archive":
		return ArchiveSessionMsg{SessionName: d.session.Name}
	case "comment":
		return CommentSessionMsg{SessionName: d.session.Name}
	case "cycle_status":
		return CycleStatusMsg{SessionName: d.session.Name}
	case "flag":
		return ToggleFlagSessionMsg{SessionName: d.session.Name}
	case "help":
		return ShowHelpMsg{}
	case "kill":
		return KillSessionMsg{SessionName: d.session.Name}
	case "new_from_repo":
		return NewSessionFromTemplateMsg{TemplateSessionName: d.session.Name}
	case "new_session":
		return NewSessionMsg{}
	case "open":
		return AttachSessionMsg{Session: d.session}
	case "open_editor":
		return OpenEditorSessionMsg{SessionName: d.session.Name}
	case "open_shell":
		return AttachShellSessionMsg{Session: d.session}
	case "quit":
		return QuitMsg{}
	case "rename":
		return RenameSessionMsg{SessionName: d.session.Name}
	case "send_text":
		return SendTextSessionMsg{SessionName: d.session.Name}
	case "set_status":
		return SetStatusSessionMsg{SessionName: d.session.Name}
	case "timestamps":
		return ToggleTimestampsMsg{}
	case "token_chart":
		return ToggleTokenChartMsg{}
	default:
		return nil
	}
}
