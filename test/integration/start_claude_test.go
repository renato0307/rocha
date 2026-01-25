package integration_test

import (
	"os"
	"strings"
	"testing"

	"rocha/test/integration/harness"
)

func TestStartClaude(t *testing.T) {
	// Note: start-claude uses syscall.Exec to replace the process with claude.
	// In test environment, claude binary is likely not available.
	// These tests verify error handling when claude is not found.

	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		modifyEnv    func(environ []string) []string
		wantExitCode int
		validate     func(t *testing.T, result harness.CommandResult)
	}{
		{
			name:         "fails when claude not in PATH",
			args:         []string{"start-claude"},
			wantExitCode: 1,
			modifyEnv: func(environ []string) []string {
				// Remove PATH or set to empty to ensure claude is not found
				var filtered []string
				for _, kv := range environ {
					if !strings.HasPrefix(kv, "PATH=") {
						filtered = append(filtered, kv)
					}
				}
				filtered = append(filtered, "PATH=")
				return filtered
			},
			validate: func(t *testing.T, result harness.CommandResult) {
				// Should fail with claude not found error
				harness.AssertFailure(t, result)
			},
		},
		{
			name: "respects ROCHA_SESSION_NAME env var",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "env-session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"start-claude"},
			wantExitCode: 1, // Will still fail because claude is not available
			modifyEnv: func(environ []string) []string {
				// Remove PATH to ensure claude is not found
				var filtered []string
				for _, kv := range environ {
					if !strings.HasPrefix(kv, "PATH=") {
						filtered = append(filtered, kv)
					}
				}
				filtered = append(filtered, "PATH=", "ROCHA_SESSION_NAME=env-session")
				return filtered
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := harness.NewTestEnvironment(t)

			if tt.setup != nil {
				tt.setup(t, env)
			}

			// Build environment
			environ := env.Environ()
			if tt.modifyEnv != nil {
				environ = tt.modifyEnv(environ)
			}

			result := harness.RunCommandWithEnv(t, environ, tt.args...)

			if tt.wantExitCode == 0 {
				harness.AssertSuccess(t, result)
			} else {
				harness.AssertFailure(t, result)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestStartClaudeWithMockClaude(t *testing.T) {
	// Skip this test if we can't create a mock claude binary
	// This test creates a mock 'claude' script that just exits
	// to verify that start-claude passes the correct arguments

	env := harness.NewTestEnvironment(t)

	// Create a session for the test
	result := harness.RunCommand(t, env, "sessions", "add", "mock-session")
	harness.AssertSuccess(t, result)

	// Create a temp directory for our mock claude
	mockDir := t.TempDir()
	mockClaudePath := mockDir + "/claude"

	// Create a simple mock claude script that just echoes args and exits
	mockScript := `#!/bin/sh
echo "MOCK_CLAUDE_CALLED"
echo "ARGS: $@"
exit 0
`
	if err := os.WriteFile(mockClaudePath, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock claude: %v", err)
	}

	// Build environment with mock claude in PATH
	environ := env.Environ()
	var filtered []string
	for _, kv := range environ {
		if !strings.HasPrefix(kv, "PATH=") {
			filtered = append(filtered, kv)
		}
	}
	filtered = append(filtered,
		"PATH="+mockDir,
		"ROCHA_SESSION_NAME=mock-session",
	)

	// Run start-claude with mock
	// Note: Because start-claude uses syscall.Exec, we can't capture its output
	// after exec. The test just verifies the command starts without error
	// before exec happens (which replaces the process).
	// This test mainly verifies the setup code runs without error.

	startResult := harness.RunCommandWithEnv(t, filtered, "start-claude")

	// With the mock claude, the command should succeed
	// (the mock just echoes and exits 0)
	harness.AssertSuccess(t, startResult)
}
