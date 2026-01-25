package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"rocha/internal/config"
)

// ApplicationKeys defines key bindings for application-level actions
type ApplicationKeys struct {
	ForceQuit  KeyWithTip
	Help       KeyWithTip
	Quit       KeyWithTip
	Timestamps KeyWithTip
}

// newApplicationKeys creates application key bindings
func newApplicationKeys(keysConfig *config.KeyBindingsConfig) ApplicationKeys {
	defaults := config.GetDefaultKeyBindings()

	return ApplicationKeys{
		ForceQuit: buildBinding(defaults["force_quit"], keysConfig.GetBindingByName("force_quit"), "quit", ""),
		Help:      buildBinding(defaults["help"], keysConfig.GetBindingByName("help"), "help", "press %s to see all shortcuts"),
		Quit:      buildBinding(defaults["quit"], keysConfig.GetBindingByName("quit"), "quit", ""),
		Timestamps: buildBinding(defaults["timestamps"], keysConfig.GetBindingByName("timestamps"), "toggle timestamps", "press %s to toggle timestamp display"),
	}
}

// buildBinding creates a KeyWithTip using custom keys if provided, otherwise defaults.
// If tipFormat is provided, it uses the first key to format the tip text.
func buildBinding(defaultKeys []string, customKeys config.KeyBindingValue, helpDesc string, tipFormat string) KeyWithTip {
	keys := defaultKeys
	if len(customKeys) > 0 {
		keys = customKeys
	}
	helpKeys := strings.Join(keys, "/")

	result := KeyWithTip{
		Binding: key.NewBinding(
			key.WithKeys(keys...),
			key.WithHelp(helpKeys, helpDesc),
		),
	}

	if tipFormat != "" && len(keys) > 0 {
		result.Tip = newTip(tipFormat, keys[0])
	}

	return result
}
