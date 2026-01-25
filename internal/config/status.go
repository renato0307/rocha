package config

import "strings"

// StatusConfig holds the configuration for session implementation statuses
type StatusConfig struct {
	Colors   []string
	Icons    []string
	Statuses []string
}

// NewStatusConfig creates a new StatusConfig from comma-separated strings
func NewStatusConfig(statuses, icons, colors string) *StatusConfig {
	config := &StatusConfig{
		Statuses: parseList(statuses),
		Icons:    parseList(icons),
		Colors:   parseList(colors),
	}

	// If no statuses provided, use default
	if len(config.Statuses) == 0 {
		config.Statuses = []string{"spec", "plan", "implement", "review", "done"}
	}

	// If no colors provided, use default palette
	if len(config.Colors) == 0 {
		config.Colors = []string{"141", "33", "214", "226", "46"}
	}

	return config
}

// GetIcon returns the icon for a given status name, or empty string if not found
func (c *StatusConfig) GetIcon(status string) string {
	for i, s := range c.Statuses {
		if s == status {
			if i < len(c.Icons) {
				return c.Icons[i]
			}
			return ""
		}
	}
	return ""
}

// GetColor returns a color for a given status based on its position
// Uses configured color palette for visual distinction
func (c *StatusConfig) GetColor(status string) string {
	for i, s := range c.Statuses {
		if s == status {
			if i < len(c.Colors) {
				return c.Colors[i]
			}
			// If more statuses than colors, cycle through colors
			return c.Colors[i%len(c.Colors)]
		}
	}

	// Default color if status not found
	if len(c.Colors) > 0 {
		return c.Colors[0]
	}
	return "141"
}

// GetNextStatus returns the next status in the cycle
// nil -> first status -> second status -> ... -> last status -> nil
func (c *StatusConfig) GetNextStatus(currentStatus *string) *string {
	// If no statuses defined, return nil
	if len(c.Statuses) == 0 {
		return nil
	}

	// If current status is nil, return first status
	if currentStatus == nil || *currentStatus == "" {
		return &c.Statuses[0]
	}

	// Find current status position
	for i, s := range c.Statuses {
		if s == *currentStatus {
			// If at the end, cycle back to nil
			if i == len(c.Statuses)-1 {
				return nil
			}
			// Otherwise, return next status
			nextStatus := c.Statuses[i+1]
			return &nextStatus
		}
	}

	// Current status not found in list, return first status
	return &c.Statuses[0]
}

// parseList splits a comma-separated string into a list, trimming whitespace
func parseList(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
