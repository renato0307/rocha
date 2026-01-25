package integration_test

import (
	"testing"

	"rocha/test/integration/harness"
)

func TestSessionsFlag(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "flag session",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "flag", "test-session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Session 'test-session' flagged")

				// Verify via view
				viewResult := harness.RunCommand(t, env, "sessions", "view", "test-session")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Flagged: true")
			},
		},
		{
			name: "unflag session",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
				// Flag first
				result = harness.RunCommand(t, env, "sessions", "flag", "test-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "flag", "test-session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Session 'test-session' unflagged")

				// Verify via view
				viewResult := harness.RunCommand(t, env, "sessions", "view", "test-session")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Flagged: false")
			},
		},
		{
			name:         "flag nonexistent session fails",
			args:         []string{"sessions", "flag", "nonexistent"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
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
