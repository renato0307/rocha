package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"rocha/logging"
	"rocha/sound"
	"rocha/state"
	"rocha/storage"
	"rocha/tmux"
)

// NotifyCmd handles notification events from Claude hooks
type NotifyCmd struct {
	SessionName string `arg:"" help:"Name of the session triggering the notification"`
	EventType   string `arg:"" help:"Type of event: stop, prompt, working, start, notification, end" default:"stop"`
	ExecutionID string `help:"Execution ID from parent rocha TUI" optional:""`
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
		sessionState = state.StateIdle // Session started and ready

		// Check for initial prompt and send it
		logging.Logger.Debug("Checking for initial prompt", "session", n.SessionName, "db_path", dbPath)
		tmpStore, err := storage.NewStore(dbPath)
		if err != nil {
			logging.Logger.Error("Failed to open database for initial prompt", "error", err)
		} else {
			defer tmpStore.Close()
			session, err := tmpStore.GetSession(context.Background(), n.SessionName)
			if err != nil {
				logging.Logger.Error("Failed to get session for initial prompt", "error", err, "session", n.SessionName)
			} else {
				logging.Logger.Debug("Session retrieved", "session", n.SessionName, "initial_prompt_length", len(session.InitialPrompt))
				if session.InitialPrompt != "" {
					logging.Logger.Info("Sending initial prompt", "session", n.SessionName)

					client := tmux.NewClient()
					escapedPrompt := shellEscape(session.InitialPrompt)

					if err := client.SendKeys(n.SessionName, escapedPrompt, "Enter"); err != nil {
						logging.Logger.Error("Failed to send initial prompt", "error", err)
					} else {
						logging.Logger.Info("Initial prompt sent successfully")
						sessionState = state.StateWorking // Update state since prompt was submitted
					}
				} else {
					logging.Logger.Debug("No initial prompt found for session", "session", n.SessionName)
				}
			}
		}
	case "prompt":
		sessionState = state.StateWorking // User submitted prompt
	case "working":
		sessionState = state.StateWorking // Claude is actively working (after answering question)
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

// shellEscape escapes a string for safe use in shell commands
// Uses single-quote escaping which is POSIX-compliant and handles all special chars
func shellEscape(s string) string {
	// Replace ' with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return fmt.Sprintf("'%s'", escaped)
}
