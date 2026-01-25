package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/services"
)

// SessionCommentFormResult contains the result of the comment operation
type SessionCommentFormResult struct {
	Cancelled   bool
	Error       error
	NewComment  string
	SessionName string
}

// SessionCommentForm is a Bubble Tea component for editing session comments
type SessionCommentForm struct {
	Completed      bool
	cancelled      bool
	currentComment string
	form           *huh.Form
	result         SessionCommentFormResult
	sessionName    string
	sessionService *services.SessionService
}

// NewSessionCommentForm creates a new session comment form
func NewSessionCommentForm(sessionService *services.SessionService, sessionName, currentComment string) *SessionCommentForm {
	sf := &SessionCommentForm{
		currentComment: currentComment,
		sessionName:    sessionName,
		sessionService: sessionService,
		result: SessionCommentFormResult{
			SessionName: sessionName,
			NewComment:  currentComment, // Preload the current comment for editing
		},
	}

	// Build form with multi-line text input
	sf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Session comment").
				Description(fmt.Sprintf("Comment for: %s (empty to delete)", sessionName)).
				Value(&sf.result.NewComment).
				CharLimit(500),
		),
	)

	return sf
}

func (sf *SessionCommentForm) Init() tea.Cmd {
	return sf.form.Init()
}

func (sf *SessionCommentForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			sf.cancelled = true
			sf.result.Cancelled = true
			sf.Completed = true
			return sf, nil
		}
	}

	// Forward message to form
	form, cmd := sf.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		sf.form = f
	}

	// Check if form completed
	if sf.form.State == huh.StateCompleted {
		sf.Completed = true
		// Execute the comment update
		if err := sf.updateComment(); err != nil {
			logging.Logger.Error("Failed to update comment", "error", err)
			sf.result.Error = err
		}
		return sf, nil
	}

	return sf, cmd
}

func (sf *SessionCommentForm) View() string {
	if sf.form != nil {
		return sf.form.View()
	}
	return ""
}

// Result returns the form result
func (sf *SessionCommentForm) Result() SessionCommentFormResult {
	return sf.result
}

// updateComment performs the actual comment update operation
func (sf *SessionCommentForm) updateComment() error {
	// Trim whitespace - empty after trim means delete
	newComment := strings.TrimSpace(sf.result.NewComment)

	logging.Logger.Info("Updating session comment",
		"session_name", sf.sessionName,
		"comment_length", len(newComment))

	// Update via service (empty string = delete comment)
	if err := sf.sessionService.UpdateComment(context.Background(), sf.sessionName, newComment); err != nil {
		return fmt.Errorf("failed to update session comment: %w", err)
	}

	logging.Logger.Info("Session comment updated successfully", "session_name", sf.sessionName)
	return nil
}
