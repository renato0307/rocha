package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSessionsList(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "list empty returns success",
			args:         []string{"sessions", "list"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Total: 0 sessions")
			},
		},
		{
			name: "list with one session",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "session-a")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "list"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "session-a")
				harness.AssertStdoutContains(t, result, "Total: 1 sessions")
			},
		},
		{
			name: "list with multiple sessions",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "session-a")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "add", "session-b")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "add", "session-c")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "list"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "session-a")
				harness.AssertStdoutContains(t, result, "session-b")
				harness.AssertStdoutContains(t, result, "session-c")
				harness.AssertStdoutContains(t, result, "Total: 3 sessions")
			},
		},
		{
			name:         "list JSON format empty",
			args:         []string{"sessions", "list", "--format", "json"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				var sessions []map[string]any
				harness.AssertValidJSON(t, result, &sessions)
				if len(sessions) != 0 {
					t.Errorf("Expected 0 sessions, got %d", len(sessions))
				}
			},
		},
		{
			name: "list JSON format with sessions",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "json-test",
					"--display-name", "JSON Test Session")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "list", "--format", "json"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				var sessions []map[string]any
				harness.AssertValidJSON(t, result, &sessions)
				if len(sessions) != 1 {
					t.Errorf("Expected 1 session, got %d", len(sessions))
				}
				if sessions[0]["Name"] != "json-test" {
					t.Errorf("Expected session name 'json-test', got %v", sessions[0]["Name"])
				}
			},
		},
		{
			name: "list excludes archived by default",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "active-session")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "add", "archived-session")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "archive", "archived-session", "-f", "-s")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "list"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "active-session")
				if strings.Contains(result.Stdout, "archived-session") {
					t.Error("Archived session should not appear in default list")
				}
				harness.AssertStdoutContains(t, result, "Total: 1 sessions")
			},
		},
		{
			name: "list with show-archived flag includes archived",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "visible")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "add", "hidden")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "archive", "hidden", "-f", "-s")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "list", "-a"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "visible")
				harness.AssertStdoutContains(t, result, "hidden")
				harness.AssertStdoutContains(t, result, "Total: 2 sessions")
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

func TestSessionsListJSONStructure(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Add a session with specific fields
	result := harness.RunCommand(t, env,
		"sessions", "add", "structured-test",
		"--display-name", "Structured Test",
		"--branch-name", "main",
		"--state", "working",
	)
	harness.AssertSuccess(t, result)

	result = harness.RunCommand(t, env, "sessions", "list", "--format", "json")
	harness.AssertSuccess(t, result)

	var sessions []struct {
		Name        string `json:"Name"`
		DisplayName string `json:"DisplayName"`
		BranchName  string `json:"BranchName"`
		State       string `json:"State"`
	}

	err := json.Unmarshal([]byte(result.Stdout), &sessions)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.Name != "structured-test" {
		t.Errorf("Expected Name 'structured-test', got %q", s.Name)
	}
	if s.DisplayName != "Structured Test" {
		t.Errorf("Expected DisplayName 'Structured Test', got %q", s.DisplayName)
	}
	if s.BranchName != "main" {
		t.Errorf("Expected BranchName 'main', got %q", s.BranchName)
	}
	if s.State != "working" {
		t.Errorf("Expected State 'working', got %q", s.State)
	}
}
