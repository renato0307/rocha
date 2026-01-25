package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestSessionsView(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, env *harness.TestEnvironment)
		args         []string
		wantExitCode int
		validate     func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult)
	}{
		{
			name:         "view nonexistent session fails",
			args:         []string{"sessions", "view", "does-not-exist"},
			wantExitCode: 1,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertFailure(t, result)
			},
		},
		{
			name: "view existing session table format",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env,
					"sessions", "add", "view-test",
					"--display-name", "View Test Session",
					"--branch-name", "feature/test",
					"--state", "working",
				)
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "view", "view-test"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Session: view-test")
				harness.AssertStdoutContains(t, result, "Display Name: View Test Session")
				harness.AssertStdoutContains(t, result, "State: working")
				harness.AssertStdoutContains(t, result, "Branch Name: feature/test")
			},
		},
		{
			name: "view existing session JSON format",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env,
					"sessions", "add", "json-view",
					"--display-name", "JSON View Session",
					"--state", "idle",
				)
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "view", "json-view", "--format", "json"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				var session map[string]any
				harness.AssertValidJSON(t, result, &session)
				if session["Name"] != "json-view" {
					t.Errorf("Expected Name 'json-view', got %v", session["Name"])
				}
				if session["DisplayName"] != "JSON View Session" {
					t.Errorf("Expected DisplayName 'JSON View Session', got %v", session["DisplayName"])
				}
				if session["State"] != "idle" {
					t.Errorf("Expected State 'idle', got %v", session["State"])
				}
			},
		},
		{
			name: "view shows archived status",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "archive-view-test")
				harness.AssertSuccess(t, result)
				result = harness.RunCommand(t, env, "sessions", "archive", "archive-view-test", "-f", "-s")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "view", "archive-view-test"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Archived: true")
			},
		},
		{
			name: "view shows not archived status",
			setup: func(t *testing.T, env *harness.TestEnvironment) {
				result := harness.RunCommand(t, env, "sessions", "add", "not-archived-test")
				harness.AssertSuccess(t, result)
			},
			args:         []string{"sessions", "view", "not-archived-test"},
			wantExitCode: 0,
			validate: func(t *testing.T, env *harness.TestEnvironment, result harness.CommandResult) {
				harness.AssertStdoutContains(t, result, "Archived: false")
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

func TestSessionsViewJSONStructure(t *testing.T) {
	env := harness.NewTestEnvironment(t)

	// Add a session with specific fields
	result := harness.RunCommand(t, env,
		"sessions", "add", "structured-view",
		"--display-name", "Structured View Test",
		"--branch-name", "main",
		"--repo-path", "/home/test/repo",
		"--state", "waiting",
	)
	harness.AssertSuccess(t, result)

	result = harness.RunCommand(t, env, "sessions", "view", "structured-view", "--format", "json")
	harness.AssertSuccess(t, result)

	var session struct {
		Name        string `json:"Name"`
		DisplayName string `json:"DisplayName"`
		BranchName  string `json:"BranchName"`
		RepoPath    string `json:"RepoPath"`
		State       string `json:"State"`
		IsArchived  bool   `json:"IsArchived"`
		IsFlagged   bool   `json:"IsFlagged"`
	}

	err := json.Unmarshal([]byte(result.Stdout), &session)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if session.Name != "structured-view" {
		t.Errorf("Expected Name 'structured-view', got %q", session.Name)
	}
	if session.DisplayName != "Structured View Test" {
		t.Errorf("Expected DisplayName 'Structured View Test', got %q", session.DisplayName)
	}
	if session.BranchName != "main" {
		t.Errorf("Expected BranchName 'main', got %q", session.BranchName)
	}
	if session.RepoPath != "/home/test/repo" {
		t.Errorf("Expected RepoPath '/home/test/repo', got %q", session.RepoPath)
	}
	if session.State != "waiting" {
		t.Errorf("Expected State 'waiting', got %q", session.State)
	}
	if session.IsArchived {
		t.Errorf("Expected IsArchived false, got %v", session.IsArchived)
	}
}
