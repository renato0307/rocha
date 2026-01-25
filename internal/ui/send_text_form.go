package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"rocha/internal/logging"
	"rocha/internal/services"
)

// SendTextFormResult contains the result of the send text operation
type SendTextFormResult struct {
	Cancelled   bool
	Error       error
	SessionName string
	Text        string
}

// SendTextForm is a Bubble Tea component for sending text to a tmux session
type SendTextForm struct {
	Completed    bool
	cancelled    bool
	form         *huh.Form
	result       SendTextFormResult
	sessionName  string
	shellService *services.ShellService
}

// NewSendTextForm creates a new send text form
func NewSendTextForm(shellService *services.ShellService, sessionName string) *SendTextForm {
	sf := &SendTextForm{
		sessionName:  sessionName,
		shellService: shellService,
		result: SendTextFormResult{
			SessionName: sessionName,
			Text:        "rebase with origin/main",
		},
	}

	// Build form with text input
	sf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Send text to Claude").
				Description(fmt.Sprintf("Text will be sent to session: %s", sessionName)).
				Value(&sf.result.Text).
				CharLimit(1000),
		),
	)

	return sf
}

func (sf *SendTextForm) Init() tea.Cmd {
	return sf.form.Init()
}

func (sf *SendTextForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// Execute the send text operation
		if err := sf.sendText(); err != nil {
			logging.Logger.Error("Failed to send text to tmux", "error", err)
			sf.result.Error = err
		}
		return sf, nil
	}

	return sf, cmd
}

func (sf *SendTextForm) View() string {
	if sf.form != nil {
		return sf.form.View()
	}
	return ""
}

// Result returns the form result
func (sf *SendTextForm) Result() SendTextFormResult {
	return sf.result
}

// sendText sends the text to the tmux session
func (sf *SendTextForm) sendText() error {
	if sf.result.Text == "" {
		logging.Logger.Info("No text to send, skipping")
		return nil
	}

	logging.Logger.Info("Sending text to tmux session",
		"session_name", sf.sessionName,
		"text_length", len(sf.result.Text))

	// Send the text to tmux first
	if err := sf.shellService.SendKeys(sf.sessionName, sf.result.Text); err != nil {
		return fmt.Errorf("failed to send text to tmux: %w", err)
	}

	// Then send Enter key separately to submit
	if err := sf.shellService.SendKeys(sf.sessionName, "C-m"); err != nil {
		return fmt.Errorf("failed to send enter key to tmux: %w", err)
	}

	logging.Logger.Info("Text sent and submitted successfully", "session_name", sf.sessionName)
	return nil
}
