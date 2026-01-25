package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// clearErrorMsg is a message sent after the error clear delay to trigger error clearing.
type clearErrorMsg struct{}

// ErrorManager handles error display and auto-clearing functionality.
// Centralized error management component.
type ErrorManager struct {
	currentError    error
	errorClearDelay time.Duration
}

// NewErrorManager creates a new ErrorManager with the specified auto-clear delay.
func NewErrorManager(errorClearDelay time.Duration) *ErrorManager {
	return &ErrorManager{
		errorClearDelay: errorClearDelay,
	}
}

// SetError sets the current error to be displayed.
func (em *ErrorManager) SetError(err error) {
	em.currentError = err
}

// ClearError clears the current error.
func (em *ErrorManager) ClearError() {
	em.currentError = nil
}

// GetError returns the current error.
func (em *ErrorManager) GetError() error {
	return em.currentError
}

// HasError returns true if there is a current error.
func (em *ErrorManager) HasError() bool {
	return em.currentError != nil
}

// ClearAfterDelay returns a tea.Cmd that sends clearErrorMsg after the configured delay.
func (em *ErrorManager) ClearAfterDelay() tea.Cmd {
	return tea.Tick(em.errorClearDelay, func(time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}
