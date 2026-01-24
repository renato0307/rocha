package ui

import "github.com/charmbracelet/bubbles/key"

// NavigationKeys defines key bindings for navigating the session list
type NavigationKeys struct {
	ClearFilter KeyWithTip
	Down        KeyWithTip
	Filter      KeyWithTip
	MoveDown    KeyWithTip
	MoveUp      KeyWithTip
	Up          KeyWithTip
}

// newNavigationKeys creates navigation key bindings
func newNavigationKeys() NavigationKeys {
	return NavigationKeys{
		ClearFilter: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "clear filter (press twice within 500ms)"),
			),
			Tip: newTip("press %s twice to clear the filter", "esc"),
		},
		Down: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓/j", "down"),
			),
		},
		Filter: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("/"),
				key.WithHelp("/", "filter"),
			),
			Tip: newTip("press %s to filter sessions by name or branch", "/"),
		},
		MoveDown: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("J", "shift+down"),
				key.WithHelp("shift+↓/j", "move session down"),
			),
		},
		MoveUp: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("K", "shift+up"),
				key.WithHelp("shift+↑/k", "move session up"),
			),
			Tip: newTip("press %s to reorder sessions in the list", "shift+↑"),
		},
		Up: KeyWithTip{
			Binding: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("↑/k", "up"),
			),
		},
	}
}
