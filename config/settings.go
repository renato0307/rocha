package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Settings represents the structure of ~/.rocha/settings.json
type Settings struct {
	DBPath          string      `json:"db_path,omitempty"`
	Debug           *bool       `json:"debug,omitempty"`
	Editor          string      `json:"editor,omitempty"`
	ErrorClearDelay *int        `json:"error_clear_delay,omitempty"`
	MaxLogFiles     *int        `json:"max_log_files,omitempty"`
	ShowTimestamps     *bool       `json:"show_timestamps,omitempty"`
	StatusColors       StringArray `json:"status_colors,omitempty"`
	Statuses           StringArray `json:"statuses,omitempty"`
	TmuxStatusPosition string      `json:"tmux_status_position,omitempty"`
	WorktreePath       string      `json:"worktree_path,omitempty"`
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

// LoadSettings loads settings from ~/.rocha/settings.json
// Returns empty Settings if file doesn't exist (not an error)
func LoadSettings() (*Settings, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	path := filepath.Join(homeDir, ".rocha", "settings.json")
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

	// Expand paths that start with ~
	if settings.DBPath != "" {
		settings.DBPath = expandPath(settings.DBPath)
	}
	if settings.WorktreePath != "" {
		settings.WorktreePath = expandPath(settings.WorktreePath)
	}

	return &settings, nil
}

// expandPath expands ~ to home directory in paths
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // Return as-is if we can't get home dir
		}
		if len(path) == 1 {
			return homeDir
		}
		return filepath.Join(homeDir, path[1:])
	}
	return path
}
