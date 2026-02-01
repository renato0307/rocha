package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

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

// KeyBindingsConfig holds custom key binding overrides as a map.
// Keys are binding names (e.g., "archive", "help"), values are the key sequences.
type KeyBindingsConfig map[string]KeyBindingValue

// Validate checks for configuration errors in key bindings.
// The validNames parameter should come from ui.GetValidKeyNames().
func (k KeyBindingsConfig) Validate(validNames []string) error {
	if k == nil {
		return nil
	}

	// Build set of valid names for quick lookup
	validSet := make(map[string]bool, len(validNames))
	for _, name := range validNames {
		validSet[name] = true
	}

	// Track all keys to detect duplicates
	keyToAction := make(map[string]string)

	// Validate each configured binding
	for name, keys := range k {
		// Check if the key name is valid
		if !validSet[name] {
			return fmt.Errorf("unknown key binding '%s'", name)
		}

		// Check for empty values and duplicates
		if len(keys) == 0 {
			continue // Not configured, will use default
		}

		for _, key := range keys {
			if key == "" {
				return fmt.Errorf("key binding for '%s' contains empty value", name)
			}
			if existing, found := keyToAction[key]; found {
				return fmt.Errorf("key '%s' is assigned to both '%s' and '%s'", key, existing, name)
			}
			keyToAction[key] = name
		}
	}

	return nil
}

// DefaultTmuxStatusPosition is the default tmux status bar position
const DefaultTmuxStatusPosition = "bottom"

// Settings represents the structure of ~/.rocha/settings.json
type Settings struct {
	AllowDangerouslySkipPermissions *bool             `json:"allow_dangerously_skip_permissions,omitempty"`
	Debug                           *bool             `json:"debug,omitempty"`
	DebugClaude                     *bool             `json:"debug_claude,omitempty"`
	Editor                          string            `json:"editor,omitempty"`
	ErrorClearDelay                 *int              `json:"error_clear_delay,omitempty"`
	Keys                            KeyBindingsConfig `json:"keys,omitempty"`
	MaxLogFiles                     *int              `json:"max_log_files,omitempty"`
	ShowTimestamps                  *bool             `json:"show_timestamps,omitempty"`
	ShowTokenChart                  *bool             `json:"show_token_chart,omitempty"`
	StatusColors                    StringArray       `json:"status_colors,omitempty"`
	Statuses                        StringArray       `json:"statuses,omitempty"`
	TipsDisplayDurationSeconds      *int              `json:"tips_display_duration_seconds,omitempty"`
	TipsEnabled                     *bool             `json:"tips_enabled,omitempty"`
	TipsShowIntervalSeconds         *int              `json:"tips_show_interval_seconds,omitempty"`
	TmuxStatusPosition              string            `json:"tmux_status_position,omitempty"`
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
