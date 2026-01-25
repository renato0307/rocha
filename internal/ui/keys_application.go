package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"github.com/renato0307/rocha/internal/config"
)

// ApplicationKeys defines key bindings for application-level actions
type ApplicationKeys struct {
	ForceQuit  KeyWithTip
	Help       KeyWithTip
	Quit       KeyWithTip
	Timestamps KeyWithTip
	TokenChart KeyWithTip
}

// newApplicationKeys creates application key bindings
func newApplicationKeys(defaults map[string][]string, customKeys config.KeyBindingsConfig) ApplicationKeys {
	return ApplicationKeys{
		ForceQuit:  buildBinding("force_quit", defaults, customKeys),
		Help:       buildBinding("help", defaults, customKeys),
		Quit:       buildBinding("quit", defaults, customKeys),
		Timestamps: buildBinding("timestamps", defaults, customKeys),
		TokenChart: buildBinding("token_chart", defaults, customKeys),
	}
}

// buildBinding creates a KeyWithTip from the key definition, using custom keys if provided.
func buildBinding(name string, defaults map[string][]string, customKeys config.KeyBindingsConfig) KeyWithTip {
	def := GetKeyDefinition(name)
	if def == nil {
		panic("unknown key definition: " + name)
	}

	keys := defaults[name]
	if custom, ok := customKeys[name]; ok && len(custom) > 0 {
		keys = custom
	}
	helpKeys := strings.Join(keys, "/")

	result := KeyWithTip{
		Binding: key.NewBinding(
			key.WithKeys(keys...),
			key.WithHelp(helpKeys, def.Help),
		),
	}

	if def.TipFormat != "" && len(keys) > 0 {
		result.Tip = newTip(def.TipFormat, keys[0])
	}

	return result
}
