package integration_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSessionsViewAgentSettings(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment) string
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "view agent settings for nonexistent session fails",
			args:         []string{"sessions", "view-agent-settings", "does-not-exist"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name: "view agent settings for existing session",
			setup: func(t *testing.T, env *harness.TestEnvironment) string {
				result := harness.RunCommand(t, env,
					"sessions", "add", "agent-settings-test",
					"--display-name", "Agent Settings Test",
					"--start-claude",
				)
				harness.AssertSuccess(t, result)

				// Give Claude a moment to start
				time.Sleep(2 * time.Second)

				return "agent-settings-test"
			},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertSuccess(t, result)

				// Parse JSON output
				var settings map[string]any
				err := json.Unmarshal([]byte(result.Stdout), &settings)
				if err != nil {
					t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, result.Stdout)
				}

				// Verify hooks structure exists
				hooks, ok := settings["hooks"].(map[string]any)
				if !ok {
					t.Fatalf("Expected 'hooks' key in output, got: %v", settings)
				}

				// Verify required hook types are present
				requiredHooks := []string{
					"Stop",
					"UserPromptSubmit",
					"SessionStart",
					"SessionEnd",
					"Notification",
					"PreToolUse",
					"PostToolUse",
				}

				for _, hookName := range requiredHooks {
					if _, exists := hooks[hookName]; !exists {
						t.Errorf("Expected hook '%s' not found in output", hookName)
					}
				}

				// Verify session name appears in commands
				if !strings.Contains(result.Stdout, "agent-settings-test") {
					t.Errorf("Expected session name 'agent-settings-test' in output, got: %s", result.Stdout)
				}

				// Verify notify command structure
				if !strings.Contains(result.Stdout, "notify agent-settings-test") {
					t.Errorf("Expected 'notify agent-settings-test' command in output")
				}
			},
		},
		{
			name: "agent settings contain session-specific execution ID",
			setup: func(t *testing.T, env *harness.TestEnvironment) string {
				result := harness.RunCommand(t, env,
					"sessions", "add", "exec-id-test",
					"--start-claude",
				)
				harness.AssertSuccess(t, result)

				// Give Claude a moment to start
				time.Sleep(2 * time.Second)

				return "exec-id-test"
			},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertSuccess(t, result)

				// Execution ID should be present in the output
				if !strings.Contains(result.Stdout, "--execution-id=") {
					t.Errorf("Expected '--execution-id=' in output, got: %s", result.Stdout)
				}

				// Parse and verify structure
				var settings map[string]any
				err := json.Unmarshal([]byte(result.Stdout), &settings)
				if err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}

				hooks := settings["hooks"].(map[string]any)
				stopHook := hooks["Stop"].([]any)[0].(map[string]any)
				innerHooks := stopHook["hooks"].([]any)[0].(map[string]any)
				command := innerHooks["command"].(string)

				if !strings.Contains(command, "notify exec-id-test stop --execution-id=") {
					t.Errorf("Expected command with execution-id, got: %s", command)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := harness.NewTestEnvironment(t)

			var sessionName string
			if tt.setup != nil {
				sessionName = tt.setup(t, env)
			}

			// Use provided args or construct from session name
			args := tt.args
			if args == nil && sessionName != "" {
				args = []string{"sessions", "view-agent-settings", sessionName}
			}

			result := harness.RunCommand(t, env, args...)

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

func TestSessionsViewAgentSettingsHooksStructure(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Create a session
	result := harness.RunCommand(t, env,
		"sessions", "add", "hooks-structure-test",
		"--start-claude",
	)
	harness.AssertSuccess(t, result)

	// Give Claude a moment to start
	time.Sleep(2 * time.Second)

	// View agent settings
	result = harness.RunCommand(t, env,
		"sessions", "view-agent-settings", "hooks-structure-test",
	)
	harness.AssertSuccess(t, result)

	// Parse JSON output
	var settings map[string]any
	err := json.Unmarshal([]byte(result.Stdout), &settings)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	hooks := settings["hooks"].(map[string]any)

	// Test Stop hook structure
	stopHooks := hooks["Stop"].([]any)
	if len(stopHooks) == 0 {
		t.Error("Expected at least one Stop hook")
	}
	stopHook := stopHooks[0].(map[string]any)
	innerHooks := stopHook["hooks"].([]any)
	if len(innerHooks) == 0 {
		t.Error("Expected at least one inner hook in Stop")
	}
	innerHook := innerHooks[0].(map[string]any)
	if innerHook["type"] != "command" {
		t.Errorf("Expected type 'command', got %v", innerHook["type"])
	}
	if _, ok := innerHook["command"].(string); !ok {
		t.Error("Expected command to be a string")
	}

	// Test UserPromptSubmit hook structure
	promptHooks := hooks["UserPromptSubmit"].([]any)
	if len(promptHooks) == 0 {
		t.Error("Expected at least one UserPromptSubmit hook")
	}

	// Test SessionStart hook structure
	startHooks := hooks["SessionStart"].([]any)
	if len(startHooks) == 0 {
		t.Error("Expected at least one SessionStart hook")
	}

	// Test Notification hook with matcher
	notificationHooks := hooks["Notification"].([]any)
	if len(notificationHooks) == 0 {
		t.Error("Expected at least one Notification hook")
	}
	notificationHook := notificationHooks[0].(map[string]any)
	if notificationHook["matcher"] != "permission_prompt" {
		t.Errorf("Expected matcher 'permission_prompt', got %v", notificationHook["matcher"])
	}

	// Test PreToolUse hook with matcher
	preToolUseHooks := hooks["PreToolUse"].([]any)
	if len(preToolUseHooks) == 0 {
		t.Error("Expected at least one PreToolUse hook")
	}
	preToolUseHook := preToolUseHooks[0].(map[string]any)
	if preToolUseHook["matcher"] != "AskUserQuestion" {
		t.Errorf("Expected matcher 'AskUserQuestion', got %v", preToolUseHook["matcher"])
	}

	// Test PostToolUse hook with matcher
	postToolUseHooks := hooks["PostToolUse"].([]any)
	if len(postToolUseHooks) == 0 {
		t.Error("Expected at least one PostToolUse hook")
	}
	postToolUseHook := postToolUseHooks[0].(map[string]any)
	if postToolUseHook["matcher"] != "AskUserQuestion" {
		t.Errorf("Expected matcher 'AskUserQuestion', got %v", postToolUseHook["matcher"])
	}
}

func TestSessionsViewAgentSettingsCommandFormat(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Create a session
	result := harness.RunCommand(t, env,
		"sessions", "add", "command-format-test",
		"--start-claude",
	)
	harness.AssertSuccess(t, result)

	// Give Claude a moment to start
	time.Sleep(2 * time.Second)

	// View agent settings
	result = harness.RunCommand(t, env,
		"sessions", "view-agent-settings", "command-format-test",
	)
	harness.AssertSuccess(t, result)

	// Parse output
	var settings map[string]any
	err := json.Unmarshal([]byte(result.Stdout), &settings)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	hooks := settings["hooks"].(map[string]any)

	// Verify command formats for each hook type
	testCases := []struct {
		hookName        string
		expectedSuffix  string
		requiresMatcher bool
		matcherValue    string
	}{
		{
			hookName:       "Stop",
			expectedSuffix: "stop",
		},
		{
			hookName:       "UserPromptSubmit",
			expectedSuffix: "prompt",
		},
		{
			hookName:       "SessionStart",
			expectedSuffix: "start",
		},
		{
			hookName:       "SessionEnd",
			expectedSuffix: "end",
		},
		{
			hookName:        "Notification",
			expectedSuffix:  "notification",
			requiresMatcher: true,
			matcherValue:    "permission_prompt",
		},
		{
			hookName:        "PreToolUse",
			expectedSuffix:  "notification",
			requiresMatcher: true,
			matcherValue:    "AskUserQuestion",
		},
		{
			hookName:        "PostToolUse",
			expectedSuffix:  "working",
			requiresMatcher: true,
			matcherValue:    "AskUserQuestion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.hookName, func(t *testing.T) {
			hookArray := hooks[tc.hookName].([]any)
			if len(hookArray) == 0 {
				t.Fatalf("Expected at least one %s hook", tc.hookName)
			}

			hookObj := hookArray[0].(map[string]any)

			// Check matcher if required
			if tc.requiresMatcher {
				matcher, ok := hookObj["matcher"].(string)
				if !ok {
					t.Errorf("Expected matcher in %s hook", tc.hookName)
				}
				if matcher != tc.matcherValue {
					t.Errorf("Expected matcher '%s', got '%s'", tc.matcherValue, matcher)
				}
			}

			// Check command format
			innerHooks := hookObj["hooks"].([]any)
			if len(innerHooks) == 0 {
				t.Fatalf("Expected at least one inner hook in %s", tc.hookName)
			}

			innerHook := innerHooks[0].(map[string]any)
			command := innerHook["command"].(string)

			expectedPattern := "notify command-format-test " + tc.expectedSuffix + " --execution-id="
			if !strings.Contains(command, expectedPattern) {
				t.Errorf("Expected command to contain '%s', got: %s", expectedPattern, command)
			}
		})
	}
}
