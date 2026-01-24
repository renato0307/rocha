package ui

import "github.com/charmbracelet/bubbles/key"

// tips is the private collection of all tips, populated by newTip()
var tips []string

// newTip registers a tip and returns it for inline assignment
func newTip(tip string) string {
	tips = append(tips, tip)
	return tip
}

// GetTips returns all registered tips
func GetTips() []string {
	return tips
}

// KeyWithTip wraps a key.Binding with an optional tip for rotating tips display.
type KeyWithTip struct {
	Binding key.Binding
	Tip     string
}

// TipsConfig holds configuration for the tips feature
type TipsConfig struct {
	DisplayDurationSeconds int
	Enabled                bool
	ShowIntervalSeconds    int
}
