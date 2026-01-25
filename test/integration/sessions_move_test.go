package integration_test

import (
	"path/filepath"
	"testing"

	"rocha/test/integration/harness"
)

func TestSessionsMove(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, sourceEnv, destEnv *harness.TestEnvironment)
		args         func(sourceEnv, destEnv *harness.TestEnvironment) []string
		wantExitCode int
		validate     func(t *testing.T, sourceEnv, destEnv *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name: "move sessions between homes",
			setup: func(t *testing.T, sourceEnv, destEnv *harness.TestEnvironment) {
				// Create session in source with repo info (owner/repo format)
				result := harness.RunCommand(t, sourceEnv, "sessions", "add", "test-session", "--repo-info", "owner/repo")
				harness.AssertSuccess(t, result)
			},
			args: func(sourceEnv, destEnv *harness.TestEnvironment) []string {
				return []string{
					"sessions", "move",
					"-f",
					"--from", sourceEnv.RochaHome,
					"--to", destEnv.RochaHome,
					"--repo", "owner/repo",
				}
			},
			wantExitCode: 0,
			validate: func(t *testing.T, sourceEnv, destEnv *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Moved repository")
			},
		},
		{
			name: "move non-existent repo fails",
			args: func(sourceEnv, destEnv *harness.TestEnvironment) []string {
				return []string{
					"sessions", "move",
					"-f",
					"--from", sourceEnv.RochaHome,
					"--to", destEnv.RochaHome,
					"--repo", "non/existent",
				}
			},
			wantExitCode: 1,
			validate: func(t *testing.T, sourceEnv, destEnv *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name: "invalid repo format fails",
			args: func(sourceEnv, destEnv *harness.TestEnvironment) []string {
				return []string{
					"sessions", "move",
					"-f",
					"--from", sourceEnv.RochaHome,
					"--to", destEnv.RochaHome,
					"--repo", "invalid-format",
				}
			},
			wantExitCode: 1,
			validate: func(t *testing.T, sourceEnv, destEnv *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name: "non-existent source path fails",
			args: func(sourceEnv, destEnv *harness.TestEnvironment) []string {
				return []string{
					"sessions", "move",
					"-f",
					"--from", filepath.Join(sourceEnv.RochaHome, "nonexistent"),
					"--to", destEnv.RochaHome,
					"--repo", "owner/repo",
				}
			},
			wantExitCode: 1,
			validate: func(t *testing.T, sourceEnv, destEnv *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create two separate environments for source and destination
			sourceEnv := harness.NewTestEnvironment(t)
			destEnv := harness.NewTestEnvironment(t)

			if tt.setup != nil {
				tt.setup(t, sourceEnv, destEnv)
			}

			args := tt.args(sourceEnv, destEnv)
			result := harness.RunCommand(t, sourceEnv, args...)

			if tt.wantExitCode == 0 {
				harness.AssertSuccess(t, result)
			} else {
				harness.AssertFailure(t, result)
			}

			if tt.validate != nil {
				tt.validate(t, sourceEnv, destEnv, result)
			}
		})
	}
}
