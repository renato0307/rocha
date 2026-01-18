package ui

import (
	"context"
	"fmt"
	"rocha/logging"
	"rocha/storage"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// SessionStatusFormResult contains the result of the status update operation
type SessionStatusFormResult struct {
	Cancelled bool
	Error     error
	SessionName string // Session being updated
	Status      *string // New status (nil = cleared)
}

// SessionStatusForm is a Bubble Tea component for setting session status
type SessionStatusForm struct {
	Completed    bool
	cancelled    bool
	form         *huh.Form
	result       SessionStatusFormResult
	selectedItem string // Holds the selected option (status name or "<clear>")
	sessionName  string
	statusConfig *StatusConfig
	store        *storage.Store
}

// NewSessionStatusForm creates a new session status form
func NewSessionStatusForm(store *storage.Store, sessionName string, currentStatus *string, statusConfig *StatusConfig) *SessionStatusForm {
	sf := &SessionStatusForm{
		result: SessionStatusFormResult{
			SessionName: sessionName,
		},
		sessionName:  sessionName,
		statusConfig: statusConfig,
		store:        store,
	}

	// Build status options
	options := make([]huh.Option[string], 0, len(statusConfig.Statuses)+1)

	// Add clear option at the top
	options = append(options, huh.NewOption("<clear>", "<clear>"))

	// Add all configured statuses (no icons, colors will be shown in the list)
	for _, status := range statusConfig.Statuses {
		options = append(options, huh.NewOption(status, status))
	}

	// Set initial selected value
	if currentStatus != nil {
		sf.selectedItem = *currentStatus
	} else {
		sf.selectedItem = "<clear>"
	}

	// Build form with select field
	sf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Set implementation status").
				Description(fmt.Sprintf("Session: %s", sessionName)).
				Options(options...).
				Value(&sf.selectedItem),
		),
	)

	return sf
}

func (sf *SessionStatusForm) Init() tea.Cmd {
	return sf.form.Init()
}

func (sf *SessionStatusForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// Update the status
		if err := sf.updateStatus(); err != nil {
			logging.Logger.Error("Failed to update status", "error", err)
			sf.result.Error = err
		}
		return sf, nil
	}

	return sf, cmd
}

func (sf *SessionStatusForm) View() string {
	if sf.form != nil {
		return sf.form.View()
	}
	return ""
}

// Result returns the form result
func (sf *SessionStatusForm) Result() SessionStatusFormResult {
	return sf.result
}

// updateStatus performs the actual status update
func (sf *SessionStatusForm) updateStatus() error {
	var newStatus *string

	// Check if user selected to clear the status
	if sf.selectedItem == "<clear>" {
		newStatus = nil
	} else {
		newStatus = &sf.selectedItem
	}

	sf.result.Status = newStatus

	logging.Logger.Info("Updating session status",
		"session", sf.sessionName,
		"status", sf.selectedItem)

	// Update in database
	if err := sf.store.UpdateSessionStatus(context.Background(), sf.sessionName, newStatus); err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	logging.Logger.Info("Session status updated successfully")
	return nil
}
