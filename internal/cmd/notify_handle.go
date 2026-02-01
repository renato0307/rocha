package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/renato0307/rocha/internal/logging"
)

// NotifyHandleCmd handles notification events from Claude hooks
// NOTE: Field order matters for Kong positional args - SessionName must come before EventType
type NotifyHandleCmd struct {
	SessionName string `arg:"" help:"Name of the session triggering the notification"`
	EventType   string `arg:"" help:"Type of event: stop, prompt, working, start, notification, end, permission-request, tool-complete, tool-failure, subagent-start, subagent-stop, pre-compact, setup" default:"stop"`
	ExecutionID string `help:"Execution ID from parent rocha TUI" optional:""`
}

// Run executes the notification handler
func (n *NotifyHandleCmd) Run(cli *CLI) error {
	// Always initialize hook-specific logging for easier debugging
	hookLogFile, err := logging.InitHookLogger(n.SessionName, n.EventType)
	if err != nil {
		logging.Logger.Warn("Failed to initialize hook logger", "error", err)
	} else {
		logging.Logger.Info("Hook logger initialized", "log_file", hookLogFile)
	}

	// Resolve execution ID using the service
	executionID := cli.Container.NotificationService.ResolveExecutionID(
		context.Background(),
		n.SessionName,
		n.ExecutionID,
	)

	logging.Logger.Info("=== NOTIFY HOOK TRIGGERED ===",
		"session", n.SessionName,
		"event", n.EventType,
		"execution_id", executionID,
		"timestamp", time.Now().Format(time.RFC3339Nano),
		"pid", os.Getpid(),
		"ppid", os.Getppid())

	// Play sound (presentation concern - stays in command)
	if cli.Container.NotificationService.ShouldPlaySound(n.EventType) {
		logging.Logger.Debug("Playing notification sound", "event", n.EventType)
		if err := cli.Container.NotificationService.PlaySoundForEvent(n.EventType); err != nil {
			logging.Logger.Error("Failed to play sound", "error", err)
			return fmt.Errorf("failed to play notification sound: %w", err)
		}
		logging.Logger.Debug("Sound played successfully")
	} else {
		logging.Logger.Debug("Skipping sound for event type", "event", n.EventType)
	}

	// Handle event using the service
	_, err = cli.Container.NotificationService.HandleEvent(
		context.Background(),
		n.SessionName,
		n.EventType,
		executionID,
	)
	if err != nil {
		logging.Logger.Error("Failed to handle notification event", "error", err)
		return nil // Don't fail notification on state errors
	}

	// Future: Add OS native notifications here
	// For example:
	// - Linux: notify-send
	// - macOS: osascript -e 'display notification...'
	// - Windows: Windows Toast notifications

	return nil
}
