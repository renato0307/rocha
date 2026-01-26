package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/services"
)

// HooksCmd shows Claude Code hook events
type HooksCmd struct {
	Event       string `help:"Filter by hook event type" short:"e"`
	Format      string `help:"Output format (table or json)" default:"table" enum:"table,json" short:"f"`
	From        string `help:"Start time (RFC3339 or relative)"`
	Limit       int    `help:"Maximum number of results" default:"100" short:"l"`
	SessionName string `arg:"" optional:"" help:"Rocha session name"`
	To          string `help:"End time (RFC3339 or relative)"`
}

// Run executes the hooks command
func (h *HooksCmd) Run(cli *CLI) error {
	// Parse time strings
	var fromTime, toTime time.Time
	var err error

	if h.From != "" {
		fromTime, err = services.ParseTimeString(h.From)
		if err != nil {
			return fmt.Errorf("invalid --from time: %w", err)
		}
	}

	if h.To != "" {
		toTime, err = services.ParseTimeString(h.To)
		if err != nil {
			return fmt.Errorf("invalid --to time: %w", err)
		}
	}

	// Build filter
	filter := ports.HookFilter{
		EventType:   h.Event,
		From:        fromTime,
		Limit:       h.Limit,
		SessionName: h.SessionName,
		To:          toTime,
	}

	// Get hook events
	events, err := cli.Container.HookStatsService.GetHookEvents(filter)
	if err != nil {
		return fmt.Errorf("failed to get hook events: %w", err)
	}

	// Display based on format
	switch h.Format {
	case "json":
		h.renderJSON(events)
	default:
		h.renderTable(events)
	}

	return nil
}

// renderTable displays hook events in table format
func (h *HooksCmd) renderTable(events []domain.HookEvent) {
	today := time.Now().Format("2006-01-02")
	fmt.Printf("Hook Events - %s\n\n", today)

	if len(events) == 0 {
		fmt.Println("No hooks found.")
		return
	}

	// Header
	fmt.Println("Session          Event             Timestamp            Hook Name")
	fmt.Println(strings.Repeat("─", 80))

	// Data rows
	for _, e := range events {
		// Truncate session name if too long
		sessionName := e.SessionName
		if len(sessionName) > 16 {
			sessionName = sessionName[:13] + "..."
		}

		// Truncate event type if too long
		eventType := e.HookEvent
		if len(eventType) > 17 {
			eventType = eventType[:14] + "..."
		}

		// Truncate hook name if too long
		hookName := e.HookName
		if len(hookName) > 30 {
			hookName = hookName[:27] + "..."
		}

		fmt.Printf("%-16s %-17s %-20s %s\n",
			sessionName,
			eventType,
			e.Timestamp.Format("2006-01-02 15:04:05"),
			hookName)
	}
}

// hookEventJSON represents a hook event in JSON format
type hookEventJSON struct {
	Command     string `json:"command"`
	HookEvent   string `json:"hook_event"`
	HookName    string `json:"hook_name"`
	SessionName string `json:"session_name"`
	Timestamp   string `json:"timestamp"`
}

// renderJSON displays hook events in JSON format
func (h *HooksCmd) renderJSON(events []domain.HookEvent) {
	if len(events) == 0 {
		fmt.Println("[]")
		return
	}

	// Convert to JSON-friendly format
	var jsonEvents []hookEventJSON
	for _, e := range events {
		jsonEvents = append(jsonEvents, hookEventJSON{
			Command:     e.Command,
			HookEvent:   e.HookEvent,
			HookName:    e.HookName,
			SessionName: e.SessionName,
			Timestamp:   e.Timestamp.Format(time.RFC3339),
		})
	}

	// Marshal and print
	output, err := json.MarshalIndent(jsonEvents, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(output))
}
