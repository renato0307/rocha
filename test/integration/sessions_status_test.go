package integration_test

import (
	"testing"

	"rocha/test/integration/harness"
)

func TestSessionsStatus(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "set status",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "status", "test-session", "--status", "implement"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Status set to 'implement' for session 'test-session'")
			},
		},
		{
			name: "clear status with empty string",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "status", "test-session", "--status", "implement")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "status", "test-session", "--status", ""},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Status cleared for session 'test-session'")
			},
		},
		{
			name: "clear status with keyword",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "status", "test-session", "--status", "implement")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "status", "test-session", "--status", "clear"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Status cleared for session 'test-session'")
			},
		},
		{
			name:         "list statuses",
			args:         []string{"sessions", "status", "--list"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Available statuses:")
				harness.AssertStdoutContains(t, result, "spec")
				harness.AssertStdoutContains(t, result, "plan")
				harness.AssertStdoutContains(t, result, "implement")
				harness.AssertStdoutContains(t, result, "review")
				harness.AssertStdoutContains(t, result, "done")
			},
		},
		{
			name:         "status nonexistent session fails",
			args:         []string{"sessions", "status", "nonexistent", "--status", "implement"},
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
