package services

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/renato0307/rocha/internal/domain"
	portsmocks "github.com/renato0307/rocha/internal/ports/mocks"
)

func TestHandleEvent_EventTypeToStateMapping(t *testing.T) {
	tests := []struct {
		eventType     string
		expectedState domain.SessionState
	}{
		{"stop", domain.StateIdle},
		{"notification", domain.StateWaiting},
		{"permission-request", domain.StateWaiting},
		{"start", domain.StateIdle},
		{"prompt", domain.StateWorking},
		{"working", domain.StateWorking},
		{"tool-complete", domain.StateWorking},
		{"tool-failure", domain.StateWorking},
		{"subagent-start", domain.StateWorking},
		{"subagent-stop", domain.StateWorking},
		{"pre-compact", domain.StateWorking},
		{"setup", domain.StateWorking},
		{"end", domain.StateExited},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			sessionReader := portsmocks.NewMockSessionReader(t)
			stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
			soundPlayer := portsmocks.NewMockSoundPlayer(t)

			// For non-intermediate events, just expect UpdateState
			// For intermediate events, we need to set up Get first
			intermediateEvents := map[string]bool{
				"tool-complete":  true,
				"tool-failure":   true,
				"subagent-start": true,
				"subagent-stop":  true,
				"pre-compact":    true,
				"setup":          true,
			}

			if intermediateEvents[tt.eventType] {
				// Return a working state so intermediate event proceeds
				sessionReader.EXPECT().Get(mock.Anything, "test-session").
					Return(&domain.Session{State: domain.StateWorking}, nil)
			}

			stateUpdater.EXPECT().UpdateState(mock.Anything, "test-session", tt.expectedState, "exec-123").
				Return(nil)

			service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

			state, err := service.HandleEvent(context.Background(), "test-session", tt.eventType, "exec-123")

			require.NoError(t, err)
			assert.Equal(t, tt.expectedState, state)
		})
	}
}

func TestHandleEvent_UnknownEventType(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	state, err := service.HandleEvent(context.Background(), "test-session", "unknown-event", "exec-123")

	require.NoError(t, err)
	assert.Empty(t, state)
}

func TestHandleEvent_UpdateStateError(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	stateUpdater.EXPECT().UpdateState(mock.Anything, "test-session", domain.StateIdle, "exec-123").
		Return(errors.New("database error"))

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	state, err := service.HandleEvent(context.Background(), "test-session", "stop", "exec-123")

	require.Error(t, err)
	assert.Equal(t, domain.StateIdle, state)
}

func TestHandleEvent_IntermediateEventSkipsTerminalState(t *testing.T) {
	tests := []struct {
		name         string
		eventType    string
		currentState domain.SessionState
	}{
		{"subagent-stop skips idle", "subagent-stop", domain.StateIdle},
		{"subagent-stop skips exited", "subagent-stop", domain.StateExited},
		{"tool-complete skips idle", "tool-complete", domain.StateIdle},
		{"setup skips exited", "setup", domain.StateExited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionReader := portsmocks.NewMockSessionReader(t)
			stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
			soundPlayer := portsmocks.NewMockSoundPlayer(t)

			// Return terminal state - intermediate event should not overwrite
			sessionReader.EXPECT().Get(mock.Anything, "test-session").
				Return(&domain.Session{State: tt.currentState}, nil)

			// Note: UpdateState should NOT be called
			service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

			state, err := service.HandleEvent(context.Background(), "test-session", tt.eventType, "exec-123")

			require.NoError(t, err)
			assert.Equal(t, tt.currentState, state)
		})
	}
}

func TestHandleEvent_IntermediateEventProceedsOnGetError(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	// Get fails but event should still proceed (fail open)
	sessionReader.EXPECT().Get(mock.Anything, "test-session").
		Return(nil, errors.New("not found"))
	stateUpdater.EXPECT().UpdateState(mock.Anything, "test-session", domain.StateWorking, "exec-123").
		Return(nil)

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	state, err := service.HandleEvent(context.Background(), "test-session", "subagent-stop", "exec-123")

	require.NoError(t, err)
	assert.Equal(t, domain.StateWorking, state)
}

func TestResolveExecutionID_FlagValueTakesPrecedence(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	result := service.ResolveExecutionID(context.Background(), "test-session", "flag-value")

	assert.Equal(t, "flag-value", result)
}

func TestResolveExecutionID_EnvVarSecondPriority(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	os.Setenv("ROCHA_EXECUTION_ID", "env-value")
	defer os.Unsetenv("ROCHA_EXECUTION_ID")

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	result := service.ResolveExecutionID(context.Background(), "test-session", "")

	assert.Equal(t, "env-value", result)
}

func TestResolveExecutionID_DatabaseThirdPriority(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	// Make sure env var is not set
	os.Unsetenv("ROCHA_EXECUTION_ID")

	sessionReader.EXPECT().Get(mock.Anything, "test-session").
		Return(&domain.Session{ExecutionID: "db-value"}, nil)

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	result := service.ResolveExecutionID(context.Background(), "test-session", "")

	assert.Equal(t, "db-value", result)
}

func TestResolveExecutionID_FallbackToUnknown(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	// Make sure env var is not set
	os.Unsetenv("ROCHA_EXECUTION_ID")

	sessionReader.EXPECT().Get(mock.Anything, "test-session").
		Return(nil, errors.New("not found"))

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	result := service.ResolveExecutionID(context.Background(), "test-session", "")

	assert.Equal(t, "unknown", result)
}

func TestResolveExecutionID_EmptyDbValueFallsBack(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	// Make sure env var is not set
	os.Unsetenv("ROCHA_EXECUTION_ID")

	sessionReader.EXPECT().Get(mock.Anything, "test-session").
		Return(&domain.Session{ExecutionID: ""}, nil)

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	result := service.ResolveExecutionID(context.Background(), "test-session", "")

	assert.Equal(t, "unknown", result)
}

func TestShouldPlaySound(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	tests := []struct {
		eventType string
		expected  bool
	}{
		{"stop", true},
		{"start", true},
		{"notification", true},
		{"permission-request", true},
		{"end", true},
		{"tool-failure", false},
		{"subagent-start", false},
		{"subagent-stop", false},
		{"pre-compact", false},
		{"setup", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			result := service.ShouldPlaySound(tt.eventType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlaySound(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	soundPlayer.EXPECT().PlaySound().Return(nil)

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	err := service.PlaySound()

	require.NoError(t, err)
}

func TestPlaySoundForEvent(t *testing.T) {
	sessionReader := portsmocks.NewMockSessionReader(t)
	stateUpdater := portsmocks.NewMockSessionStateUpdater(t)
	soundPlayer := portsmocks.NewMockSoundPlayer(t)

	soundPlayer.EXPECT().PlaySoundForEvent("stop").Return(nil)

	service := NewNotificationService(stateUpdater, sessionReader, soundPlayer)

	err := service.PlaySoundForEvent("stop")

	require.NoError(t, err)
}
