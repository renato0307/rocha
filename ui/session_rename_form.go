package ui

import (
	"context"
	"fmt"
	"rocha/logging"
	"rocha/storage"
	"rocha/tmux"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// sanitizeTmuxName converts a display name to a tmux-compatible session name
// - Alphanumeric, underscores, hyphens, and periods are kept
// - Spaces and parentheses become underscores (multiple consecutive become single underscore)
// - Special characters like []{}:;,!@#$%^&*+=|\/'"<>? are removed
func sanitizeTmuxName(displayName string) string {
	var result strings.Builder
	lastWasUnderscore := false

	for _, r := range displayName {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '.' {
			// Keep alphanumeric, hyphens, and periods
			result.WriteRune(r)
			lastWasUnderscore = false
		} else if r == '_' {
			// Keep explicit underscores
			result.WriteRune('_')
			lastWasUnderscore = true
		} else if unicode.IsSpace(r) || r == '(' || r == ')' {
			// Replace spaces and parentheses with underscore (but avoid consecutive underscores)
			if !lastWasUnderscore && result.Len() > 0 {
				result.WriteRune('_')
				lastWasUnderscore = true
			}
		}
		// All other special characters are removed (no continue needed, just don't add them)
	}

	// Trim trailing underscore if any
	str := result.String()
	return strings.TrimRight(str, "_")
}

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
	sessionManager     tmux.SessionManager
	sessionState       *storage.SessionState
	store              *storage.Store
}

// NewSessionRenameForm creates a new session rename form
func NewSessionRenameForm(sessionManager tmux.SessionManager, store *storage.Store, sessionState *storage.SessionState, oldTmuxName, currentDisplayName string) *SessionRenameForm {
	sf := &SessionRenameForm{
		currentDisplayName: currentDisplayName,
		oldTmuxName:        oldTmuxName,
		result: SessionRenameFormResult{
			OldTmuxName: oldTmuxName,
		},
		sessionManager: sessionManager,
		sessionState:   sessionState,
		store:          store,
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
					tmuxName := sanitizeTmuxName(s)
					if sessionManager.Exists(tmuxName) && tmuxName != oldTmuxName {
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
	newTmuxName := sanitizeTmuxName(newDisplayName)
	sf.result.NewTmuxName = newTmuxName

	logging.Logger.Info("Renaming session",
		"old_name", sf.oldTmuxName,
		"new_tmux_name", newTmuxName,
		"new_display_name", newDisplayName)

	// Rename tmux session first
	if err := sf.sessionManager.Rename(sf.oldTmuxName, newTmuxName); err != nil {
		return fmt.Errorf("failed to rename tmux session: %w", err)
	}

	// Update state (re-key the map)
	if sessionInfo, exists := sf.sessionState.Sessions[sf.oldTmuxName]; exists {
		// Update the session info with new names
		sessionInfo.Name = newTmuxName
		sessionInfo.DisplayName = newDisplayName
		sessionInfo.LastUpdated = time.Now().UTC()

		// Re-key the map
		delete(sf.sessionState.Sessions, sf.oldTmuxName)
		sf.sessionState.Sessions[newTmuxName] = sessionInfo

		// Save to database (OrderedSessionNames will be repopulated on next Load)
		if err := sf.store.Save(context.Background(), sf.sessionState); err != nil {
			// Try to rename back in tmux if state update fails
			sf.sessionManager.Rename(newTmuxName, sf.oldTmuxName)
			return fmt.Errorf("failed to update session state: %w", err)
		}
	} else {
		// Session not found in state
		sf.sessionManager.Rename(newTmuxName, sf.oldTmuxName)
		return fmt.Errorf("session %s not found in state", sf.oldTmuxName)
	}

	logging.Logger.Info("Session renamed successfully", "new_name", newTmuxName)
	return nil
}
