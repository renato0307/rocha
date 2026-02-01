package integration_test

import (
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
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
		{
			name: "duplicate preserves debug-claude flag",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				// Create a local git repository
				gitSetup := harness.NewTestGitSetup(t)
				repoURL := "file://" + gitSetup.GetBareRepoPath()

				// Add a session with repo source and debug-claude enabled
				result := harness.RunCommand(t, env, "sessions", "add", "source-session",
					"--repo-source", repoURL,
					"--debug-claude")
				harness.AssertSuccess(t, result)

				// Verify source has debug enabled
				viewResult := harness.RunCommand(t, env, "sessions", "view", "source-session")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")
			},
			args:         []string{"sessions", "duplicate", "source-session", "--new-name", "duplicated-session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertSuccess(t, result)
				harness.AssertStdoutContains(t, result, "Session 'duplicated-session' created")

				// Verify duplicated session has debug-claude enabled
				viewResult := harness.RunCommand(t, env, "sessions", "view", "duplicated-session")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")
			},
		},
		{
			name: "duplicate preserves skip-permissions and debug-claude independently",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				// Create a local git repository
				gitSetup := harness.NewTestGitSetup(t)
				repoURL := "file://" + gitSetup.GetBareRepoPath()

				// Add a session with repo source, skip-permissions enabled, debug-claude disabled
				result := harness.RunCommand(t, env, "sessions", "add", "mixed-flags-session",
					"--repo-source", repoURL,
					"--allow-dangerously-skip-permissions")
				harness.AssertSuccess(t, result)

				// Verify flags
				viewResult := harness.RunCommand(t, env, "sessions", "view", "mixed-flags-session")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Allow Dangerously Skip Permissions: true")
				harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")
			},
			args:         []string{"sessions", "duplicate", "mixed-flags-session", "--new-name", "mixed-dup"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertSuccess(t, result)

				// Verify duplicated session preserves both flags correctly
				viewResult := harness.RunCommand(t, env, "sessions", "view", "mixed-dup")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Allow Dangerously Skip Permissions: true")
				harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")
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
