package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"rocha/logging"
	"rocha/storage"
)

// StartClaudeCmd starts Claude Code with hooks configured
type StartClaudeCmd struct {
	Args []string `arg:"" optional:"" help:"Additional arguments to pass to claude"`
}

// Run executes Claude with hooks configuration
func (s *StartClaudeCmd) Run(cli *CLI) error {
	// Get the path to the rocha binary
	rochaBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get rocha executable path: %w", err)
	}

	// Get session name from environment variable
	sessionName := os.Getenv("ROCHA_SESSION_NAME")
	if sessionName == "" {
		sessionName = "unknown"
	}

	// Load current state to get ExecutionID for this session
	var executionID string
	dbPath := expandPath(cli.DBPath)
	store, err := storage.NewStore(dbPath)
	if err != nil {
		logging.Logger.Warn("Failed to open database for execution ID", "error", err)
		// Fall back to environment variable
		executionID = os.Getenv("ROCHA_EXECUTION_ID")
		if executionID == "" {
			executionID = "unknown"
		}
	} else {
		defer store.Close()
		st, err := store.Load(context.Background(), false)
		if err != nil {
			logging.Logger.Warn("Failed to load state for execution ID", "error", err)
			executionID = os.Getenv("ROCHA_EXECUTION_ID")
			if executionID == "" {
				executionID = "unknown"
			}
		} else {
			// Get ExecutionID from session info
			if session, exists := st.Sessions[sessionName]; exists {
				executionID = session.ExecutionID
				logging.Logger.Info("Using execution ID from session", "execution_id", executionID)
			} else {
				// Session not found, fallback to env or unknown
				executionID = os.Getenv("ROCHA_EXECUTION_ID")
				if executionID == "" {
					executionID = "unknown"
				}
				logging.Logger.Warn("Session not found, using fallback execution ID", "execution_id", executionID)
			}
		}
	}

	logging.Logger.Info("Starting Claude with hooks",
		"session", sessionName,
		"execution_id", executionID)

	// Build the hooks configuration with multiple event types
	hooks := map[string]interface{}{
		"hooks": map[string]interface{}{
			// Stop: When Claude finishes and is waiting for input
			"Stop": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s stop --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// UserPromptSubmit: When user submits a prompt (Claude starts processing)
			"UserPromptSubmit": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s prompt --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// SessionStart: When Claude Code session initializes
			"SessionStart": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s start --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// Notification: Catch permission_prompt (when Claude needs user permission)
			"Notification": []map[string]interface{}{
				{
					"matcher": "permission_prompt",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s notification --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// SessionEnd: When Claude Code session ends
			"SessionEnd": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s end --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// PreToolUse: When Claude is about to ask user a question
			"PreToolUse": []map[string]interface{}{
				{
					"matcher": "AskUserQuestion",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s notification --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// PostToolUse: When AskUserQuestion tool completes (user has answered)
			"PostToolUse": []map[string]interface{}{
				{
					"matcher": "AskUserQuestion",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s working --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
		},
	}

	// Serialize to JSON
	hooksJSON, err := json.Marshal(hooks)
	if err != nil {
		return fmt.Errorf("failed to marshal hooks configuration: %w", err)
	}

	// Log the hooks configuration for debugging
	logging.Logger.Debug("Claude hooks configuration",
		"session", sessionName,
		"hooks_json", string(hooksJSON))

	// Build claude command with settings
	args := []string{"--settings", string(hooksJSON)}
	args = append(args, s.Args...)

	// Find claude executable
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Execute claude using syscall.Exec to replace the current process
	// This ensures claude receives all signals properly and behaves as if run directly
	env := os.Environ()
	if err := syscall.Exec(claudePath, append([]string{"claude"}, args...), env); err != nil {
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// This line should never be reached if Exec succeeds
	return nil
}
