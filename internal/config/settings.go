package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ValidKeyBindingNames contains all valid key binding names that can be customized.
// These names correspond to the JSON field names in KeyBindingsConfig and are used
// for validation in CLI commands. Must be kept in alphabetical order.
var ValidKeyBindingNames = []string{
	"archive", "clear_filter", "comment", "cycle_status", "detach", "down",
	"filter", "flag", "force_quit", "help", "kill", "move_down", "move_up",
	"new", "new_from_repo", "open", "open_editor", "open_shell", "quick_open",
	"quit", "rename", "send_text", "set_status", "timestamps", "up",
}

// KeyBindingValue supports "a" or ["up", "k"] in JSON
type KeyBindingValue []string

// UnmarshalJSON implements custom unmarshaling for KeyBindingValue
func (kv *KeyBindingValue) UnmarshalJSON(data []byte) error {
	// Try array format first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*kv = arr
		return nil
	}

	// Fall back to single string
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if str != "" {
		*kv = []string{str}
	}
	return nil
}

// MarshalJSON implements custom marshaling for KeyBindingValue
func (kv KeyBindingValue) MarshalJSON() ([]byte, error) {
	if len(kv) == 1 {
		return json.Marshal(kv[0])
	}
	return json.Marshal([]string(kv))
}

// KeyBindingsConfig holds custom key binding overrides
type KeyBindingsConfig struct {
	Archive     KeyBindingValue `json:"archive,omitempty"`
	ClearFilter KeyBindingValue `json:"clear_filter,omitempty"`
	Comment     KeyBindingValue `json:"comment,omitempty"`
	CycleStatus KeyBindingValue `json:"cycle_status,omitempty"`
	Detach      KeyBindingValue `json:"detach,omitempty"`
	Down        KeyBindingValue `json:"down,omitempty"`
	Filter      KeyBindingValue `json:"filter,omitempty"`
	Flag        KeyBindingValue `json:"flag,omitempty"`
	ForceQuit   KeyBindingValue `json:"force_quit,omitempty"`
	Help        KeyBindingValue `json:"help,omitempty"`
	Kill        KeyBindingValue `json:"kill,omitempty"`
	MoveDown    KeyBindingValue `json:"move_down,omitempty"`
	MoveUp      KeyBindingValue `json:"move_up,omitempty"`
	New         KeyBindingValue `json:"new,omitempty"`
	NewFromRepo KeyBindingValue `json:"new_from_repo,omitempty"`
	Open        KeyBindingValue `json:"open,omitempty"`
	OpenEditor  KeyBindingValue `json:"open_editor,omitempty"`
	OpenShell   KeyBindingValue `json:"open_shell,omitempty"`
	QuickOpen   KeyBindingValue `json:"quick_open,omitempty"`
	Quit        KeyBindingValue `json:"quit,omitempty"`
	Rename      KeyBindingValue `json:"rename,omitempty"`
	SendText    KeyBindingValue `json:"send_text,omitempty"`
	SetStatus   KeyBindingValue `json:"set_status,omitempty"`
	Timestamps  KeyBindingValue `json:"timestamps,omitempty"`
	Up          KeyBindingValue `json:"up,omitempty"`
}

// GetBindingByName returns the key binding value for a given name
func (k *KeyBindingsConfig) GetBindingByName(name string) KeyBindingValue {
	if k == nil {
		return nil
	}
	switch name {
	case "archive":
		return k.Archive
	case "clear_filter":
		return k.ClearFilter
	case "comment":
		return k.Comment
	case "cycle_status":
		return k.CycleStatus
	case "detach":
		return k.Detach
	case "down":
		return k.Down
	case "filter":
		return k.Filter
	case "flag":
		return k.Flag
	case "force_quit":
		return k.ForceQuit
	case "help":
		return k.Help
	case "kill":
		return k.Kill
	case "move_down":
		return k.MoveDown
	case "move_up":
		return k.MoveUp
	case "new":
		return k.New
	case "new_from_repo":
		return k.NewFromRepo
	case "open":
		return k.Open
	case "open_editor":
		return k.OpenEditor
	case "open_shell":
		return k.OpenShell
	case "quick_open":
		return k.QuickOpen
	case "quit":
		return k.Quit
	case "rename":
		return k.Rename
	case "send_text":
		return k.SendText
	case "set_status":
		return k.SetStatus
	case "timestamps":
		return k.Timestamps
	case "up":
		return k.Up
	default:
		return nil
	}
}

// SetBindingByName sets a key binding value by name
func (k *KeyBindingsConfig) SetBindingByName(name string, value KeyBindingValue) bool {
	if k == nil {
		return false
	}
	switch name {
	case "archive":
		k.Archive = value
	case "clear_filter":
		k.ClearFilter = value
	case "comment":
		k.Comment = value
	case "cycle_status":
		k.CycleStatus = value
	case "detach":
		k.Detach = value
	case "down":
		k.Down = value
	case "filter":
		k.Filter = value
	case "flag":
		k.Flag = value
	case "force_quit":
		k.ForceQuit = value
	case "help":
		k.Help = value
	case "kill":
		k.Kill = value
	case "move_down":
		k.MoveDown = value
	case "move_up":
		k.MoveUp = value
	case "new":
		k.New = value
	case "new_from_repo":
		k.NewFromRepo = value
	case "open":
		k.Open = value
	case "open_editor":
		k.OpenEditor = value
	case "open_shell":
		k.OpenShell = value
	case "quick_open":
		k.QuickOpen = value
	case "quit":
		k.Quit = value
	case "rename":
		k.Rename = value
	case "send_text":
		k.SendText = value
	case "set_status":
		k.SetStatus = value
	case "timestamps":
		k.Timestamps = value
	case "up":
		k.Up = value
	default:
		return false
	}
	return true
}

// Validate checks for configuration errors in key bindings
func (k *KeyBindingsConfig) Validate() error {
	if k == nil {
		return nil
	}

	// Track all keys to detect duplicates
	keyToAction := make(map[string]string)

	// Validate each configured binding
	for _, name := range ValidKeyBindingNames {
		keys := k.GetBindingByName(name)
		if err := validateBinding(name, keys, keyToAction); err != nil {
			return err
		}
	}

	return nil
}

// validateBinding checks a single binding for errors and tracks keys for duplicate detection
func validateBinding(action string, keys KeyBindingValue, keyToAction map[string]string) error {
	if len(keys) == 0 {
		return nil // Not configured, will use default
	}

	for _, k := range keys {
		if k == "" {
			return fmt.Errorf("key binding for '%s' contains empty value", action)
		}
		if existing, found := keyToAction[k]; found {
			return fmt.Errorf("key '%s' is assigned to both '%s' and '%s'", k, existing, action)
		}
		keyToAction[k] = action
	}
	return nil
}

// GetDefaultKeyBindings returns the default key bindings as a sorted map
func GetDefaultKeyBindings() map[string][]string {
	return map[string][]string{
		"archive":      {"a"},
		"clear_filter": {"esc"},
		"comment":      {"c"},
		"cycle_status": {"s"},
		"detach":       {"ctrl+q"},
		"down":         {"down", "j"},
		"filter":       {"/"},
		"flag":         {"f"},
		"force_quit":   {"ctrl+c"},
		"help":         {"h", "?"},
		"kill":         {"x"},
		"move_down":    {"J", "shift+down"},
		"move_up":      {"K", "shift+up"},
		"new":          {"n"},
		"new_from_repo": {"N"},
		"open":         {"enter"},
		"open_editor":  {"o"},
		"open_shell":   {"ctrl+s"},
		"quick_open":   {"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"},
		"quit":         {"q"},
		"rename":       {"r"},
		"send_text":    {"p"},
		"set_status":   {"S"},
		"timestamps":   {"t"},
		"up":           {"up", "k"},
	}
}

// GetSortedKeyBindingNames returns key binding names in sorted order.
// Note: ValidKeyBindingNames is already sorted, but we sort defensively
// to ensure correctness even if keys are added out of order in the future.
func GetSortedKeyBindingNames() []string {
	names := make([]string, len(ValidKeyBindingNames))
	copy(names, ValidKeyBindingNames)
	sort.Strings(names)
	return names
}

// IsValidKeyBindingName checks if a name is a valid key binding name
func IsValidKeyBindingName(name string) bool {
	for _, valid := range ValidKeyBindingNames {
		if name == valid {
			return true
		}
	}
	return false
}

// DefaultTmuxStatusPosition is the default tmux status bar position
const DefaultTmuxStatusPosition = "bottom"

// Settings represents the structure of ~/.rocha/settings.json
type Settings struct {
	AllowDangerouslySkipPermissions *bool              `json:"allow_dangerously_skip_permissions,omitempty"`
	Debug                           *bool              `json:"debug,omitempty"`
	Editor                          string             `json:"editor,omitempty"`
	ErrorClearDelay                 *int               `json:"error_clear_delay,omitempty"`
	Keys                            *KeyBindingsConfig `json:"keys,omitempty"`
	MaxLogFiles                     *int               `json:"max_log_files,omitempty"`
	ShowTimestamps                  *bool              `json:"show_timestamps,omitempty"`
	StatusColors                    StringArray        `json:"status_colors,omitempty"`
	Statuses                        StringArray        `json:"statuses,omitempty"`
	TipsDisplayDurationSeconds      *int               `json:"tips_display_duration_seconds,omitempty"`
	TipsEnabled                     *bool              `json:"tips_enabled,omitempty"`
	TipsShowIntervalSeconds         *int               `json:"tips_show_interval_seconds,omitempty"`
	TmuxStatusPosition              string             `json:"tmux_status_position,omitempty"`
}

// StringArray supports both JSON arrays and comma-separated strings
type StringArray []string

// UnmarshalJSON implements custom unmarshaling for StringArray
func (sa *StringArray) UnmarshalJSON(data []byte) error {
	// Try array format first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*sa = arr
		return nil
	}

	// Fall back to comma-separated string
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*sa = parseCommaSeparated(str)
	return nil
}

// parseCommaSeparated splits comma-separated string and trims whitespace
func parseCommaSeparated(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// LoadSettings loads settings from $ROCHA_HOME/settings.json (or ~/.rocha/settings.json if not set)
// Returns empty Settings if file doesn't exist (not an error)
func LoadSettings() (*Settings, error) {
	path := GetSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{}, nil // Not an error, use defaults
		}
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("invalid settings.json: %w", err)
	}

	// Expand Editor path if it starts with ~
	if settings.Editor != "" {
		settings.Editor = ExpandPath(settings.Editor)
	}

	return &settings, nil
}

// SaveSettings saves settings to $ROCHA_HOME/settings.json
func SaveSettings(settings *Settings) error {
	path := GetSettingsPath()
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}
