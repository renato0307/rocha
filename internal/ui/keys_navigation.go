package ui

import (
	"rocha/internal/config"
)

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
func newNavigationKeys(keysConfig *config.KeyBindingsConfig) NavigationKeys {
	defaults := config.GetDefaultKeyBindings()

	return NavigationKeys{
		ClearFilter: buildBinding(defaults["clear_filter"], keysConfig.GetBindingByName("clear_filter"), "clear filter (press twice within 500ms)", "press %s twice to clear the filter"),
		Down:        buildBinding(defaults["down"], keysConfig.GetBindingByName("down"), "down", ""),
		Filter:      buildBinding(defaults["filter"], keysConfig.GetBindingByName("filter"), "filter", "press %s to filter sessions by name or branch"),
		MoveDown:    buildBinding(defaults["move_down"], keysConfig.GetBindingByName("move_down"), "move session down", ""),
		MoveUp:      buildBinding(defaults["move_up"], keysConfig.GetBindingByName("move_up"), "move session up", "press %s to reorder sessions in the list"),
		Up:          buildBinding(defaults["up"], keysConfig.GetBindingByName("up"), "up", ""),
	}
}
