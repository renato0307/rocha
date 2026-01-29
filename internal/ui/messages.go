package ui

import "github.com/renato0307/rocha/internal/ports"

// Action messages - these replace the mutable result fields in SessionList.
// Each message type represents a specific action the user wants to perform.
// Model handles these messages in updateList() and takes appropriate action.

// Phase 1: Foundation messages

// AttachSessionMsg requests attaching to a Claude session
type AttachSessionMsg struct {
	Session *ports.TmuxSession
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

// NewSessionFromTemplateMsg requests creating a new session from a template
type NewSessionFromTemplateMsg struct {
	TemplateSessionName string
}

// NewSessionMsg requests showing the new session dialog
type NewSessionMsg struct {
	DefaultRepoSource string
}

// OpenEditorSessionMsg requests opening the editor for a session's worktree
type OpenEditorSessionMsg struct {
	SessionName string
}

// RenameSessionMsg requests showing the rename dialog for a session
type RenameSessionMsg struct {
	SessionName string
}

// SendTextSessionMsg requests showing the send text dialog for a session
type SendTextSessionMsg struct {
	SessionName string
}

// SetStatusSessionMsg requests showing the status dialog for a session
type SetStatusSessionMsg struct {
	SessionName string
}

// Phase 3: Complex action messages

// ArchiveSessionMsg requests archiving a session
type ArchiveSessionMsg struct {
	SessionName string
}

// AttachShellSessionMsg requests attaching to a shell session
type AttachShellSessionMsg struct {
	Session *ports.TmuxSession
}

// KillSessionMsg requests killing a session
type KillSessionMsg struct {
	SessionName string
}

// TestErrorMsg requests generating a test error (hidden debug feature, triggered by alt+e)
type TestErrorMsg struct{}

// ToggleFlagSessionMsg requests toggling the flag on a session
type ToggleFlagSessionMsg struct {
	SessionName string
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
