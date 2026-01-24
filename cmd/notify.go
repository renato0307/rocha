package cmd

import (
	"context"
	"log"
	"os"
	"time"

	"rocha/logging"
	"rocha/sound"
	"rocha/state"
	"rocha/storage"
)

// NotifyCmd handles notification events from Claude hooks
type NotifyCmd struct {
	EnableSound bool   `help:"Enable notification sounds" env:"ROCHA_ENABLE_SOUND"`
	EventType   string `arg:"" help:"Type of event: stop, prompt, start" default:"stop"`
	ExecutionID string `help:"Execution ID from parent rocha TUI" optional:""`
	SessionName string `arg:"" help:"Name of the session triggering the notification"`
}

// resolveSoundEnabled determines if sound should be enabled based on precedence:
// CLI flag > environment variable > settings.json > default (false)
func (n *NotifyCmd) resolveSoundEnabled(cli *CLI) bool {
	// Check if flag was explicitly set (non-default value)
	if n.EnableSound {
		return true
	}

	// Check settings.json (Kong's env tag handles ROCHA_ENABLE_SOUND automatically)
	if cli.settings != nil && cli.settings.SoundEnabled != nil {
		return *cli.settings.SoundEnabled
	}

	// Default: sound OFF
	return false
}

// Run executes the notification handler
func (n *NotifyCmd) Run(cli *CLI) error {
	// Always initialize hook-specific logging for easier debugging
	hookLogFile, err := logging.InitHookLogger(n.SessionName, n.EventType)
	if err != nil {
		logging.Logger.Warn("Failed to initialize hook logger", "error", err)
	} else {
		logging.Logger.Info("Hook logger initialized", "log_file", hookLogFile)
	}

	// Get database path
	dbPath := expandPath(cli.DBPath)

	// Determine execution ID: flag > env var > database > "unknown"
	executionID := n.ExecutionID
	if executionID == "" {
		executionID = os.Getenv("ROCHA_EXECUTION_ID")
		if executionID == "" {
			// Load from database to find execution ID for this session
			store, err := storage.NewStore(dbPath)
			if err == nil {
				defer store.Close()
				session, err := store.GetSession(context.Background(), n.SessionName)
				if err == nil {
					executionID = session.ExecutionID
					logging.Logger.Info("Using execution ID from session state", "execution_id", executionID)
				} else {
					// Session not found, use unknown
					executionID = "unknown"
					logging.Logger.Warn("Could not determine execution ID", "error", err)
				}
			} else {
				executionID = "unknown"
				logging.Logger.Warn("Could not open database", "error", err)
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

	// Play sound for stop, start, notification, and end events if enabled
	shouldPlaySound := n.resolveSoundEnabled(cli)
	if shouldPlaySound && (n.EventType == "stop" || n.EventType == "start" ||
		n.EventType == "notification" || n.EventType == "end") {
		logging.Logger.Debug("Playing notification sound", "event", n.EventType, "enabled", true)
		if err := sound.PlaySoundForEvent(n.EventType); err != nil {
			logging.Logger.Error("Failed to play sound", "error", err)
			// Don't fail notification on sound error
		}
		logging.Logger.Debug("Sound played successfully")
	} else {
		logging.Logger.Debug("Skipping sound", "event", n.EventType, "enabled", shouldPlaySound)
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

	// Update session state directly in SQLite (NO MORE EVENT QUEUE!)
	store, err := storage.NewStore(dbPath)
	if err != nil {
		log.Printf("Warning: failed to open database: %v", err)
		logging.Logger.Error("Failed to open database", "error", err)
		return nil // Don't fail notification on state errors
	}
	defer store.Close()

	if err := store.UpdateSession(context.Background(), n.SessionName, sessionState, executionID); err != nil {
		log.Printf("Warning: failed to update session state: %v", err)
		logging.Logger.Error("Failed to update session state", "error", err)
		return nil // Don't fail notification on state errors
	}

	logging.Logger.Info("Session state updated successfully (direct SQLite write)",
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
