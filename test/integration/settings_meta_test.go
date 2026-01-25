package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSettingsMeta(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		validate     func(t *testing.T, result harness.CommandResult)
	}{
		{
			name:         "table format (default)",
			args:         []string{"settings", "meta"},
			wantExitCode: 0,
			validate: func(t *testing.T, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Settings file:")
				harness.AssertStdoutContains(t, result, "Example settings.json:")
			},
		},
		{
			name:         "table format explicit",
			args:         []string{"settings", "meta", "--format", "table"},
			wantExitCode: 0,
			validate: func(t *testing.T, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Settings file:")
				harness.AssertStdoutContains(t, result, "Example settings.json:")
			},
		},
		{
			name:         "json format",
			args:         []string{"settings", "meta", "--format", "json"},
			wantExitCode: 0,
			validate: func(t *testing.T, result harness.CommandResult) {
				// Verify output is valid JSON
				var output map[string]any
				err := json.Unmarshal([]byte(result.Stdout), &output)
				if err != nil {
					t.Errorf("Expected valid JSON output, got error: %v", err)
					return
				}

				// Check for expected fields
				if _, ok := output["settings_file"]; !ok {
					t.Error("Expected 'settings_file' field in JSON output")
				}
				if _, ok := output["format"]; !ok {
					t.Error("Expected 'format' field in JSON output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := harness.NewTestEnvironment(t)

			result := harness.RunCommand(t, env, tt.args...)

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
