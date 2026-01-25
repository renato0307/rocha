package integration_test

import (
	"testing"

	"rocha/test/integration/harness"
)

func TestSessionsRename(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "rename display name",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "rename", "test-session", "--display-name", "New Display Name"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Session 'test-session' display name updated to 'New Display Name'")

				// Verify via view
				viewResult := harness.RunCommand(t, env, "sessions", "view", "test-session")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Display Name: New Display Name")
			},
		},
		{
			name:         "rename nonexistent session fails",
			args:         []string{"sessions", "rename", "nonexistent", "--display-name", "New Name"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name:         "rename without display name fails",
			args:         []string{"sessions", "rename", "test-session"},
			wantExitCode: 1,
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
