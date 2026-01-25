package integration_test

import (
	"testing"

	"rocha/test/integration/harness"
)

func TestStatus(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		wantExitCode int
		validate     func(t *testing.T, result harness.CommandResult)
	}{
		{
			name:         "no sessions shows zeros",
			wantExitCode: 0,
			validate: func(t *testing.T, result harness.CommandResult) {
				// Status should contain state symbols with counts
				// When no sessions exist, it shows zeros
				harness.AssertStdoutContains(t, result, ":0")
			},
		},
		{
			name: "with idle session",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				// Add a session with idle state
				result := harness.RunCommand(t, env, "sessions", "add", "test-session", "--state", "idle")
				harness.AssertSuccess(t, result)
			},
			wantExitCode: 0,
			validate: func(t *testing.T, result harness.CommandResult) {
				// Should show at least one idle session
				harness.AssertStdoutContains(t, result, ":1")
			},
		},
		{
			name: "with working session",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "working-session", "--state", "working")
				harness.AssertSuccess(t, result)
			},
			wantExitCode: 0,
			validate: func(t *testing.T, result harness.CommandResult) {
				// Should show at least one working session
				harness.AssertStdoutContains(t, result, ":1")
			},
		},
		{
			name: "with multiple sessions",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				// Add multiple sessions with different states
				harness.RunCommand(t, env, "sessions", "add", "idle-1", "--state", "idle")
				harness.RunCommand(t, env, "sessions", "add", "idle-2", "--state", "idle")
				harness.RunCommand(t, env, "sessions", "add", "working-1", "--state", "working")
			},
			wantExitCode: 0,
			validate: func(t *testing.T, result harness.CommandResult) {
				// Should show counts for the sessions
				// We have 2 idle and 1 working
				harness.AssertStdoutContains(t, result, ":2") // idle count
				harness.AssertStdoutContains(t, result, ":1") // working count
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := harness.NewTestEnvironment(t)

			if tt.setup != nil {
				tt.setup(t, env)
			}

			result := harness.RunCommand(t, env, "status")

			if tt.wantExitCode == 0 {
				harness.AssertSuccess(t, result)
			} else {
				harness.AssertFailure(t, result)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
