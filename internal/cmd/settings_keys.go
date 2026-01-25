package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ui"
)

// SettingsKeysCmd manages keyboard shortcuts
type SettingsKeysCmd struct {
	List SettingsKeysListCmd `cmd:"list" help:"List all key bindings (defaults and custom)" default:"1"`
	Set  SettingsKeysSetCmd  `cmd:"set" help:"Set a key binding"`
}

// SettingsKeysListCmd lists all key bindings
type SettingsKeysListCmd struct {
	Format string `help:"Output format: table or json" enum:"table,json" default:"table"`
}

// SettingsKeysSetCmd sets a key binding
type SettingsKeysSetCmd struct {
	Key   string `arg:"" help:"Key name (e.g., archive, help, quit)"`
	Value string `arg:"" help:"Key binding (e.g., a, ctrl+s, or comma-separated for multiple: up,k)"`
}

// Run executes the list command
func (s *SettingsKeysListCmd) Run(cli *CLI) error {
	defaults := ui.GetDefaultKeyBindings()
	names := ui.GetValidKeyNames()

	// Get custom bindings from settings
	var customKeys config.KeyBindingsConfig
	if cli.settings != nil && cli.settings.Keys != nil {
		customKeys = cli.settings.Keys
	}

	if s.Format == "json" {
		return s.outputJSON(names, defaults, customKeys)
	}

	return s.outputTable(names, defaults, customKeys)
}

func (s *SettingsKeysListCmd) outputJSON(names []string, defaults map[string][]string, customKeys config.KeyBindingsConfig) error {
	result := make(map[string]map[string]any)

	for _, name := range names {
		entry := make(map[string]any)
		entry["default"] = defaults[name]

		if custom, ok := customKeys[name]; ok && len(custom) > 0 {
			entry["custom"] = custom
		}

		result[name] = entry
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func (s *SettingsKeysListCmd) outputTable(names []string, defaults map[string][]string, customKeys config.KeyBindingsConfig) error {
	settingsFile := config.GetSettingsPath()
	fmt.Printf("Key Bindings (settings file: %s)\n\n", settingsFile)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "Name\tDefault\tCustom")
	fmt.Fprintln(w, "────\t───────\t──────")

	for _, name := range names {
		defaultKeys := strings.Join(defaults[name], ", ")
		customStr := "-"

		if custom, ok := customKeys[name]; ok && len(custom) > 0 {
			customStr = strings.Join(custom, ", ")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", name, defaultKeys, customStr)
	}

	w.Flush()

	fmt.Println()
	fmt.Println("Use 'rocha settings keys set <name> <value>' to customize.")
	return nil
}

// Run executes the set command
func (s *SettingsKeysSetCmd) Run(cli *CLI) error {
	// Validate key name
	if !ui.IsValidKeyName(s.Key) {
		return fmt.Errorf("unknown key '%s'. Valid keys: %s",
			s.Key, strings.Join(ui.GetValidKeyNames(), ", "))
	}

	// Parse value (comma-separated for multiple keys)
	values := parseKeyValues(s.Value)
	if len(values) == 0 {
		return fmt.Errorf("value cannot be empty")
	}

	logging.Logger.Debug("Setting key binding", "key", s.Key, "values", values)

	// Load existing settings
	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Initialize Keys if needed
	if settings.Keys == nil {
		settings.Keys = make(config.KeyBindingsConfig)
	}

	// Set the new binding
	settings.Keys[s.Key] = values

	// Validate for conflicts
	if err := settings.Keys.Validate(ui.GetValidKeyNames()); err != nil {
		return fmt.Errorf("conflict: %w", err)
	}

	// Save settings
	if err := config.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	fmt.Printf("Set '%s' to: %s\n", s.Key, strings.Join(values, ", "))
	return nil
}

// parseKeyValues parses comma-separated key values
func parseKeyValues(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
