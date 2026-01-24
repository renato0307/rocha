package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnvironment provides an isolated test environment with its own ROCHA_HOME.
type TestEnvironment struct {
	RochaHome string
	tb        testing.TB
}

// NewTestEnvironment creates an isolated test environment with a temp ROCHA_HOME.
// The temp directory is automatically cleaned up when the test completes.
func NewTestEnvironment(tb testing.TB) *TestEnvironment {
	tb.Helper()

	rochaHome := tb.TempDir()

	// Create required subdirectories
	if err := os.MkdirAll(filepath.Join(rochaHome, "worktrees"), 0755); err != nil {
		tb.Fatalf("Failed to create worktrees directory: %v", err)
	}

	return &TestEnvironment{
		RochaHome: rochaHome,
		tb:        tb,
	}
}

// Environ returns environment variables configured for test isolation.
// It filters out ROCHA_* variables and sets:
//   - ROCHA_HOME to the temp directory
//   - ROCHA_DEBUG to empty string (disables debug logging)
//   - ROCHA_EDITOR to "true" (no-op command)
func (e *TestEnvironment) Environ() []string {
	env := make([]string, 0, len(os.Environ())+3)

	// Filter out existing ROCHA_* variables
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "ROCHA_") {
			env = append(env, kv)
		}
	}

	// Add isolated environment variables
	env = append(env,
		"ROCHA_HOME="+e.RochaHome,
		"ROCHA_DEBUG=",
		"ROCHA_EDITOR=true",
	)

	return env
}

// DBPath returns the path to the test database.
func (e *TestEnvironment) DBPath() string {
	return filepath.Join(e.RochaHome, "state.db")
}

// WorktreesPath returns the path to the worktrees directory.
func (e *TestEnvironment) WorktreesPath() string {
	return filepath.Join(e.RochaHome, "worktrees")
}
