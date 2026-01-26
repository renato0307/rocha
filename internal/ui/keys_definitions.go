package ui

import (
	"sort"
	"sync"
)

// KeyDefinition defines the metadata for a configurable key binding.
// All key bindings are defined here as the single source of truth.
type KeyDefinition struct {
	Defaults  []string
	Help      string
	Name      string
	TipFormat string
}

// AllKeyDefinitions contains all configurable key bindings.
// This is the single source of truth for key names, defaults, help text, and tips.
var AllKeyDefinitions = []KeyDefinition{
	// Application keys
	{Name: "force_quit", Defaults: []string{"ctrl+c"}, Help: "quit", TipFormat: ""},
	{Name: "help", Defaults: []string{"h", "?"}, Help: "help", TipFormat: "press %s to see all shortcuts"},
	{Name: "quit", Defaults: []string{"q"}, Help: "quit", TipFormat: ""},
	{Name: "timestamps", Defaults: []string{"t"}, Help: "toggle timestamps", TipFormat: "press %s to toggle timestamp display"},
	{Name: "token_chart", Defaults: []string{"T"}, Help: "toggle token chart", TipFormat: "press %s to toggle token usage chart"},

	// Navigation keys
	{Name: "clear_filter", Defaults: []string{"esc"}, Help: "clear filter (press twice within 500ms)", TipFormat: "press %s twice to clear the filter"},
	{Name: "down", Defaults: []string{"down", "j"}, Help: "down", TipFormat: ""},
	{Name: "filter", Defaults: []string{"/"}, Help: "filter", TipFormat: "press %s to filter sessions by name or branch"},
	{Name: "move_down", Defaults: []string{"J", "shift+down"}, Help: "move session down", TipFormat: ""},
	{Name: "move_up", Defaults: []string{"K", "shift+up"}, Help: "move session up", TipFormat: "press %s to reorder sessions in the list"},
	{Name: "up", Defaults: []string{"up", "k"}, Help: "up", TipFormat: ""},

	// Session management keys
	{Name: "archive", Defaults: []string{"a"}, Help: "archive", TipFormat: "press %s to archive a session (hidden from list)"},
	{Name: "kill", Defaults: []string{"x"}, Help: "kill", TipFormat: "press %s to kill a session and optionally remove its worktree"},
	{Name: "new", Defaults: []string{"n"}, Help: "new", TipFormat: "press %s to create a new session"},
	{Name: "new_from_repo", Defaults: []string{"N"}, Help: "new from same repo", TipFormat: "press %s to create a new session based on the selected session"},
	{Name: "rename", Defaults: []string{"r"}, Help: "rename", TipFormat: "press %s to rename a session"},

	// Session metadata keys
	{Name: "comment", Defaults: []string{"c"}, Help: "comment (⌨)", TipFormat: "press %s to add a comment to a session"},
	{Name: "cycle_status", Defaults: []string{"s"}, Help: "cycle status", TipFormat: "press %s to cycle through implementation statuses"},
	{Name: "flag", Defaults: []string{"f"}, Help: "flag (⚑)", TipFormat: "press %s to flag a session for attention"},
	{Name: "send_text", Defaults: []string{"p"}, Help: "send text (prompt)", TipFormat: "press %s to send text to a session (experimental)"},
	{Name: "set_status", Defaults: []string{"S"}, Help: "set status", TipFormat: "press %s to pick a specific status"},

	// Session action keys
	{Name: "detach", Defaults: []string{"ctrl+q"}, Help: "detach from session (return to list)", TipFormat: "press %s inside a session to return to the list"},
	{Name: "open", Defaults: []string{"enter"}, Help: "open", TipFormat: ""},
	{Name: "open_editor", Defaults: []string{"o"}, Help: "editor", TipFormat: "press %s to open the session's folder in your editor"},
	{Name: "open_shell", Defaults: []string{"ctrl+s"}, Help: "shell (>_)", TipFormat: "press %s to open a shell session alongside claude"},
	{Name: "command_palette", Defaults: []string{"O"}, Help: "command palette", TipFormat: "press %s to open the command palette"},
	{Name: "quick_open", Defaults: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}, Help: "quick open (0=10th)", TipFormat: "press %s to quickly open sessions by their number"},
}

var (
	defaultBindingsCache map[string][]string
	defaultBindingsOnce  sync.Once

	keyDefinitionsMap     map[string]KeyDefinition
	keyDefinitionsMapOnce sync.Once

	validKeyNames     []string
	validKeyNamesOnce sync.Once
)

// GetDefaultKeyBindings returns the default key bindings as a map.
// The result is cached after the first call.
func GetDefaultKeyBindings() map[string][]string {
	defaultBindingsOnce.Do(func() {
		defaultBindingsCache = make(map[string][]string, len(AllKeyDefinitions))
		for _, def := range AllKeyDefinitions {
			defaultBindingsCache[def.Name] = def.Defaults
		}
	})
	return defaultBindingsCache
}

// GetKeyDefinition returns the definition for a key by name.
// Returns nil if not found.
func GetKeyDefinition(name string) *KeyDefinition {
	keyDefinitionsMapOnce.Do(func() {
		keyDefinitionsMap = make(map[string]KeyDefinition, len(AllKeyDefinitions))
		for _, def := range AllKeyDefinitions {
			keyDefinitionsMap[def.Name] = def
		}
	})
	if def, ok := keyDefinitionsMap[name]; ok {
		return &def
	}
	return nil
}

// GetValidKeyNames returns all valid key binding names in sorted order.
// The result is cached after the first call.
func GetValidKeyNames() []string {
	validKeyNamesOnce.Do(func() {
		validKeyNames = make([]string, len(AllKeyDefinitions))
		for i, def := range AllKeyDefinitions {
			validKeyNames[i] = def.Name
		}
		sort.Strings(validKeyNames)
	})
	return validKeyNames
}

// IsValidKeyName checks if a name is a valid key binding name.
func IsValidKeyName(name string) bool {
	return GetKeyDefinition(name) != nil
}
