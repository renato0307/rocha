package integration_test

import (
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSessionsArchive(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "archive existing session with force",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "to-archive")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "archive", "-f", "-s", "to-archive"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "archived successfully")

				// Verify session is archived (appears in --show-archived list)
				listResult := harness.RunCommand(t, env, "sessions", "list", "--show-archived", "--format", "json")
				harness.AssertSuccess(t, listResult)
				harness.AssertStdoutContains(t, listResult, "to-archive")
			},
		},
		{
			name:         "archive non-existent session fails",
			args:         []string{"sessions", "archive", "-f", "-s", "non-existent"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name: "archive with remove worktree flag",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "archive-rm-wt")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "archive", "-f", "-w", "archive-rm-wt"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "archived successfully")
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

func TestSessionsArchiveToggle(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Add a session
	addResult := harness.RunCommand(t, env, "sessions", "add", "toggle-session")
	harness.AssertSuccess(t, addResult)

	// Verify session is NOT archived initially
	listResult := harness.RunCommand(t, env, "sessions", "list", "--format", "json")
	harness.AssertSuccess(t, listResult)
	harness.AssertStdoutContains(t, listResult, "toggle-session")

	// Archive the session
	archiveResult := harness.RunCommand(t, env, "sessions", "archive", "-f", "-s", "toggle-session")
	harness.AssertSuccess(t, archiveResult)
	harness.AssertStdoutContains(t, archiveResult, "archived successfully")

	// Verify session is now archived (not in regular list)
	listResult = harness.RunCommand(t, env, "sessions", "list", "--format", "json")
	harness.AssertSuccess(t, listResult)
	harness.AssertStdoutNotContains(t, listResult, "toggle-session")

	// Verify session appears in archived list
	archivedListResult := harness.RunCommand(t, env, "sessions", "list", "--show-archived", "--format", "json")
	harness.AssertSuccess(t, archivedListResult)
	harness.AssertStdoutContains(t, archivedListResult, "toggle-session")

	// Unarchive the session (toggle)
	unarchiveResult := harness.RunCommand(t, env, "sessions", "archive", "toggle-session")
	harness.AssertSuccess(t, unarchiveResult)
	harness.AssertStdoutContains(t, unarchiveResult, "unarchived successfully")

	// Verify session is back in regular list
	listResult = harness.RunCommand(t, env, "sessions", "list", "--format", "json")
	harness.AssertSuccess(t, listResult)
	harness.AssertStdoutContains(t, listResult, "toggle-session")
}
