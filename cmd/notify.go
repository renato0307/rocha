package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"rocha/logging"
	"rocha/sound"
	"rocha/state"
)

// NotifyCmd handles notification events from Claude hooks
type NotifyCmd struct {
	SessionName string `arg:"" help:"Name of the session triggering the notification"`
	EventType   string `arg:"" help:"Type of event: stop, prompt, start" default:"stop"`
	ExecutionID string `help:"Execution ID from parent rocha TUI" optional:""`
}

// Run executes the notification handler
func (n *NotifyCmd) Run() error {
	// Always initialize hook-specific logging for easier debugging
	hookLogFile, err := logging.InitHookLogger(n.SessionName, n.EventType)
	if err != nil {
		logging.Logger.Warn("Failed to initialize hook logger", "error", err)
	} else {
		logging.Logger.Info("Hook logger initialized", "log_file", hookLogFile)
	}

	// Determine execution ID: flag > env var > state file > "unknown"
	executionID := n.ExecutionID
	if executionID == "" {
		executionID = os.Getenv("ROCHA_EXECUTION_ID")
		if executionID == "" {
			// Load state file to find execution ID for this session
			st, err := state.Load()
			if err == nil {
				if session, exists := st.Sessions[n.SessionName]; exists {
					executionID = session.ExecutionID
					logging.Logger.Info("Using execution ID from session state", "execution_id", executionID)
				} else {
					executionID = st.ExecutionID
					logging.Logger.Info("Using global execution ID from state", "execution_id", executionID)
				}
			} else {
				executionID = "unknown"
				logging.Logger.Warn("Could not determine execution ID", "error", err)
			}
		}
	}

	logging.Logger.Info("=== NOTIFY HOOK TRIGGERED ===",
		"session", n.SessionName,
		"event", n.EventType,
		"execution_id", executionID,
		"timestamp", time.Now().Format(time.RFC3339Nano),
		"pid", os.Getpid(),
		"ppid", os.Getppid())

	// Play sound for stop, start, notification, and end events (not for prompt - user already knows they submitted)
	if n.EventType == "stop" || n.EventType == "start" || n.EventType == "notification" || n.EventType == "end" {
		logging.Logger.Debug("Playing notification sound", "event", n.EventType)
		if err := sound.PlaySoundForEvent(n.EventType); err != nil {
			logging.Logger.Error("Failed to play sound", "error", err)
			return fmt.Errorf("failed to play notification sound: %w", err)
		}
		logging.Logger.Debug("Sound played successfully")
	} else {
		logging.Logger.Debug("Skipping sound for event type", "event", n.EventType)
	}

	// Map event type to session state
	var sessionState string
	switch n.EventType {
	case "stop":
		sessionState = state.StateIdle // Claude finished working
	case "notification":
		sessionState = state.StateWaitingUser // Claude needs user input
	case "start":
		sessionState = state.StateIdle // Session started and ready for input
	case "prompt":
		sessionState = state.StateWorking // User submitted prompt
	case "end":
		sessionState = state.StateExited // Claude has exited
	default:
		// Unknown event type, don't update state
		logging.Logger.Warn("Unknown event type, skipping state update", "event", n.EventType)
		return nil
	}

	logging.Logger.Debug("Mapped event to state",
		"event", n.EventType,
		"state", sessionState)

	// Queue session state update
	if err := state.QueueUpdateSession(n.SessionName, sessionState, executionID); err != nil {
		log.Printf("Warning: failed to queue session state update: %v", err)
		logging.Logger.Error("Failed to queue session state update", "error", err)
		return nil // Don't fail notification on state errors
	}

	logging.Logger.Info("Session state update queued successfully",
		"session", n.SessionName,
		"state", sessionState,
		"execution_id", executionID)

	// Future: Add OS native notifications here
	// For example:
	// - Linux: notify-send
	// - macOS: osascript -e 'display notification...'
	// - Windows: Windows Toast notifications

	return nil
}
