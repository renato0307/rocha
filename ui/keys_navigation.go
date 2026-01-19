package ui

import "github.com/charmbracelet/bubbles/key"

// NavigationKeys defines key bindings for navigating the session list
type NavigationKeys struct {
	ClearFilter key.Binding
	Filter      key.Binding
	MoveDown    key.Binding
	MoveUp      key.Binding
}

// newNavigationKeys creates navigation key bindings
func newNavigationKeys() NavigationKeys {
	return NavigationKeys{
		ClearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter (press twice within 500ms)"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		MoveDown: key.NewBinding(
			key.WithKeys("J"),
			key.WithHelp("shift+↓/j", "move session down"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("shift+↑/k", "move session up"),
		),
	}
}
