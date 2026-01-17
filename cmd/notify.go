package cmd

import (
	"fmt"
	"log"
	"os"
	"rocha/logging"
	"rocha/state"
)

// NotifyCmd handles notification events from Claude hooks
type NotifyCmd struct {
	SessionName string `arg:"" help:"Name of the session triggering the notification"`
	EventType   string `arg:"" help:"Type of event: stop, prompt, start" default:"stop"`
}

// Run executes the notification handler
func (n *NotifyCmd) Run() error {
	logging.Logger.Info("Notification hook triggered",
		"session", n.SessionName,
		"event", n.EventType)

	// Play sound for stop, start, and notification events (not for prompt - user already knows they submitted)
	if n.EventType == "stop" || n.EventType == "start" || n.EventType == "notification" {
		logging.Logger.Debug("Playing notification sound", "event", n.EventType)
		if err := PlaySoundForEvent(n.EventType); err != nil {
			logging.Logger.Error("Failed to play sound", "error", err)
			return fmt.Errorf("failed to play notification sound: %w", err)
		}
		logging.Logger.Debug("Sound played successfully")
	} else {
		logging.Logger.Debug("Skipping sound for event type", "event", n.EventType)
	}

	// Update session state based on event type
	executionID := os.Getenv("ROCHA_EXECUTION_ID")
	if executionID == "" {
		executionID = "unknown" // Fallback for sessions not created by current TUI
		logging.Logger.Warn("No ROCHA_EXECUTION_ID found, using 'unknown'")
	}
	logging.Logger.Debug("Execution ID retrieved", "execution_id", executionID)

	// Map event type to session state
	var sessionState string
	switch n.EventType {
	case "stop", "notification":
		sessionState = state.StateWaiting // Claude finished or needs input
	case "prompt", "start":
		sessionState = state.StateWorking // User submitted prompt or session started
	default:
		// Unknown event type, don't update state
		logging.Logger.Warn("Unknown event type, skipping state update", "event", n.EventType)
		return nil
	}

	logging.Logger.Debug("Mapped event to state",
		"event", n.EventType,
		"state", sessionState)

	// Load state and update session
	st, err := state.Load()
	if err != nil {
		log.Printf("Warning: failed to load state: %v", err)
		logging.Logger.Error("Failed to load state", "error", err)
		return nil // Don't fail notification on state errors
	}

	logging.Logger.Debug("State loaded successfully", "sessions_count", len(st.Sessions))

	if err := st.UpdateSession(n.SessionName, sessionState, executionID); err != nil {
		log.Printf("Warning: failed to update session state: %v", err)
		logging.Logger.Error("Failed to update session state", "error", err)
	} else {
		logging.Logger.Info("Session state updated successfully",
			"session", n.SessionName,
			"state", sessionState,
			"execution_id", executionID)
	}

	// Future: Add OS native notifications here
	// For example:
	// - Linux: notify-send
	// - macOS: osascript -e 'display notification...'
	// - Windows: Windows Toast notifications

	return nil
}
