package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"rocha/internal/domain"
	"rocha/internal/logging"
	"rocha/internal/services"
)


// SessionRenameFormResult contains the result of the rename operation
type SessionRenameFormResult struct {
	OldTmuxName    string // Original tmux session name
	NewTmuxName    string // New tmux session name (sanitized)
	NewDisplayName string // New display name (user input)
	Cancelled      bool
	Error          error
}

// SessionRenameForm is a Bubble Tea component for renaming sessions
type SessionRenameForm struct {
	Completed          bool
	cancelled          bool
	currentDisplayName string // Current display name for reference
	form               *huh.Form
	oldTmuxName        string // Immutable - the session we're renaming
	result             SessionRenameFormResult
	sessionService     *services.SessionService
	sessionState       *domain.SessionCollection
}

// NewSessionRenameForm creates a new session rename form
func NewSessionRenameForm(sessionService *services.SessionService, sessionState *domain.SessionCollection, oldTmuxName, currentDisplayName string) *SessionRenameForm {
	sf := &SessionRenameForm{
		currentDisplayName: currentDisplayName,
		oldTmuxName:        oldTmuxName,
		result: SessionRenameFormResult{
			OldTmuxName: oldTmuxName,
		},
		sessionService: sessionService,
		sessionState:   sessionState,
	}

	// Build form with single input field
	sf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("New session name").
				Description(fmt.Sprintf("Renaming: %s", currentDisplayName)).
				Value(&sf.result.NewDisplayName).
				Placeholder(currentDisplayName).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("session name required")
					}
					// Sanitize for tmux name check
					tmuxName := domain.SanitizeSessionName(s)
					if sessionService.SessionExists(tmuxName) && tmuxName != oldTmuxName {
						return fmt.Errorf("session %s already exists", tmuxName)
					}
					return nil
				}),
		),
	)

	return sf
}

func (sf *SessionRenameForm) Init() tea.Cmd {
	return sf.form.Init()
}

func (sf *SessionRenameForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// Execute the rename
		if err := sf.renameSession(); err != nil {
			logging.Logger.Error("Failed to rename session", "error", err)
			sf.result.Error = err
		}
		return sf, nil
	}

	return sf, cmd
}

func (sf *SessionRenameForm) View() string {
	if sf.form != nil {
		return sf.form.View()
	}
	return ""
}

// Result returns the form result
func (sf *SessionRenameForm) Result() SessionRenameFormResult {
	return sf.result
}

// renameSession performs the actual rename operation
func (sf *SessionRenameForm) renameSession() error {
	newDisplayName := sf.result.NewDisplayName

	// Sanitize for tmux (remove colons, replace spaces/special chars with underscores)
	newTmuxName := domain.SanitizeSessionName(newDisplayName)
	sf.result.NewTmuxName = newTmuxName

	logging.Logger.Info("Renaming session",
		"old_name", sf.oldTmuxName,
		"new_tmux_name", newTmuxName,
		"new_display_name", newDisplayName)

	// Rename in both tmux and database (preserves position)
	if err := sf.sessionService.RenameSession(context.Background(), sf.oldTmuxName, newTmuxName, newDisplayName); err != nil {
		return err
	}

	logging.Logger.Info("Session renamed successfully", "new_name", newTmuxName)
	return nil
}
