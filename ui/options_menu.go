package ui

import (
	"rocha/logging"
	"rocha/tmux"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// OptionsMenuResult contains the result of the options menu selection
type OptionsMenuResult struct {
	ActionID  string        // Selected action identifier
	Cancelled bool          // User pressed ESC/Ctrl+C
	Session   *tmux.Session // Session the action applies to
}

// OptionsMenu is a Bubble Tea component for displaying context-sensitive session options
type OptionsMenu struct {
	Completed    bool
	form         *huh.Form
	result       OptionsMenuResult
	selectedItem string // Holds the selected option
	session      *tmux.Session
}

// NewOptionsMenu creates a new options menu for a session
func NewOptionsMenu(session *tmux.Session) *OptionsMenu {
	om := &OptionsMenu{
		result: OptionsMenuResult{
			Session: session,
		},
		session: session,
	}

	// Build menu options
	options := []huh.Option[string]{
		huh.NewOption("Rebase branch (coming soon)", "rebase"),
		huh.NewOption("Open PR link (coming soon)", "open-pr"),
		huh.NewOption("Copy session info (coming soon)", "copy-info"),
	}

	// Set initial selected value
	om.selectedItem = "rebase"

	// Build form with select field
	om.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Session options").
				Description(session.Name).
				Options(options...).
				Value(&om.selectedItem),
		),
	)

	return om
}

func (om *OptionsMenu) Init() tea.Cmd {
	return om.form.Init()
}

func (om *OptionsMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			om.result.Cancelled = true
			om.Completed = true
			return om, nil
		}
	}

	// Forward message to form
	form, cmd := om.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		om.form = f
	}

	// Check if form completed
	if om.form.State == huh.StateCompleted {
		om.Completed = true
		om.result.ActionID = om.selectedItem

		logging.Logger.Info("Options menu selection",
			"session", om.session.Name,
			"action", om.selectedItem)

		return om, nil
	}

	return om, cmd
}

func (om *OptionsMenu) View() string {
	if om.form != nil {
		return om.form.View()
	}
	return ""
}

// Result returns the menu result
func (om *OptionsMenu) Result() OptionsMenuResult {
	return om.result
}
