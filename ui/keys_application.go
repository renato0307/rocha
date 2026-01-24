package ui

import "github.com/charmbracelet/bubbles/key"

// ApplicationKeys defines key bindings for application-level actions
type ApplicationKeys struct {
	ForceQuit  KeyWithTip
	Help       KeyWithTip
	Quit       KeyWithTip
	Timestamps KeyWithTip
}

// newApplicationKeys creates application key bindings
func newApplicationKeys() ApplicationKeys {
	return ApplicationKeys{
		ForceQuit: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("ctrl+c"),
				key.WithHelp("ctrl+c", "quit"),
			),
		},
		Help: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("h", "?"),
				key.WithHelp("h/?", "help"),
			),
			Tip: newTip("press '?' to see all shortcuts"),
		},
		Quit: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("q"),
				key.WithHelp("q", "quit"),
			),
		},
		Timestamps: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("t"),
				key.WithHelp("t", "toggle timestamps"),
			),
			Tip: newTip("press 't' to toggle timestamp display"),
		},
	}
}
