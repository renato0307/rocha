package ui

import (
	"github.com/renato0307/rocha/internal/config"
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
func newNavigationKeys(defaults map[string][]string, customKeys config.KeyBindingsConfig) NavigationKeys {
	return NavigationKeys{
		ClearFilter: buildBinding("clear_filter", defaults, customKeys),
		Down:        buildBinding("down", defaults, customKeys),
		Filter:      buildBinding("filter", defaults, customKeys),
		MoveDown:    buildBinding("move_down", defaults, customKeys),
		MoveUp:      buildBinding("move_up", defaults, customKeys),
		Up:          buildBinding("up", defaults, customKeys),
	}
}
