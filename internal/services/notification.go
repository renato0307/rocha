package services

import (
	"context"
	"os"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// NotificationService handles notification events from Claude hooks
type NotificationService struct {
	sessionReader ports.SessionReader
	sessionRepo   ports.SessionStateUpdater
	soundPlayer   ports.SoundPlayer
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(
	sessionRepo ports.SessionStateUpdater,
	sessionReader ports.SessionReader,
	soundPlayer ports.SoundPlayer,
) *NotificationService {
	return &NotificationService{
		sessionReader: sessionReader,
		sessionRepo:   sessionRepo,
		soundPlayer:   soundPlayer,
	}
}

// HandleEvent processes a notification event and updates session state
// Returns the mapped session state for the event type
func (s *NotificationService) HandleEvent(
	ctx context.Context,
	sessionName string,
	eventType string,
	executionID string,
) (domain.SessionState, error) {
	// Map event type to session state
	var sessionState domain.SessionState
	switch eventType {
	case "stop":
		sessionState = domain.StateIdle // Claude finished working
	case "notification", "permission-request":
		// Keep both for backward compatibility
		sessionState = domain.StateWaiting // Claude is waiting for user input or permission
	case "start":
		sessionState = domain.StateIdle // Session started and ready for input
	case "prompt":
		sessionState = domain.StateWorking // User submitted prompt
	case "working":
		sessionState = domain.StateWorking // Claude is actively working
	case "tool-failure":
		sessionState = domain.StateWorking // Tool failed, Claude continues
	case "subagent-start":
		sessionState = domain.StateWorking // Spawning subagent
	case "subagent-stop":
		sessionState = domain.StateWorking // Subagent finished
	case "pre-compact":
		sessionState = domain.StateWorking // Context compression
	case "setup":
		sessionState = domain.StateWorking // Init/maintenance work
	case "end":
		sessionState = domain.StateExited // Claude has exited
	default:
		// Unknown event type, don't update state
		logging.Logger.Warn("Unknown event type, skipping state update", "event", eventType)
		return "", nil
	}

	logging.Logger.Debug("Mapped event to state", "event", eventType, "state", sessionState)

	// Update session state in repository
	if err := s.sessionRepo.UpdateState(ctx, sessionName, sessionState, executionID); err != nil {
		logging.Logger.Error("Failed to update session state", "error", err)
		return sessionState, err
	}

	logging.Logger.Info("Session state updated successfully",
		"session", sessionName,
		"state", sessionState,
		"execution_id", executionID)

	return sessionState, nil
}

// ResolveExecutionID determines execution ID with precedence:
// flag value > env var > database > "unknown"
func (s *NotificationService) ResolveExecutionID(
	ctx context.Context,
	sessionName string,
	flagValue string,
) string {
	// 1. Flag value takes precedence
	if flagValue != "" {
		return flagValue
	}

	// 2. Check environment variable
	if envValue := os.Getenv("ROCHA_EXECUTION_ID"); envValue != "" {
		return envValue
	}

	// 3. Try to load from database
	session, err := s.sessionReader.Get(ctx, sessionName)
	if err == nil && session.ExecutionID != "" {
		logging.Logger.Info("Using execution ID from session state", "execution_id", session.ExecutionID)
		return session.ExecutionID
	}

	if err != nil {
		logging.Logger.Warn("Could not determine execution ID", "error", err)
	}

	// 4. Fall back to unknown
	return "unknown"
}

// ShouldPlaySound determines if a sound should be played for the event type
func (s *NotificationService) ShouldPlaySound(eventType string) bool {
	switch eventType {
	case "stop", "start", "notification", "permission-request", "end":
		return true // User-facing events
	case "tool-failure", "subagent-start", "subagent-stop", "pre-compact", "setup":
		return false // Internal operations
	default:
		return false
	}
}

// PlaySound plays the default notification sound
func (s *NotificationService) PlaySound() error {
	logging.Logger.Debug("Playing notification sound")
	return s.soundPlayer.PlaySound()
}

// PlaySoundForEvent plays a sound for a specific event type
func (s *NotificationService) PlaySoundForEvent(eventType string) error {
	logging.Logger.Debug("Playing sound for event", "event", eventType)
	return s.soundPlayer.PlaySoundForEvent(eventType)
}
