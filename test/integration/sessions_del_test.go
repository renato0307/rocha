package integration_test

import (
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSessionsDel(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "delete existing session with force",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "to-delete")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "del", "-f", "to-delete"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "deleted successfully")

				// Verify session no longer exists
				listResult := harness.RunCommand(t, env, "sessions", "list", "--format", "json")
				harness.AssertSuccess(t, listResult)
				harness.AssertStdoutNotContains(t, listResult, "to-delete")
			},
		},
		{
			name:         "delete non-existent session fails",
			args:         []string{"sessions", "del", "-f", "non-existent"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name: "delete with skip tmux flag",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "skip-tmux-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "del", "-f", "-k", "skip-tmux-session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "deleted successfully")
			},
		},
		{
			name: "delete with skip worktree flag",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "skip-worktree-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "del", "-f", "-w", "skip-worktree-session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "deleted successfully")
			},
		},
		{
			name: "delete with all skip flags",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "skip-all-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "del", "-f", "-k", "-w", "skip-all-session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "deleted successfully")
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

func TestSessionsDelAndVerify(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Add multiple sessions
	harness.RunCommand(t, env, "sessions", "add", "session-1")
	harness.RunCommand(t, env, "sessions", "add", "session-2")
	harness.RunCommand(t, env, "sessions", "add", "session-3")

	// Verify all exist
	listResult := harness.RunCommand(t, env, "sessions", "list", "--format", "json")
	harness.AssertSuccess(t, listResult)
	harness.AssertStdoutContains(t, listResult, "session-1")
	harness.AssertStdoutContains(t, listResult, "session-2")
	harness.AssertStdoutContains(t, listResult, "session-3")

	// Delete one
	delResult := harness.RunCommand(t, env, "sessions", "del", "-f", "session-2")
	harness.AssertSuccess(t, delResult)

	// Verify only session-2 is gone
	listResult = harness.RunCommand(t, env, "sessions", "list", "--format", "json")
	harness.AssertSuccess(t, listResult)
	harness.AssertStdoutContains(t, listResult, "session-1")
	harness.AssertStdoutNotContains(t, listResult, "session-2")
	harness.AssertStdoutContains(t, listResult, "session-3")
}
