package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/renato0307/rocha/internal/logging"
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

	// Load current state to get ExecutionID, ClaudeDir, and agent CLI flags for this session
	var claudeDir string
	var executionID string
	var allowDangerouslySkipPermissions bool

	ctx := context.Background()
	st, err := cli.Container.SessionService.LoadState(ctx, false)
	if err != nil {
		logging.Logger.Warn("Failed to load state for execution ID", "error", err)
		executionID = os.Getenv("ROCHA_EXECUTION_ID")
		if executionID == "" {
			executionID = "unknown"
		}
	} else {
		// Get ExecutionID, ClaudeDir, and agent CLI flags from session info
		if session, exists := st.Sessions[sessionName]; exists {
			claudeDir = session.ClaudeDir
			executionID = session.ExecutionID
			allowDangerouslySkipPermissions = session.AllowDangerouslySkipPermissions
			logging.Logger.Info("Using execution ID from session", "execution_id", executionID)
			if allowDangerouslySkipPermissions {
				logging.Logger.Warn("DANGEROUS MODE ENABLED: Claude will skip permission prompts",
					"session", sessionName)
			}
			if claudeDir != "" {
				logging.Logger.Info("Using ClaudeDir from session", "claude_dir", claudeDir)
			}
		} else {
			// Session not found, fallback to env or unknown
			executionID = os.Getenv("ROCHA_EXECUTION_ID")
			if executionID == "" {
				executionID = "unknown"
			}
			logging.Logger.Warn("Session not found, using fallback execution ID", "execution_id", executionID)
		}
	}

	logging.Logger.Info("Starting Claude with hooks",
		"session", sessionName,
		"execution_id", executionID,
		"allow_dangerously_skip_permissions", allowDangerouslySkipPermissions)

	// Build the hooks configuration with multiple event types
	hooks := map[string]interface{}{
		"hooks": map[string]interface{}{
			// Stop: When Claude finishes and is waiting for input
			"Stop": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify handle %s stop --execution-id=%s", rochaBin, sessionName, executionID),
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
							"command": fmt.Sprintf("%s notify handle %s prompt --execution-id=%s", rochaBin, sessionName, executionID),
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
							"command": fmt.Sprintf("%s notify handle %s start --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// PermissionRequest: When Claude needs user permission (replaces Notification hook)
			"PermissionRequest": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify handle %s permission-request --execution-id=%s", rochaBin, sessionName, executionID),
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
							"command": fmt.Sprintf("%s notify handle %s end --execution-id=%s", rochaBin, sessionName, executionID),
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
							"command": fmt.Sprintf("%s notify handle %s notification --execution-id=%s", rochaBin, sessionName, executionID),
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
							"command": fmt.Sprintf("%s notify handle %s working --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// PostToolUseFailure: When a tool fails and Claude continues
			"PostToolUseFailure": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify handle %s tool-failure --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// SubagentStart: When Claude spawns a subagent
			"SubagentStart": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify handle %s subagent-start --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// SubagentStop: When a subagent finishes
			"SubagentStop": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify handle %s subagent-stop --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// PreCompact: When Claude compresses context
			"PreCompact": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify handle %s pre-compact --execution-id=%s", rochaBin, sessionName, executionID),
						},
					},
				},
			},
			// Setup: When Claude does initialization or maintenance work
			"Setup": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": fmt.Sprintf("%s notify handle %s setup --execution-id=%s", rochaBin, sessionName, executionID),
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

	// Add --allow-dangerously-skip-permissions flag if enabled for this session
	if allowDangerouslySkipPermissions {
		args = append(args, "--allow-dangerously-skip-permissions")
		logging.Logger.Warn("Adding --allow-dangerously-skip-permissions to Claude command",
			"session", sessionName)
	}

	args = append(args, s.Args...)

	// Find claude executable
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Execute claude using syscall.Exec to replace the current process
	// This ensures claude receives all signals properly and behaves as if run directly
	env := os.Environ()

	// Set CLAUDE_CONFIG_DIR if configured for this session
	if claudeDir != "" {
		env = append(env, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", claudeDir))
		logging.Logger.Info("Setting CLAUDE_CONFIG_DIR environment variable", "path", claudeDir)
	}

	if err := syscall.Exec(claudePath, append([]string{"claude"}, args...), env); err != nil {
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// This line should never be reached if Exec succeeds
	return nil
}
