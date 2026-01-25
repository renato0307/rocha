package cmd

import (
	"context"
	"fmt"

	"github.com/renato0307/rocha/internal/logging"
)

// SessionsStatusCmd sets or clears the implementation status of a session
type SessionsStatusCmd struct {
	List   bool   `help:"List available statuses" short:"l" xor:"action"`
	Name   string `arg:"" optional:"" help:"Session name"`
	Status string `help:"Status (empty or 'clear' clears)" xor:"action"`
}

// AfterApply validates that Name is provided when not listing
func (s *SessionsStatusCmd) AfterApply() error {
	if !s.List && s.Name == "" {
		return fmt.Errorf("session name is required when setting status")
	}
	return nil
}

// Run executes the status command
func (s *SessionsStatusCmd) Run(cli *CLI) error {
	logging.Logger.Debug("Executing sessions status command", "name", s.Name, "status", s.Status, "list", s.List)

	if s.List {
		return s.listStatuses()
	}
	return s.setStatus()
}

func (s *SessionsStatusCmd) listStatuses() error {
	container, err := NewContainer(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	statuses, err := container.SettingsService.GetAvailableStatuses()
	if err != nil {
		return fmt.Errorf("failed to get available statuses: %w", err)
	}

	fmt.Println("Available statuses:")
	for _, status := range statuses {
		fmt.Printf("  - %s\n", status)
	}
	return nil
}

func (s *SessionsStatusCmd) setStatus() error {
	container, err := NewContainer(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	ctx := context.Background()

	// Validate session exists
	if _, err := container.SessionService.GetSession(ctx, s.Name); err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Determine status value
	var statusPtr *string
	if s.Status != "" && s.Status != "clear" {
		statusPtr = &s.Status
	}

	if err := container.SessionService.UpdateStatus(ctx, s.Name, statusPtr); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if statusPtr == nil {
		fmt.Printf("Status cleared for session '%s'\n", s.Name)
	} else {
		fmt.Printf("Status set to '%s' for session '%s'\n", s.Status, s.Name)
	}
	return nil
}
