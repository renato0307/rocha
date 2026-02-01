package integration_test

import (
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSessionsSet(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "set claudedir for single session",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "test-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "set", "test-session", "--variable", "claudedir", "--value", "/path/to/claude"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Updated")
			},
		},
		{
			name: "set allow-dangerously-skip-permissions true",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "dangerous-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "set", "dangerous-session", "--variable", "allow-dangerously-skip-permissions", "--value", "true"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Updated")
			},
		},
		{
			name: "set allow-dangerously-skip-permissions false",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "safe-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "set", "safe-session", "--variable", "allow-dangerously-skip-permissions", "--value", "false"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Updated")
			},
		},
		{
			name: "set with yes/no values",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "yesno-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "set", "yesno-session", "--variable", "allow-dangerously-skip-permissions", "--value", "yes"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Updated")
			},
		},
		{
			name: "set debug-claude true",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "debug-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "set", "debug-session", "--variable", "debug-claude", "--value", "true"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Updated")

				// Verify the flag was set by viewing the session
				viewResult := harness.RunCommand(t, env, "sessions", "view", "debug-session")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")
			},
		},
		{
			name: "set debug-claude false",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "no-debug-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "set", "no-debug-session", "--variable", "debug-claude", "--value", "false"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Updated")

				// Verify the flag was set to false
				viewResult := harness.RunCommand(t, env, "sessions", "view", "no-debug-session")
				harness.AssertSuccess(t, viewResult)
				harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")
			},
		},
		{
			name:         "set on non-existent session fails",
			args:         []string{"sessions", "set", "non-existent", "--variable", "claudedir", "--value", "/path"},
			wantExitCode: 0, // The command doesn't fail but prints failure message
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Failed to update")
			},
		},
		{
			name: "set invalid boolean value fails",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "invalid-bool-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "set", "invalid-bool-session", "--variable", "allow-dangerously-skip-permissions", "--value", "invalid"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name:         "set without name or --all fails",
			args:         []string{"sessions", "set", "--variable", "claudedir", "--value", "/path"},
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

func TestSessionsSetAll(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Create multiple sessions
	harness.RunCommand(t, env, "sessions", "add", "session-1")
	harness.RunCommand(t, env, "sessions", "add", "session-2")
	harness.RunCommand(t, env, "sessions", "add", "session-3")

	// Set claudedir for all sessions
	result := harness.RunCommand(t, env, "sessions", "set", "--all", "--variable", "claudedir", "--value", "/shared/claude")
	harness.AssertSuccess(t, result)

	// Should show updating all sessions
	harness.AssertStdoutContains(t, result, "Updating claudedir for 3 sessions")
	harness.AssertStdoutContains(t, result, "Updated 'session-1'")
	harness.AssertStdoutContains(t, result, "Updated 'session-2'")
	harness.AssertStdoutContains(t, result, "Updated 'session-3'")
	harness.AssertStdoutContains(t, result, "Updated claudedir for 3 session(s)")
}

func TestSessionsSetWithBothNameAndAllFails(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	harness.RunCommand(t, env, "sessions", "add", "test-session")

	// Try to use both name and --all
	result := harness.RunCommand(t, env, "sessions", "set", "test-session", "--all", "--variable", "claudedir", "--value", "/path")
	harness.AssertFailure(t, result)
}
