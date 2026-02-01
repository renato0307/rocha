package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestDebugClaudeEndToEnd(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Test 1: Create session with --debug-claude flag
	t.Run("create with flag", func(t *testing.T) {
		result := harness.RunCommand(t, env, "sessions", "add", "debug-test", "--debug-claude")
		harness.AssertSuccess(t, result)
		harness.AssertStdoutContains(t, result, "Session 'debug-test' added successfully")

		// Verify flag was saved
		viewResult := harness.RunCommand(t, env, "sessions", "view", "debug-test")
		harness.AssertSuccess(t, viewResult)
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")
	})

	// Test 2: Create session without flag (should default to false)
	t.Run("create without flag", func(t *testing.T) {
		result := harness.RunCommand(t, env, "sessions", "add", "no-debug-test")
		harness.AssertSuccess(t, result)

		// Verify flag defaults to false
		viewResult := harness.RunCommand(t, env, "sessions", "view", "no-debug-test")
		harness.AssertSuccess(t, viewResult)
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")
	})

	// Test 3: Enable debug-claude on existing session
	t.Run("enable on existing session", func(t *testing.T) {
		// First verify it's false
		viewResult := harness.RunCommand(t, env, "sessions", "view", "no-debug-test")
		harness.AssertSuccess(t, viewResult)
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")

		// Enable debug-claude
		setResult := harness.RunCommand(t, env, "sessions", "set", "no-debug-test", "--variable", "debug-claude", "--value", "true")
		harness.AssertSuccess(t, setResult)
		harness.AssertStdoutContains(t, setResult, "Updated")

		// Verify it's now true
		viewResult2 := harness.RunCommand(t, env, "sessions", "view", "no-debug-test")
		harness.AssertSuccess(t, viewResult2)
		harness.AssertStdoutContains(t, viewResult2, "Debug Claude: true")
	})

	// Test 4: Disable debug-claude on existing session
	t.Run("disable on existing session", func(t *testing.T) {
		// Enable first
		harness.RunCommand(t, env, "sessions", "set", "debug-test", "--variable", "debug-claude", "--value", "true")

		// Now disable
		setResult := harness.RunCommand(t, env, "sessions", "set", "debug-test", "--variable", "debug-claude", "--value", "false")
		harness.AssertSuccess(t, setResult)

		// Verify it's false
		viewResult := harness.RunCommand(t, env, "sessions", "view", "debug-test")
		harness.AssertSuccess(t, viewResult)
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")
	})

	// Test 5: Toggle debug-claude multiple times
	t.Run("toggle multiple times", func(t *testing.T) {
		result := harness.RunCommand(t, env, "sessions", "add", "toggle-test")
		harness.AssertSuccess(t, result)

		// Enable
		harness.RunCommand(t, env, "sessions", "set", "toggle-test", "--variable", "debug-claude", "--value", "true")
		viewResult := harness.RunCommand(t, env, "sessions", "view", "toggle-test")
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")

		// Disable
		harness.RunCommand(t, env, "sessions", "set", "toggle-test", "--variable", "debug-claude", "--value", "false")
		viewResult = harness.RunCommand(t, env, "sessions", "view", "toggle-test")
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")

		// Enable again
		harness.RunCommand(t, env, "sessions", "set", "toggle-test", "--variable", "debug-claude", "--value", "true")
		viewResult = harness.RunCommand(t, env, "sessions", "view", "toggle-test")
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")
	})

	// Test 6: List sessions shows debug-claude in JSON format
	t.Run("list shows debug-claude in JSON", func(t *testing.T) {
		listResult := harness.RunCommand(t, env, "sessions", "list", "--format", "json")
		harness.AssertSuccess(t, listResult)

		// Parse JSON
		var sessions []map[string]interface{}
		err := json.Unmarshal([]byte(listResult.Stdout), &sessions)
		if err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Find debug-test session and verify it has DebugClaude field
		found := false
		for _, session := range sessions {
			if session["Name"] == "debug-test" {
				found = true
				_, hasDebugClaude := session["DebugClaude"]
				if !hasDebugClaude {
					t.Error("Session JSON missing DebugClaude field")
				}
				break
			}
		}
		if !found {
			t.Error("debug-test session not found in list output")
		}
	})
}

func TestDebugClaudeIndependentFromSkipPermissions(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Test that DebugClaude and AllowDangerouslySkipPermissions are independent
	t.Run("flags are independent", func(t *testing.T) {
		// Create session with both flags
		result := harness.RunCommand(t, env, "sessions", "add", "both-flags",
			"--debug-claude",
			"--allow-dangerously-skip-permissions")
		harness.AssertSuccess(t, result)

		// Verify both are true
		viewResult := harness.RunCommand(t, env, "sessions", "view", "both-flags")
		harness.AssertSuccess(t, viewResult)
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")
		harness.AssertStdoutContains(t, viewResult, "Allow Dangerously Skip Permissions: true")

		// Disable debug-claude, skip-permissions should remain true
		harness.RunCommand(t, env, "sessions", "set", "both-flags", "--variable", "debug-claude", "--value", "false")
		viewResult = harness.RunCommand(t, env, "sessions", "view", "both-flags")
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")
		harness.AssertStdoutContains(t, viewResult, "Allow Dangerously Skip Permissions: true")

		// Disable skip-permissions, it should remain false
		harness.RunCommand(t, env, "sessions", "set", "both-flags", "--variable", "allow-dangerously-skip-permissions", "--value", "false")
		viewResult = harness.RunCommand(t, env, "sessions", "view", "both-flags")
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: false")
		harness.AssertStdoutContains(t, viewResult, "Allow Dangerously Skip Permissions: false")

		// Enable debug-claude again, skip-permissions should stay false
		harness.RunCommand(t, env, "sessions", "set", "both-flags", "--variable", "debug-claude", "--value", "true")
		viewResult = harness.RunCommand(t, env, "sessions", "view", "both-flags")
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")
		harness.AssertStdoutContains(t, viewResult, "Allow Dangerously Skip Permissions: false")
	})
}

func TestDebugClaudeWithYesNoValues(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"yes value", "yes", "true"},
		{"no value", "no", "false"},
		{"1 value", "1", "true"},
		{"0 value", "0", "false"},
		{"true value", "true", "true"},
		{"false value", "false", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionName := "yesno-test-" + tt.name
			harness.RunCommand(t, env, "sessions", "add", sessionName)

			setResult := harness.RunCommand(t, env, "sessions", "set", sessionName, "--variable", "debug-claude", "--value", tt.value)
			harness.AssertSuccess(t, setResult)

			viewResult := harness.RunCommand(t, env, "sessions", "view", sessionName)
			harness.AssertSuccess(t, viewResult)
			harness.AssertStdoutContains(t, viewResult, "Debug Claude: "+tt.expected)
		})
	}
}

func TestDebugClaudeInvalidValue(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	harness.RunCommand(t, env, "sessions", "add", "invalid-test")

	// Try to set invalid value
	result := harness.RunCommand(t, env, "sessions", "set", "invalid-test", "--variable", "debug-claude", "--value", "maybe")
	harness.AssertFailure(t, result)
}

func TestDebugClaudeSetAll(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Create multiple sessions
	harness.RunCommand(t, env, "sessions", "add", "bulk-1")
	harness.RunCommand(t, env, "sessions", "add", "bulk-2")
	harness.RunCommand(t, env, "sessions", "add", "bulk-3")

	// Enable debug-claude for all sessions
	result := harness.RunCommand(t, env, "sessions", "set", "--all", "--variable", "debug-claude", "--value", "true")
	harness.AssertSuccess(t, result)
	harness.AssertStdoutContains(t, result, "Updating debug-claude for 3 sessions")
	harness.AssertStdoutContains(t, result, "Updated 'bulk-1'")
	harness.AssertStdoutContains(t, result, "Updated 'bulk-2'")
	harness.AssertStdoutContains(t, result, "Updated 'bulk-3'")

	// Verify all have debug enabled
	for _, name := range []string{"bulk-1", "bulk-2", "bulk-3"} {
		viewResult := harness.RunCommand(t, env, "sessions", "view", name)
		harness.AssertSuccess(t, viewResult)
		harness.AssertStdoutContains(t, viewResult, "Debug Claude: true")
	}
}
