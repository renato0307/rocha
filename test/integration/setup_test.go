package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSetup(t *testing.T) {
	// Note: Setup command modifies shell config files (.bashrc, .zshrc, .tmux.conf)
	// In Docker container, this is safe. The test environment uses temp HOME.
	// Claude CLI is installed in the Docker image.

	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment, homeDir string)
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, homeDir string, result harness.CommandResult)
	}{
		{
			name: "setup creates tmux config",
			setup: func(t *testing.T, env *harness.TestEnvironment, homeDir string) {
				// Create empty .bashrc so setup has something to work with
				bashrcPath := filepath.Join(homeDir, ".bashrc")
				if err := os.WriteFile(bashrcPath, []byte("# Empty bashrc\n"), 0644); err != nil {
					t.Fatalf("Failed to create .bashrc: %v", err)
				}
			},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, homeDir string, result harness.CommandResult) {
				// Check that setup found dependencies
				harness.AssertStdoutContains(t, result, "tmux found")
				harness.AssertStdoutContains(t, result, "git found")
				harness.AssertStdoutContains(t, result, "Claude Code CLI found")

				// Verify .tmux.conf was created/modified
				tmuxConfPath := filepath.Join(homeDir, ".tmux.conf")
				content, err := os.ReadFile(tmuxConfPath)
				if err != nil {
					t.Fatalf("Failed to read .tmux.conf: %v", err)
				}

				// Should contain rocha status bar configuration
				if !strings.Contains(string(content), "rocha status") {
					t.Error("Expected .tmux.conf to contain 'rocha status'")
				}
			},
		},
		{
			name: "setup is idempotent",
			setup: func(t *testing.T, env *harness.TestEnvironment, homeDir string) {
				// Create .bashrc with existing rocha PATH
				bashrcPath := filepath.Join(homeDir, ".bashrc")
				content := "# Empty bashrc\nexport PATH=\"/some/path:$PATH\" # Added by rocha setup\n"
				if err := os.WriteFile(bashrcPath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create .bashrc: %v", err)
				}

				// Create .tmux.conf with existing rocha config
				tmuxConfPath := filepath.Join(homeDir, ".tmux.conf")
				tmuxContent := `# Existing config
# Rocha status bar configuration
set -g status-left-length 50
set -g status-right "Claude: #(rocha status) | %H:%M"
set -g status-interval 1
set -g mouse on
`
				if err := os.WriteFile(tmuxConfPath, []byte(tmuxContent), 0644); err != nil {
					t.Fatalf("Failed to create .tmux.conf: %v", err)
				}
			},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, homeDir string, result harness.CommandResult) {
				// Should indicate things are already configured
				harness.AssertStdoutContains(t, result, "already")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := harness.NewTestEnvironment(t)

			// Create a temporary home directory for this test
			homeDir := t.TempDir()

			// Setup files if needed
			if tt.setup != nil {
				tt.setup(t, env, homeDir)
			}

			// Build environment with custom HOME
			customEnv := make([]string, 0)
			for _, kv := range env.Environ() {
				if !strings.HasPrefix(kv, "HOME=") {
					customEnv = append(customEnv, kv)
				}
			}
			customEnv = append(customEnv, "HOME="+homeDir)

			// Run setup command with custom environment
			result := harness.RunCommandWithEnv(t, customEnv, "setup")

			if tt.wantExitCode == 0 {
				harness.AssertSuccess(t, result)
			} else {
				harness.AssertFailure(t, result)
			}

			if tt.validate != nil {
				tt.validate(t, env, homeDir, result)
			}
		})
	}
}
