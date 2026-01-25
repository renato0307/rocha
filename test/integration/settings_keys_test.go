package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSettingsKeysList(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "list shows defaults when no settings",
			args:         []string{"settings", "keys", "list"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "archive")
				harness.AssertStdoutContains(t, result, "a") // default key
				harness.AssertStdoutContains(t, result, "help")
				harness.AssertStdoutContains(t, result, "h, ?") // default keys
			},
		},
		{
			name: "list shows custom key when configured",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "settings", "keys", "set", "archive", "A")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"settings", "keys", "list"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "archive")
				harness.AssertStdoutContains(t, result, "A")
			},
		},
		{
			name:         "list JSON format",
			args:         []string{"settings", "keys", "list", "--format", "json"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				var keys map[string]any
				harness.AssertValidJSON(t, result, &keys)
				// Verify archive key has default
				if archiveData, ok := keys["archive"].(map[string]any); ok {
					if _, hasDefault := archiveData["default"]; !hasDefault {
						t.Error("Expected 'archive' to have 'default' field")
					}
				} else {
					t.Error("Expected 'archive' key in output")
				}
			},
		},
		{
			name: "list JSON format with custom keys",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "settings", "keys", "set", "help", "H")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"settings", "keys", "list", "--format", "json"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				var keys map[string]any
				harness.AssertValidJSON(t, result, &keys)
				// Verify help key has custom binding
				if helpData, ok := keys["help"].(map[string]any); ok {
					if custom, hasCustom := helpData["custom"]; hasCustom {
						if customArr, ok := custom.([]any); ok {
							if len(customArr) != 1 || customArr[0] != "H" {
								t.Errorf("Expected custom to be ['H'], got %v", customArr)
							}
						}
					} else {
						t.Error("Expected 'help' to have 'custom' field")
					}
				}
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

func TestSettingsKeysSet(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "set valid key",
			args:         []string{"settings", "keys", "set", "archive", "A"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Set 'archive' to: A")
				// Verify settings file was created/updated
				listResult := harness.RunCommand(t, env, "settings", "keys", "list")
				harness.AssertStdoutContains(t, listResult, "A")
			},
		},
		{
			name:         "set invalid key name",
			args:         []string{"settings", "keys", "set", "invalid_key", "a"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStderrContains(t, result, "unknown key")
			},
		},
		{
			name: "set conflicting key fails",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "settings", "keys", "set", "archive", "z")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"settings", "keys", "set", "kill", "z"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStderrContains(t, result, "conflict")
			},
		},
		{
			name:         "set multiple keys with comma",
			args:         []string{"settings", "keys", "set", "up", "up,k,w"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Set 'up' to: up, k, w")
				// Verify via list command
				listResult := harness.RunCommand(t, env, "settings", "keys", "list", "--format", "json")
				var keys map[string]any
				harness.AssertValidJSON(t, listResult, &keys)
				if upData, ok := keys["up"].(map[string]any); ok {
					if custom, ok := upData["custom"].([]any); ok {
						if len(custom) != 3 {
							t.Errorf("Expected 3 custom keys, got %d", len(custom))
						}
					}
				}
			},
		},
		{
			name:         "set empty value fails",
			args:         []string{"settings", "keys", "set", "archive", ""},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStderrContains(t, result, "cannot be empty")
			},
		},
		{
			name: "override existing custom key",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "settings", "keys", "set", "archive", "A")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"settings", "keys", "set", "archive", "Z"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Set 'archive' to: Z")
				// Verify the new value
				listResult := harness.RunCommand(t, env, "settings", "keys", "list", "--format", "json")
				var keys map[string]any
				err := json.Unmarshal([]byte(listResult.Stdout), &keys)
				if err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				if archiveData, ok := keys["archive"].(map[string]any); ok {
					if custom, ok := archiveData["custom"].([]any); ok {
						if len(custom) != 1 || custom[0] != "Z" {
							t.Errorf("Expected custom to be ['Z'], got %v", custom)
						}
					}
				}
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
