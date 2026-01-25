package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// GetSettingsFilePath returns the path to the settings file
func GetSettingsFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.rocha/settings.json" // Fallback to unexpanded path
	}
	return filepath.Join(homeDir, ".rocha", "settings.json")
}

// GetSettingsExample uses reflection to generate example settings
// This automatically stays in sync when new fields are added to Settings
func GetSettingsExample() map[string]any {
	var s Settings
	t := reflect.TypeOf(s)
	example := make(map[string]any)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		// Extract the JSON field name (before comma)
		jsonName := strings.Split(jsonTag, ",")[0]

		// Generate example value based on field type
		example[jsonName] = generateExampleValue(field.Type, jsonName)
	}

	return example
}

// generateExampleValue creates appropriate example values based on type and field name
func generateExampleValue(t reflect.Type, fieldName string) any {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		elemType := t.Elem()

		// Handle KeyBindingsConfig pointer
		if elemType.Name() == "KeyBindingsConfig" {
			return map[string]any{
				"archive": "A",
				"help":    []string{"H", "?"},
			}
		}

		switch elemType.Kind() {
		case reflect.Bool:
			// Return boolean value directly (not pointer)
			if fieldName == "debug" || fieldName == "show_timestamps" {
				return true
			}
			return false
		case reflect.Int:
			// Return int value directly (not pointer)
			if fieldName == "error_clear_delay" {
				return 10
			}
			if fieldName == "max_log_files" {
				return 1000
			}
			return 10
		}
	}

	// Handle direct types
	switch t.Kind() {
	case reflect.String:
		// Generate contextual examples based on field name
		switch fieldName {
		case "db_path":
			return "~/.rocha/state.db"
		case "editor":
			return "code"
		case "tmux_status_position":
			return "bottom"
		case "worktree_path":
			return "~/.rocha/worktrees"
		default:
			return "example"
		}
	case reflect.Slice:
		// Check if it's StringArray type
		if t.Name() == "StringArray" || (t.Elem().Kind() == reflect.String) {
			switch fieldName {
			case "status_colors":
				return []string{"141", "33", "214"}
			case "statuses":
				return []string{"spec", "plan", "implement"}
			default:
				return []string{"example1", "example2"}
			}
		}
	}

	return nil
}
