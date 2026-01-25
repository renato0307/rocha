package integration_test

import (
	"testing"

	"rocha/test/integration/harness"
)

func TestNotify(t *testing.T) {
	// The notify command is a hook handler that updates session state
	// based on events from Claude Code.
	// Event types: stop, prompt, working, start, notification, end

	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "notify stop event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "notify-session", "--state", "working")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "notify-session", "stop"},
			wantExitCode: 0,
		},
		{
			name: "notify prompt event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "prompt-session", "--state", "idle")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "prompt-session", "prompt"},
			wantExitCode: 0,
		},
		{
			name: "notify working event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "working-session", "--state", "idle")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "working-session", "working"},
			wantExitCode: 0,
		},
		{
			name: "notify start event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "start-session", "--state", "idle")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "start-session", "start"},
			wantExitCode: 0,
		},
		{
			name: "notify notification event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "notification-session", "--state", "working")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "notification-session", "notification"},
			wantExitCode: 0,
		},
		{
			name: "notify end event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "end-session", "--state", "working")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "end-session", "end"},
			wantExitCode: 0,
		},
		{
			name: "notify with execution-id",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "exec-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "exec-session", "stop", "--execution-id", "test-exec-123"},
			wantExitCode: 0,
		},
		{
			name:         "notify for non-existent session succeeds silently",
			args:         []string{"notify", "non-existent-session", "stop"},
			wantExitCode: 0, // Notify doesn't fail on missing sessions
		},
		{
			name:         "notify default event type is stop",
			args:         []string{"notify", "default-session"},
			wantExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := harness.NewTestEnvironment(t)

			if tt.setup != nil {
				tt.setup(t, env)
			}

			result := harness.RunCommand(t, env, tt.args...)

			if tt.wantExitCode == 0 {
				harness.AssertSuccess(t, result)
			} else {
				harness.AssertFailure(t, result)
			}

			if tt.validate != nil {
				tt.validate(t, env, result)
			}
		})
	}
}

func TestNotifyStateTransition(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Create a session in idle state
	harness.RunCommand(t, env, "sessions", "add", "state-session", "--state", "idle")

	// Verify initial state
	viewResult := harness.RunCommand(t, env, "sessions", "view", "state-session")
	harness.AssertSuccess(t, viewResult)
	harness.AssertStdoutContains(t, viewResult, "State: idle")

	// Send prompt event (should transition to working)
	notifyResult := harness.RunCommand(t, env, "notify", "state-session", "prompt")
	harness.AssertSuccess(t, notifyResult)

	// Verify state changed to working
	viewResult = harness.RunCommand(t, env, "sessions", "view", "state-session")
	harness.AssertSuccess(t, viewResult)
	harness.AssertStdoutContains(t, viewResult, "State: working")

	// Send stop event (should transition to idle - Claude finished working)
	notifyResult = harness.RunCommand(t, env, "notify", "state-session", "stop")
	harness.AssertSuccess(t, notifyResult)

	// Verify state changed to idle
	viewResult = harness.RunCommand(t, env, "sessions", "view", "state-session")
	harness.AssertSuccess(t, viewResult)
	harness.AssertStdoutContains(t, viewResult, "State: idle")
}
