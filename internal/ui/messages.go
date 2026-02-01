package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/ports"
)

// SessionAwareMsg is implemented by messages that need session context.
// Messages without session requirements don't need to implement this.
type SessionAwareMsg interface {
	WithSession(session *ports.TmuxSession) tea.Msg
}

// Action messages - these replace the mutable result fields in SessionList.
// Each message type represents a specific action the user wants to perform.
// Model handles these messages in updateList() and takes appropriate action.

// Phase 1: Foundation messages

// AttachSessionMsg requests attaching to a Claude session
type AttachSessionMsg struct {
	Session *ports.TmuxSession
}

func (m AttachSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return AttachSessionMsg{Session: s}
}

// QuitMsg requests quitting the application
type QuitMsg struct{}

// ShowHelpMsg requests showing the help screen
type ShowHelpMsg struct{}

// Phase 2: Dialog action messages

// CommentSessionMsg requests showing the comment dialog for a session
type CommentSessionMsg struct {
	SessionName string
}

func (m CommentSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return CommentSessionMsg{SessionName: s.Name}
}

// NewSessionFromTemplateMsg requests creating a new session from a template
type NewSessionFromTemplateMsg struct {
	TemplateSessionName string
}

func (m NewSessionFromTemplateMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return NewSessionFromTemplateMsg{TemplateSessionName: s.Name}
}

// NewSessionMsg requests showing the new session dialog
type NewSessionMsg struct {
	DefaultRepoSource string
}

// OpenEditorSessionMsg requests opening the editor for a session's worktree
type OpenEditorSessionMsg struct {
	SessionName string
}

func (m OpenEditorSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return OpenEditorSessionMsg{SessionName: s.Name}
}

// RenameSessionMsg requests showing the rename dialog for a session
type RenameSessionMsg struct {
	SessionName string
}

func (m RenameSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return RenameSessionMsg{SessionName: s.Name}
}

// SendTextSessionMsg requests showing the send text dialog for a session
type SendTextSessionMsg struct {
	SessionName string
}

func (m SendTextSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return SendTextSessionMsg{SessionName: s.Name}
}

// SetStatusSessionMsg requests showing the status dialog for a session
type SetStatusSessionMsg struct {
	SessionName string
}

func (m SetStatusSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return SetStatusSessionMsg{SessionName: s.Name}
}

// Phase 3: Complex action messages

// ArchiveSessionMsg requests archiving a session
type ArchiveSessionMsg struct {
	SessionName string
}

func (m ArchiveSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return ArchiveSessionMsg{SessionName: s.Name}
}

// AttachShellSessionMsg requests attaching to a shell session
type AttachShellSessionMsg struct {
	Session *ports.TmuxSession
}

func (m AttachShellSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return AttachShellSessionMsg{Session: s}
}

// KillSessionMsg requests killing a session
type KillSessionMsg struct {
	SessionName string
}

func (m KillSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return KillSessionMsg{SessionName: s.Name}
}

// TestErrorMsg requests generating a test error (hidden debug feature, triggered by alt+e)
type TestErrorMsg struct{}

// ToggleFlagSessionMsg requests toggling the flag on a session
type ToggleFlagSessionMsg struct {
	SessionName string
}

func (m ToggleFlagSessionMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return ToggleFlagSessionMsg{SessionName: s.Name}
}

// Command palette messages

// ShowCommandPaletteMsg requests showing the command palette
type ShowCommandPaletteMsg struct{}

// ExecuteActionMsg requests executing an action from the command palette
type ExecuteActionMsg struct {
	ActionName string
}

// CloseCommandPaletteMsg requests closing the command palette
type CloseCommandPaletteMsg struct{}

// CycleStatusMsg requests cycling the status of a session
type CycleStatusMsg struct {
	SessionName string
}

func (m CycleStatusMsg) WithSession(s *ports.TmuxSession) tea.Msg {
	return CycleStatusMsg{SessionName: s.Name}
}

// ToggleTimestampsMsg requests toggling timestamp display
type ToggleTimestampsMsg struct{}

// ToggleTokenChartMsg requests toggling the token chart
type ToggleTokenChartMsg struct{}
