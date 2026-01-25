package integration_test

import (
	"testing"

	"rocha/test/integration/harness"
)

func TestSessionsDuplicate(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "duplicate nonexistent session fails",
			args:         []string{"sessions", "duplicate", "nonexistent", "--new-name", "new-session"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name: "duplicate session without repo source fails",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				// Add a session without repo source
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "duplicate", "test-session", "--new-name", "new-session"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
				harness.AssertStderrContains(t, result, "has no repository source")
			},
		},
		{
			name:         "duplicate without new name fails",
			args:         []string{"sessions", "duplicate", "test-session"},
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
