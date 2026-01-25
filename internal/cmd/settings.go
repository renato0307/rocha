package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"rocha/internal/config"
	"text/tabwriter"
)

// SettingsCmd manages settings
type SettingsCmd struct {
	Meta SettingsMetaCmd `cmd:"meta" help:"Show settings file location and available options" default:"1"`
}

// SettingsMetaCmd displays settings metadata
type SettingsMetaCmd struct {
	Format string `help:"Output format: table or json" enum:"table,json" default:"table"`
}

// Run executes the meta command
func (s *SettingsMetaCmd) Run(cli *CLI) error {
	settingsFile := config.GetSettingsFilePath()
	example := config.GetSettingsExample()

	if s.Format == "json" {
		output := map[string]any{
			"settings_file": settingsFile,
			"format":        example,
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("Settings file: %s\n\n", settingsFile)
	fmt.Println("Example settings.json:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Get keys in sorted order from the example map
	// We'll print them in the order they appear in the map
	for key, value := range example {
		// Format the value for display
		var valueStr string
		switch v := value.(type) {
		case []string:
			// Format string arrays as JSON
			data, _ := json.Marshal(v)
			valueStr = string(data)
		case string:
			valueStr = v
		case bool:
			valueStr = fmt.Sprintf("%t", v)
		case int:
			valueStr = fmt.Sprintf("%d", v)
		default:
			valueStr = fmt.Sprintf("%v", v)
		}

		fmt.Fprintf(w, "%s\t%s\n", key, valueStr)
	}

	w.Flush()

	fmt.Println()
	fmt.Println("Create or edit this file to configure rocha.")
	fmt.Println("All settings are optional and have sensible defaults.")

	return nil
}
