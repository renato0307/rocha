package tmux

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"

	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// attachmentState tracks the state of an attached session
type attachmentState struct {
	ptmx     *os.File
	attachCh chan struct{}
	mu       sync.Mutex
}

// DefaultClient is the default implementation of the Client interface
type DefaultClient struct {
	attachedSessions map[string]*attachmentState
	mu               sync.Mutex
}

// Compile-time interface verification
var _ ports.TmuxClient = (*DefaultClient)(nil)

// Local error aliases for backward compatibility within this package
var (
	ErrAlreadyAttached = ports.ErrTmuxAlreadyAttached
	ErrNotAttached     = ports.ErrTmuxNotAttached
	ErrSessionExists   = ports.ErrTmuxSessionExists
	ErrSessionNotFound = ports.ErrTmuxSessionNotFound
)

// NewClient creates a new DefaultClient instance
func NewClient() *DefaultClient {
	return &DefaultClient{
		attachedSessions: make(map[string]*attachmentState),
	}
}

// createBaseSession creates a tmux session without running rocha start-claude
// This is the common logic shared by CreateSession() and CreateShellSession()
func (c *DefaultClient) createBaseSession(name string, worktreePath string, statusPosition string) error {
	if c.SessionExists(name) {
		return ErrSessionExists
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}

	var cmd *exec.Cmd
	if worktreePath != "" {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", name, "-c", worktreePath, shell)
	} else {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", name, shell)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	if err := c.BindKey("root", "C-q", "detach-client"); err != nil {
		logging.Logger.Warn("Failed to bind Ctrl+Q key", "error", err)
	}

	// Bind Ctrl+] for swapping between Claude and shell sessions
	if err := c.bindSwapKey(); err != nil {
		logging.Logger.Warn("Failed to bind Ctrl+] swap key", "error", err)
	}

	// Set status bar position
	if statusPosition != "" {
		if err := c.SetOption(name, "status-position", statusPosition); err != nil {
			logging.Logger.Warn("Failed to set status position", "error", err)
		}
	}

	// Wait for session to be ready
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for session %s to be created", name)
		case <-ticker.C:
			if c.SessionExists(name) {
				return nil
			}
		}
	}
}

// bindSwapKey binds Ctrl+] to swap between Claude and shell sessions
func (c *DefaultClient) bindSwapKey() error {
	// Construct the bash script that will swap between sessions
	// The script:
	// 1. Gets the current session name
	// 2. Determines if it's a shell session (ends with -shell) or Claude session
	// 3. Calculates the target session name (add/remove -shell suffix)
	// 4. Switches to the target if it exists, otherwise shows an error message
	script := `current=$(tmux display-message -p "#{session_name}")
if [[ "$current" == *-shell ]]; then
    target="${current%-shell}"
    type="Claude"
else
    target="$current-shell"
    type="shell"
fi
if tmux has-session -t "$target" 2>/dev/null; then
    tmux switch-client -t "$target"
else
    tmux display-message "No $type session found for $current"
fi
`

	// Bind Ctrl+] to run the swap script
	command := fmt.Sprintf("run-shell '%s'", script)
	return c.BindKey("root", "C-]", command)
}

// CreateSession creates a new tmux session with the given name
// If worktreePath is provided (non-empty), the session will start in that directory
// If claudeDir is provided (non-empty), CLAUDE_CONFIG_DIR environment variable will be set
func (c *DefaultClient) CreateSession(name string, worktreePath string, claudeDir string, statusPosition string) (*ports.TmuxSession, error) {
	logging.Logger.Info("Creating new tmux session", "name", name, "worktree_path", worktreePath, "claude_dir", claudeDir, "status_position", statusPosition)

	if err := c.createBaseSession(name, worktreePath, statusPosition); err != nil {
		return nil, err
	}

	// Start Claude in the session
	rochaBin, err := os.Executable()
	if err != nil {
		rochaBin = "rocha"
		logging.Logger.Warn("Could not get rocha executable path, using PATH", "error", err)
	}
	logging.Logger.Debug("Rocha binary path", "path", rochaBin)

	// Set session name in environment and start claude with hooks
	envVars := fmt.Sprintf("ROCHA_SESSION_NAME=%s", name)

	// Add CLAUDE_CONFIG_DIR if specified
	if claudeDir != "" {
		envVars += fmt.Sprintf(" CLAUDE_CONFIG_DIR=%q", claudeDir)
		logging.Logger.Info("Setting CLAUDE_CONFIG_DIR for session", "claude_dir", claudeDir)
	}

	// Add debug environment variables if set
	if debugEnabled := os.Getenv("ROCHA_DEBUG"); debugEnabled == "1" {
		envVars += " ROCHA_DEBUG=1"
		if debugFile := os.Getenv("ROCHA_DEBUG_FILE"); debugFile != "" {
			envVars += fmt.Sprintf(" ROCHA_DEBUG_FILE=%q", debugFile)
		}
		if maxLogFiles := os.Getenv("ROCHA_MAX_LOG_FILES"); maxLogFiles != "" {
			envVars += fmt.Sprintf(" ROCHA_MAX_LOG_FILES=%s", maxLogFiles)
		}
	}

	// Add execution ID if set
	if execID := os.Getenv("ROCHA_EXECUTION_ID"); execID != "" {
		envVars += fmt.Sprintf(" ROCHA_EXECUTION_ID=%s", execID)
	}

	var startCmd string
	if worktreePath != "" {
		logging.Logger.Info("Starting Claude in worktree directory", "path", worktreePath)
		startCmd = fmt.Sprintf("cd %q && clear && %s %s start-claude", worktreePath, envVars, rochaBin)
	} else {
		startCmd = fmt.Sprintf("clear && %s %s start-claude", envVars, rochaBin)
	}
	logging.Logger.Debug("Sending start command to session", "command", startCmd)
	if err := c.SendKeys(name, startCmd, "Enter"); err != nil {
		logging.Logger.Error("Failed to send start command", "error", err)
	} else {
		logging.Logger.Info("Session created and Claude started", "name", name)
	}

	return &ports.TmuxSession{
		Name:      name,
		CreatedAt: time.Now(),
	}, nil
}

// CreateShellSession creates a plain shell session without rocha start-claude
func (c *DefaultClient) CreateShellSession(name string, worktreePath string, statusPosition string) (*ports.TmuxSession, error) {
	logging.Logger.Info("Creating shell tmux session", "name", name, "worktree_path", worktreePath)

	if err := c.createBaseSession(name, worktreePath, statusPosition); err != nil {
		return nil, err
	}

	logging.Logger.Info("Shell session created", "name", name)
	return &ports.TmuxSession{Name: name, CreatedAt: time.Now()}, nil
}

// SessionExists checks if the tmux session exists
func (c *DefaultClient) SessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

// ListSessions returns all active tmux sessions
func (c *DefaultClient) ListSessions() ([]*ports.TmuxSession, error) {
	cmd := exec.Command("tmux", "ls", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// Check if it's because there are no sessions (exit code 1)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// No sessions exist, this is fine
				return []*ports.TmuxSession{}, nil
			}
		}
		// Actual error
		return []*ports.TmuxSession{}, err
	}

	var sessions []*ports.TmuxSession
	lines := splitLines(string(output))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sessions = append(sessions, &ports.TmuxSession{
				Name:      line,
				CreatedAt: time.Now(), // We don't track creation time for existing sessions
			})
		}
	}

	return sessions, nil
}

// KillSession terminates the tmux session
func (c *DefaultClient) KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

// RenameSession renames a tmux session
func (c *DefaultClient) RenameSession(oldName, newName string) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("session names cannot be empty")
	}

	// Check if old session exists
	if !c.SessionExists(oldName) {
		return fmt.Errorf("session %s not found", oldName)
	}

	// Check if new name already exists
	if c.SessionExists(newName) {
		return fmt.Errorf("session %s already exists", newName)
	}

	cmd := exec.Command("tmux", "rename-session", "-t", oldName, newName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rename session: %w (output: %s)", err, string(output))
	}

	return nil
}

// Attach attaches to the tmux session. Returns a channel that will be closed when detached.
func (c *DefaultClient) Attach(sessionName string) (chan struct{}, error) {
	c.mu.Lock()
	state, exists := c.attachedSessions[sessionName]
	if !exists {
		state = &attachmentState{}
		c.attachedSessions[sessionName] = state
	}
	c.mu.Unlock()

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.attachCh != nil {
		return nil, ErrAlreadyAttached
	}

	// Start tmux attach command with PTY
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to attach to session: %w", err)
	}

	state.ptmx = ptmx
	state.attachCh = make(chan struct{})

	// Copy tmux output to stdout
	go func() {
		io.Copy(os.Stdout, ptmx)
	}()

	// Read stdin and forward to tmux, watch for Ctrl+Q (ASCII 17) to detach
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				break
			}

			// Check for Ctrl+Q (ASCII 17)
			for i := 0; i < n; i++ {
				if buf[i] == 17 { // Ctrl+Q
					c.Detach(sessionName)
					return
				}
			}

			// Forward to tmux
			ptmx.Write(buf[:n])
		}
	}()

	return state.attachCh, nil
}

// Detach detaches from the tmux session
func (c *DefaultClient) Detach(sessionName string) error {
	c.mu.Lock()
	state, exists := c.attachedSessions[sessionName]
	c.mu.Unlock()

	if !exists {
		return ErrNotAttached
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.attachCh == nil {
		return ErrNotAttached
	}

	if state.ptmx != nil {
		state.ptmx.Close()
		state.ptmx = nil
	}

	close(state.attachCh)
	state.attachCh = nil

	return nil
}

// GetAttachCommand returns an exec.Cmd configured for attaching to a session.
// This is useful for integration with frameworks like Bubble Tea's tea.ExecProcess.
// It unsets TMUX and TMUX_PANE environment variables to allow attaching from within tmux.
func (c *DefaultClient) GetAttachCommand(sessionName string) *exec.Cmd {
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)

	// Copy current environment and remove TMUX variables to allow nested attach
	env := os.Environ()
	var cleanEnv []string
	for _, e := range env {
		if !strings.HasPrefix(e, "TMUX=") && !strings.HasPrefix(e, "TMUX_PANE=") {
			cleanEnv = append(cleanEnv, e)
		}
	}
	cmd.Env = cleanEnv

	return cmd
}

// SendKeys sends keystrokes to the specified tmux session
func (c *DefaultClient) SendKeys(sessionName string, keys ...string) error {
	args := []string{"send-keys", "-t", sessionName}
	args = append(args, keys...)
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// CapturePane captures the content of the tmux pane
func (c *DefaultClient) CapturePane(sessionName string, startLine int) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-p", "-t", sessionName, "-S", fmt.Sprintf("%d", startLine))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// SourceFile sources a tmux configuration file
func (c *DefaultClient) SourceFile(configPath string) error {
	cmd := exec.Command("tmux", "source-file", configPath)
	return cmd.Run()
}

// BindKey binds a key in the specified key table
func (c *DefaultClient) BindKey(table, key, command string) error {
	cmd := exec.Command("tmux", "bind-key", "-T", table, key, command)
	return cmd.Run()
}

// SetOption sets a tmux option for a specific session
func (c *DefaultClient) SetOption(sessionName, option, value string) error {
	cmd := exec.Command("tmux", "set-option", "-t", sessionName, option, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set tmux option %s=%s for session %s: %w", option, value, sessionName, err)
	}
	return nil
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
