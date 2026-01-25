package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestStats(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "stats with no data shows placeholder",
			args:         []string{"stats"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Token Usage")
				harness.AssertStdoutContains(t, result, "No token data yet")
			},
		},
		{
			name:         "stats table format with no data",
			args:         []string{"stats", "--format=table"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Token Usage")
				harness.AssertStdoutContains(t, result, "No token data yet")
			},
		},
		{
			name:         "stats chart format with no data",
			args:         []string{"stats", "--format=chart"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Token Usage")
				// When no data, chart format also shows placeholder
				harness.AssertStdoutContains(t, result, "No token data yet")
			},
		},
		{
			name:         "stats with invalid format shows error",
			args:         []string{"stats", "--format=invalid"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStderrContains(t, result, "format")
			},
		},
		{
			name:         "stats help shows format options",
			args:         []string{"stats", "--help"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "format")
				harness.AssertStdoutContains(t, result, "table")
				harness.AssertStdoutContains(t, result, "chart")
			},
		},
		{
			name: "stats with mock JSONL data",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				// Create mock Claude projects directory with test JSONL data
				homeDir := env.TempDir()
				claudeProjectsDir := filepath.Join(homeDir, ".claude", "projects", "test-project")
				err := os.MkdirAll(claudeProjectsDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create mock Claude projects dir: %v", err)
				}

				// Create a test JSONL file with token usage data
				// Note: This uses today's timestamp so it will be picked up
				jsonlPath := filepath.Join(claudeProjectsDir, "test-session.jsonl")
				jsonlContent := `{"type":"assistant","timestamp":"` + harness.TodayTimestamp() + `","message":{"usage":{"input_tokens":1000,"output_tokens":500,"cache_creation_input_tokens":100,"cache_read_input_tokens":50}}}`
				err = os.WriteFile(jsonlPath, []byte(jsonlContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create mock JSONL file: %v", err)
				}

				// Set HOME to our temp directory so the parser finds our mock data
				env.SetEnv("HOME", homeDir)
			},
			args:         []string{"stats"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Token Usage")
				// Should show table with data
				harness.AssertStdoutContains(t, result, "Hour")
				harness.AssertStdoutContains(t, result, "Input")
				harness.AssertStdoutContains(t, result, "Output")
				harness.AssertStdoutContains(t, result, "Total")
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
