package harness

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertSuccess verifies the command succeeded with exit code 0.
func AssertSuccess(tb testing.TB, result CommandResult) {
	tb.Helper()
	assert.Equal(tb, 0, result.ExitCode,
		"Expected success (exit 0), got %d.\nStdout: %s\nStderr: %s",
		result.ExitCode, result.Stdout, result.Stderr)
}

// AssertFailure verifies the command failed with non-zero exit code.
func AssertFailure(tb testing.TB, result CommandResult) {
	tb.Helper()
	assert.NotEqual(tb, 0, result.ExitCode,
		"Expected failure (non-zero exit), got success.\nStdout: %s",
		result.Stdout)
}

// AssertExitCode verifies the command exited with a specific code.
func AssertExitCode(tb testing.TB, result CommandResult, expected int) {
	tb.Helper()
	assert.Equal(tb, expected, result.ExitCode,
		"Expected exit code %d, got %d.\nStdout: %s\nStderr: %s",
		expected, result.ExitCode, result.Stdout, result.Stderr)
}

// AssertStdoutContains verifies stdout contains the expected string.
func AssertStdoutContains(tb testing.TB, result CommandResult, expected string) {
	tb.Helper()
	assert.Contains(tb, result.Stdout, expected,
		"Expected stdout to contain %q.\nActual stdout: %s",
		expected, result.Stdout)
}

// AssertStdoutNotContains verifies stdout does not contain the string.
func AssertStdoutNotContains(tb testing.TB, result CommandResult, unexpected string) {
	tb.Helper()
	assert.NotContains(tb, result.Stdout, unexpected,
		"Expected stdout NOT to contain %q.\nActual stdout: %s",
		unexpected, result.Stdout)
}

// AssertStderrContains verifies stderr contains the expected string.
func AssertStderrContains(tb testing.TB, result CommandResult, expected string) {
	tb.Helper()
	assert.Contains(tb, result.Stderr, expected,
		"Expected stderr to contain %q.\nActual stderr: %s",
		expected, result.Stderr)
}

// AssertStdoutEmpty verifies stdout is empty.
func AssertStdoutEmpty(tb testing.TB, result CommandResult) {
	tb.Helper()
	assert.Empty(tb, strings.TrimSpace(result.Stdout),
		"Expected empty stdout, got: %s", result.Stdout)
}

// AssertStderrEmpty verifies stderr is empty.
func AssertStderrEmpty(tb testing.TB, result CommandResult) {
	tb.Helper()
	assert.Empty(tb, strings.TrimSpace(result.Stderr),
		"Expected empty stderr, got: %s", result.Stderr)
}

// AssertValidJSON verifies stdout is valid JSON and unmarshals it into target.
func AssertValidJSON(tb testing.TB, result CommandResult, target any) {
	tb.Helper()
	err := json.Unmarshal([]byte(result.Stdout), target)
	require.NoError(tb, err, "Expected valid JSON.\nStdout: %s", result.Stdout)
}

// AssertJSONContains verifies stdout is valid JSON and contains the expected key-value.
func AssertJSONContains(tb testing.TB, result CommandResult, key string, expected any) {
	tb.Helper()
	var data map[string]any
	err := json.Unmarshal([]byte(result.Stdout), &data)
	require.NoError(tb, err, "Expected valid JSON.\nStdout: %s", result.Stdout)
	assert.Equal(tb, expected, data[key], "JSON key %q mismatch", key)
}
