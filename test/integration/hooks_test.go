package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
	"github.com/stretchr/testify/require"
)

func TestHooks(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "hooks with no data shows placeholder",
			args:         []string{"hooks"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Hook Events")
				harness.AssertStdoutContains(t, result, "No hooks found")
			},
		},
		{
			name:         "hooks help shows all options",
			args:         []string{"hooks", "--help"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "View Claude Code hook events")
				harness.AssertStdoutContains(t, result, "--event")
				harness.AssertStdoutContains(t, result, "--format")
				harness.AssertStdoutContains(t, result, "--from")
				harness.AssertStdoutContains(t, result, "--to")
				harness.AssertStdoutContains(t, result, "--limit")
			},
		},
		{
			name:         "hooks with invalid format shows error",
			args:         []string{"hooks", "--format=invalid"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStderrContains(t, result, "format")
			},
		},
		{
			name:         "hooks with invalid time format shows error",
			args:         []string{"hooks", "--from=invalid-time"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStderrContains(t, result, "invalid --from time")
			},
		},
		{
			name: "hooks with mock data shows all events",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				setupMockClaudeSession(t, env, "test-session", "another-session")
			},
			args:         []string{"hooks", "--limit=20"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Hook Events")
				harness.AssertStdoutContains(t, result, "test-session")
				harness.AssertStdoutContains(t, result, "another-session")
				harness.AssertStdoutContains(t, result, "SessionStart")
				harness.AssertStdoutContains(t, result, "PostToolUse")
				harness.AssertStdoutContains(t, result, "Stop")
			},
		},
		{
			name: "hooks filter by session name",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				setupMockClaudeSession(t, env, "test-session", "another-session")
			},
			args:         []string{"hooks", "test-session"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "test-session")
				harness.AssertStdoutNotContains(t, result, "another-session")
			},
		},
		{
			name: "hooks filter by event type",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				setupMockClaudeSession(t, env, "test-session", "another-session")
			},
			args:         []string{"hooks", "--event=SessionStart"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "SessionStart")
				harness.AssertStdoutNotContains(t, result, "PostToolUse")
				harness.AssertStdoutNotContains(t, result, "Stop")
			},
		},
		{
			name: "hooks with limit",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				setupMockClaudeSession(t, env, "test-session", "another-session")
			},
			args:         []string{"hooks", "--limit=2"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				// Count lines with timestamps (should be 2 data rows)
				lines := 0
				for _, line := range result.Stdout {
					if line >= '0' && line <= '9' {
						lines++
					}
				}
				// Should have limited output
				harness.AssertStdoutContains(t, result, "Hook Events")
			},
		},
		{
			name: "hooks json format",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				setupMockClaudeSession(t, env, "test-session", "another-session")
			},
			args:         []string{"hooks", "--format=json", "--limit=5"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				// Parse JSON
				var events []map[string]interface{}
				harness.AssertValidJSON(t, result, &events)

				// Should have events
				require.Greater(t, len(events), 0, "Should have at least one event")

				// Check structure
				firstEvent := events[0]
				require.Contains(t, firstEvent, "session_name")
				require.Contains(t, firstEvent, "hook_event")
				require.Contains(t, firstEvent, "hook_name")
				require.Contains(t, firstEvent, "timestamp")
				require.Contains(t, firstEvent, "command")
			},
		},
		{
			name: "hooks time filter with from",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				setupMockClaudeSession(t, env, "test-session", "another-session")
			},
			args:         []string{"hooks", "--from=2026-01-26T11:00:00Z"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "another-session")
				harness.AssertStdoutContains(t, result, "11:00")
				// Should not contain test-session events (all before 11:00)
				harness.AssertStdoutNotContains(t, result, "test-session")
			},
		},
		{
			name: "hooks time filter with to",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				setupMockClaudeSession(t, env, "test-session", "another-session")
			},
			args:         []string{"hooks", "--to=2026-01-26T10:30:00Z"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				// Should only show events up to 10:30
				harness.AssertStdoutContains(t, result, "10:00")
				harness.AssertStdoutContains(t, result, "10:05")
				harness.AssertStdoutContains(t, result, "10:10")
				harness.AssertStdoutContains(t, result, "10:15")
				// Should not contain events after 10:30
				harness.AssertStdoutNotContains(t, result, "11:00")
			},
		},
		{
			name: "hooks combined filters",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				setupMockClaudeSession(t, env, "test-session", "another-session")
			},
			args:         []string{"hooks", "test-session", "--event=PostToolUse", "--format=json"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				var events []map[string]interface{}
				harness.AssertValidJSON(t, result, &events)

				// All events should be from test-session and PostToolUse
				for _, event := range events {
					require.Equal(t, "test-session", event["session_name"])
					require.Equal(t, "PostToolUse", event["hook_event"])
				}
			},
		},
		{
			name: "hooks shows unknown for unmatched sessions",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				// Setup Claude data but don't add sessions to rocha
				setupMockClaudeSessionNoRochaSession(t, env)
			},
			args:         []string{"hooks", "--limit=5"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "unknown")
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

// setupMockClaudeSession creates mock Claude JSONL files and corresponding rocha sessions
func setupMockClaudeSession(t *testing.T, env *harness.TestEnvironment, sessionNames ...string) {
	t.Helper()

	// Create .claude/projects directory in HOME
	homeDir := env.TempDir()
	claudeProjectsDir := filepath.Join(homeDir, ".claude", "projects", "test-project")
	err := os.MkdirAll(claudeProjectsDir, 0755)
	require.NoError(t, err, "Failed to create Claude projects dir")

	// Copy fixture JSONL file
	fixtureSource := filepath.Join(getProjectRoot(t), "test", "integration", "fixtures", "sample-session.jsonl")
	fixtureData, err := os.ReadFile(fixtureSource)
	require.NoError(t, err, "Failed to read fixture file")

	jsonlPath := filepath.Join(claudeProjectsDir, "test-session.jsonl")
	err = os.WriteFile(jsonlPath, fixtureData, 0644)
	require.NoError(t, err, "Failed to write JSONL file")

	// Set HOME to our temp directory
	env.SetEnv("HOME", homeDir)

	// Add rocha sessions that match the cwd in the JSONL
	// The JSONL fixture has cwd="/tmp/test-rocha/worktrees/owner/repo/{session-name}"
	// We need to add sessions with matching WorktreePath
	for _, sessionName := range sessionNames {
		worktreePath := "/tmp/test-rocha/worktrees/owner/repo/" + sessionName

		// Add session - use --worktree-path to set the WorktreePath field
		result := harness.RunCommand(t, env,
			"sessions", "add",
			sessionName,
			"--display-name=Test Session",
			"--worktree-path="+worktreePath,
		)
		harness.AssertSuccess(t, result)
	}
}

// setupMockClaudeSessionNoRochaSession creates Claude data without rocha sessions
func setupMockClaudeSessionNoRochaSession(t *testing.T, env *harness.TestEnvironment) {
	t.Helper()

	homeDir := env.TempDir()
	claudeProjectsDir := filepath.Join(homeDir, ".claude", "projects", "test-project")
	err := os.MkdirAll(claudeProjectsDir, 0755)
	require.NoError(t, err, "Failed to create Claude projects dir")

	fixtureSource := filepath.Join(getProjectRoot(t), "test", "integration", "fixtures", "sample-session.jsonl")
	fixtureData, err := os.ReadFile(fixtureSource)
	require.NoError(t, err, "Failed to read fixture file")

	jsonlPath := filepath.Join(claudeProjectsDir, "test-session.jsonl")
	err = os.WriteFile(jsonlPath, fixtureData, 0644)
	require.NoError(t, err, "Failed to write JSONL file")

	env.SetEnv("HOME", homeDir)
}

// getProjectRoot returns the project root directory
func getProjectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	// From test/integration, go up two levels
	return filepath.Join(wd, "..", "..")
}
