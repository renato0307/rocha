package integration_test

import (
	"testing"

	"rocha/test/integration/harness"
)

func TestSessionsComment(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "add comment",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "comment", "test-session", "--comment", "My test comment"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Comment updated for session 'test-session'")

				// Verify via view (JSON format to see comment)
				viewResult := harness.RunCommand(t, env, "sessions", "view", "test-session", "--format", "json")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "My test comment")
			},
		},
		{
			name: "clear comment",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "comment", "test-session", "--comment", "Initial comment")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "comment", "test-session", "--comment", ""},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Comment cleared for session 'test-session'")
			},
		},
		{
			name:         "comment nonexistent session fails",
			args:         []string{"sessions", "comment", "nonexistent", "--comment", "Test"},
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
