package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"rocha/logging"
	"syscall"
)

// StartClaudeCmd starts Claude Code with hooks configured
type StartClaudeCmd struct {
	Args []string `arg:"" optional:"" help:"Additional arguments to pass to claude"`
}

// Run executes Claude with hooks configuration
func (s *StartClaudeCmd) Run() error {
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

	// Build the hooks configuration with multiple event types
	hooks := map[string]interface{}{
		"hooks": map[string]interface{}{
			// Stop: When Claude finishes and is waiting for input
			"Stop": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s stop", rochaBin, sessionName),
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
							"command": fmt.Sprintf("%s notify %s prompt", rochaBin, sessionName),
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
							"command": fmt.Sprintf("%s notify %s start", rochaBin, sessionName),
						},
					},
				},
			},
			// Notification: Catch all notification types (including when Claude needs input)
			"Notification": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify %s notification", rochaBin, sessionName),
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
