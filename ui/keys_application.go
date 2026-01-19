package ui

import "github.com/charmbracelet/bubbles/key"

// ApplicationKeys defines key bindings for application-level actions
type ApplicationKeys struct {
	ForceQuit key.Binding
	Help      key.Binding
	Quit      key.Binding
	Timestamps key.Binding
}

// newApplicationKeys creates application key bindings
func newApplicationKeys() ApplicationKeys {
	return ApplicationKeys{
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("h", "?"),
			key.WithHelp("h/?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Timestamps: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle timestamps"),
		),
	}
}
