package integration_test

import (
	"testing"

	"rocha/test/integration/harness"
)

func TestSessionsAdd(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "add simple session",
			args:         []string{"sessions", "add", "test-session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Session 'test-session' added successfully")
			},
		},
		{
			name:         "add session with display name",
			args:         []string{"sessions", "add", "my-session", "--display-name", "My Session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Session 'my-session' added successfully")
			},
		},
		{
			name: "add session with all options",
			args: []string{
				"sessions", "add", "full-session",
				"--display-name", "Full Session",
				"--branch-name", "feature/test",
				"--repo-path", "/tmp/repo",
				"--state", "working",
			},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Session 'full-session' added successfully")
			},
		},
		{
			name: "add duplicate session fails",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				// Add the session first
				result := harness.RunCommand(t, env, "sessions", "add", "duplicate-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "add", "duplicate-session"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				// Should fail with duplicate error
				harness.AssertFailure(t, result)
			},
		},
		{
			name:         "add session with invalid state fails",
			args:         []string{"sessions", "add", "bad-state", "--state", "invalid"},
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

func TestSessionsAddAndVerify(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Add a session
	addResult := harness.RunCommand(t, env,
		"sessions", "add", "verify-session",
		"--display-name", "Verify Me",
		"--branch-name", "main",
		"--state", "idle",
	)
	harness.AssertSuccess(t, addResult)

	// Verify via list (JSON format for easier parsing)
	listResult := harness.RunCommand(t, env, "sessions", "list", "--format", "json")
	harness.AssertSuccess(t, listResult)
	harness.AssertStdoutContains(t, listResult, "verify-session")
	harness.AssertStdoutContains(t, listResult, "Verify Me")

	// Verify via view
	viewResult := harness.RunCommand(t, env, "sessions", "view", "verify-session")
	harness.AssertSuccess(t, viewResult)
	harness.AssertStdoutContains(t, viewResult, "Session: verify-session")
	harness.AssertStdoutContains(t, viewResult, "Display Name: Verify Me")
}
