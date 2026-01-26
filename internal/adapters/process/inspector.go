package process

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// OSProcessInspector implements ProcessInspector using OS commands (ps, pgrep, tmux)
type OSProcessInspector struct{}

// Compile-time interface verification
var _ ports.ProcessInspector = (*OSProcessInspector)(nil)

// NewOSProcessInspector creates a new OS process inspector
func NewOSProcessInspector() *OSProcessInspector {
	return &OSProcessInspector{}
}

// GetClaudeSettings extracts --settings JSON from running Claude process
func (i *OSProcessInspector) GetClaudeSettings(sessionName string) (string, error) {
	// 1. Get tmux pane PID
	panePID, err := i.getTmuxPanePID(sessionName)
	if err != nil {
		return "", fmt.Errorf("failed to get tmux pane PID: %w", err)
	}

	logging.Logger.Debug("Found tmux pane", "pane_pid", panePID, "session", sessionName)

	// 2. Find Claude process under pane
	claudePID, err := i.getClaudeProcessPID(panePID)
	if err != nil {
		return "", fmt.Errorf("no running Claude process found: %w", err)
	}

	logging.Logger.Debug("Found Claude process", "claude_pid", claudePID, "session", sessionName)

	// 3. Extract command line
	commandLine, err := i.getProcessCommandLine(claudePID)
	if err != nil {
		return "", fmt.Errorf("failed to get process command line: %w", err)
	}

	// 4. Extract --settings JSON
	settingsJSON, err := i.extractSettingsFromCommandLine(commandLine)
	if err != nil {
		return "", fmt.Errorf("failed to extract settings: %w", err)
	}

	return settingsJSON, nil
}

func (i *OSProcessInspector) getTmuxPanePID(sessionName string) (string, error) {
	cmd := exec.Command("tmux", "list-panes", "-t", sessionName, "-F", "#{pane_pid}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("session not running or not found: %w", err)
	}

	panePID := strings.TrimSpace(string(output))
	if panePID == "" {
		return "", fmt.Errorf("no pane PID found")
	}

	return panePID, nil
}

func (i *OSProcessInspector) getClaudeProcessPID(panePID string) (string, error) {
	cmd := exec.Command("pgrep", "-P", panePID, "-f", "claude")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude process not found: %w", err)
	}

	claudePID := strings.TrimSpace(string(output))
	if claudePID == "" {
		return "", fmt.Errorf("no claude PID found")
	}

	return claudePID, nil
}

func (i *OSProcessInspector) getProcessCommandLine(pid string) (string, error) {
	cmd := exec.Command("ps", "-p", pid, "-o", "command=")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read process: %w", err)
	}

	return string(output), nil
}

func (i *OSProcessInspector) extractSettingsFromCommandLine(commandLine string) (string, error) {
	// Pattern: --settings {JSON}
	// JSON starts with { and we find matching }
	re := regexp.MustCompile(`--settings\s+(\{.+\})`)
	matches := re.FindStringSubmatch(commandLine)

	if len(matches) < 2 {
		return "", fmt.Errorf("--settings flag not found in command line")
	}

	return matches[1], nil
}
