package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEnvironment provides an isolated test environment with its own ROCHA_HOME.
type TestEnvironment struct {
	RochaHome string
	extraEnv  map[string]string
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
		extraEnv:  make(map[string]string),
		tb:        tb,
	}
}

// Environ returns environment variables configured for test isolation.
// It filters out ROCHA_* variables and sets:
//   - ROCHA_HOME to the temp directory
//   - ROCHA_DEBUG to empty string (disables debug logging)
//   - ROCHA_EDITOR to "true" (no-op command)
func (e *TestEnvironment) Environ() []string {
	env := make([]string, 0, len(os.Environ())+3+len(e.extraEnv))

	// Build a set of keys we want to override
	overrideKeys := make(map[string]bool)
	overrideKeys["ROCHA_HOME"] = true
	overrideKeys["ROCHA_DEBUG"] = true
	overrideKeys["ROCHA_EDITOR"] = true
	for k := range e.extraEnv {
		overrideKeys[k] = true
	}

	// Filter out existing ROCHA_* variables and any we're overriding
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		key := parts[0]
		if strings.HasPrefix(key, "ROCHA_") || overrideKeys[key] {
			continue
		}
		env = append(env, kv)
	}

	// Add isolated environment variables
	env = append(env,
		"ROCHA_HOME="+e.RochaHome,
		"ROCHA_DEBUG=",
		"ROCHA_EDITOR=true",
	)

	// Add extra environment variables
	for k, v := range e.extraEnv {
		env = append(env, k+"="+v)
	}

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

// TempDir returns the root temp directory used for this test environment.
func (e *TestEnvironment) TempDir() string {
	return filepath.Dir(e.RochaHome)
}

// SetEnv sets an additional environment variable for this test environment.
func (e *TestEnvironment) SetEnv(key, value string) {
	if e.extraEnv == nil {
		e.extraEnv = make(map[string]string)
	}
	e.extraEnv[key] = value
}

// TodayTimestamp returns a timestamp string for today in RFC3339 format.
// This is useful for creating test JSONL files with "today's" data.
func TodayTimestamp() string {
	now := time.Now()
	// Set to middle of the current hour to ensure it's captured in hourly stats
	t := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 30, 0, 0, time.UTC)
	return t.Format(time.RFC3339)
}
