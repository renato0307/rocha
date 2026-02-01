package integration_test

import (
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
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
			name: "notify handle stop event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "notify-session", "--state", "working")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "handle", "notify-session", "stop"},
			wantExitCode: 0,
		},
		{
			name: "notify handle prompt event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "prompt-session", "--state", "idle")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "handle", "prompt-session", "prompt"},
			wantExitCode: 0,
		},
		{
			name: "notify handle working event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "working-session", "--state", "idle")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "handle", "working-session", "working"},
			wantExitCode: 0,
		},
		{
			name: "notify handle start event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "start-session", "--state", "idle")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "handle", "start-session", "start"},
			wantExitCode: 0,
		},
		{
			name: "notify handle notification event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "notification-session", "--state", "working")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "handle", "notification-session", "notification"},
			wantExitCode: 0,
		},
		{
			name: "notify handle end event",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "end-session", "--state", "working")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "handle", "end-session", "end"},
			wantExitCode: 0,
		},
		{
			name: "notify handle with execution-id",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "exec-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"notify", "handle", "exec-session", "stop", "--execution-id", "test-exec-123"},
			wantExitCode: 0,
		},
		{
			name:         "notify handle for non-existent session succeeds silently",
			args:         []string{"notify", "handle", "non-existent-session", "stop"},
			wantExitCode: 0, // Notify doesn't fail on missing sessions
		},
		{
			name:         "notify handle default event type is stop",
			args:         []string{"notify", "handle", "default-session"},
			wantExitCode: 0,
		},
		{
			name:         "notify backward compat without handle subcommand",
			args:         []string{"notify", "compat-session", "stop"},
			wantExitCode: 0, // Should work due to default:"withargs"
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
	notifyResult := harness.RunCommand(t, env, "notify", "handle", "state-session", "prompt")
	harness.AssertSuccess(t, notifyResult)

	// Verify state changed to working
	viewResult = harness.RunCommand(t, env, "sessions", "view", "state-session")
	harness.AssertSuccess(t, viewResult)
	harness.AssertStdoutContains(t, viewResult, "State: working")

	// Send stop event (should transition to idle - Claude finished working)
	notifyResult = harness.RunCommand(t, env, "notify", "handle", "state-session", "stop")
	harness.AssertSuccess(t, notifyResult)

	// Verify state changed to idle
	viewResult = harness.RunCommand(t, env, "sessions", "view", "state-session")
	harness.AssertSuccess(t, viewResult)
	harness.AssertStdoutContains(t, viewResult, "State: idle")
}
