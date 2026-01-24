package harness

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const defaultTimeout = 30 * time.Second

var (
	binaryPath string
	buildOnce  sync.Once
	buildErr   error
)

// CommandResult holds the result of running a CLI command
type CommandResult struct {
	ExitCode int
	Stderr   string
	Stdout   string
}

// BuildBinary compiles the rocha binary once per test run.
// Call this from TestMain before running tests.
func BuildBinary() (string, error) {
	buildOnce.Do(func() {
		tempDir, err := os.MkdirTemp("", "rocha-integration-test-*")
		if err != nil {
			buildErr = err
			return
		}

		binaryPath = filepath.Join(tempDir, "rocha")

		projectRoot, err := findProjectRoot()
		if err != nil {
			buildErr = err
			return
		}

		cmd := exec.Command("go", "build", "-o", binaryPath, ".")
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		buildErr = cmd.Run()
	})

	return binaryPath, buildErr
}

// CleanupBinary removes the compiled binary and its temp directory.
// Call this from TestMain after tests complete.
func CleanupBinary() {
	if binaryPath != "" {
		if err := os.RemoveAll(filepath.Dir(binaryPath)); err != nil {
			log.Printf("Warning: failed to cleanup binary directory: %v", err)
		}
	}
}

// GetBinaryPath returns the path to the compiled binary.
func GetBinaryPath() string {
	return binaryPath
}

// RunCommand executes the rocha binary with given arguments using default timeout.
func RunCommand(tb testing.TB, env *TestEnvironment, args ...string) CommandResult {
	tb.Helper()
	return RunCommandWithTimeout(tb, env, defaultTimeout, args...)
}

// RunCommandWithTimeout executes the rocha binary with given arguments and timeout.
func RunCommandWithTimeout(tb testing.TB, env *TestEnvironment, timeout time.Duration, args ...string) CommandResult {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = env.Environ()

	err := cmd.Run()

	exitCode := 0
	if ctx.Err() == context.DeadlineExceeded {
		tb.Logf("Command timed out after %v: %v %v", timeout, binaryPath, args)
		exitCode = -1
	} else if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		tb.Logf("Command execution error: %v", err)
		exitCode = -1
	}

	return CommandResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
}

// findProjectRoot uses go list to find the module root directory.
func findProjectRoot() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
