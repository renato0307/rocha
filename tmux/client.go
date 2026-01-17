package tmux

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"rocha/logging"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
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
var _ Client = (*DefaultClient)(nil)

// NewClient creates a new DefaultClient instance
func NewClient() *DefaultClient {
	return &DefaultClient{
		attachedSessions: make(map[string]*attachmentState),
	}
}

// Create creates a new tmux session with the given name
// If worktreePath is provided (non-empty), the session will start in that directory
func (c *DefaultClient) Create(name string, worktreePath string) (*Session, error) {
	logging.Logger.Info("Creating new tmux session", "name", name, "worktree_path", worktreePath)

	// Check if session already exists
	if c.Exists(name) {
		logging.Logger.Warn("Session already exists", "name", name)
		return nil, ErrSessionExists
	}

	// Create a new detached tmux session starting with a shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	logging.Logger.Debug("Creating tmux session with shell", "shell", shell)

	// Build tmux command with optional working directory
	var cmd *exec.Cmd
	if worktreePath != "" {
		logging.Logger.Info("Starting session in worktree directory", "path", worktreePath)
		cmd = exec.Command("tmux", "new-session", "-d", "-s", name, "-c", worktreePath, shell)
	} else {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", name, shell)
	}

	if err := cmd.Run(); err != nil {
		logging.Logger.Error("Failed to create tmux session", "error", err, "name", name)
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Bind Ctrl+Q as an additional detach shortcut for this session
	if err := c.BindKey("root", "C-q", "detach-client"); err != nil {
		logging.Logger.Warn("Failed to bind Ctrl+Q key", "error", err)
	}

	logging.Logger.Debug("Waiting for tmux session to be ready")
	// Wait for session to be created
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			logging.Logger.Error("Timeout waiting for session", "name", name)
			return nil, fmt.Errorf("timeout waiting for session %s to be created", name)
		case <-ticker.C:
			if c.Exists(name) {
				logging.Logger.Debug("Tmux session ready", "name", name)
				// Session created successfully
				// Now automatically start claude with hooks in the session
				rochaBin, err := os.Executable()
				if err != nil {
					rochaBin = "rocha" // Fallback to PATH
					logging.Logger.Warn("Could not get rocha executable path, using PATH", "error", err)
				}
				logging.Logger.Debug("Rocha binary path", "path", rochaBin)

				// Set session name in environment and start claude with hooks
				envVars := fmt.Sprintf("ROCHA_SESSION_NAME=%s", name)

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
					// Explicitly cd to worktree directory before starting Claude
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
				return &Session{
					Name:      name,
					CreatedAt: time.Now(),
				}, nil
			}
		}
	}
}

// Exists checks if the tmux session exists
func (c *DefaultClient) Exists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

// List returns all active tmux sessions
func (c *DefaultClient) List() ([]*Session, error) {
	cmd := exec.Command("tmux", "ls", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// Check if it's because there are no sessions (exit code 1)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// No sessions exist, this is fine
				return []*Session{}, nil
			}
		}
		// Actual error
		return []*Session{}, err
	}

	var sessions []*Session
	lines := splitLines(string(output))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sessions = append(sessions, &Session{
				Name:      line,
				CreatedAt: time.Now(), // We don't track creation time for existing sessions
			})
		}
	}

	return sessions, nil
}

// Kill terminates the tmux session
func (c *DefaultClient) Kill(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

// Rename renames a tmux session
func (c *DefaultClient) Rename(oldName, newName string) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("session names cannot be empty")
	}

	// Check if old session exists
	if !c.Exists(oldName) {
		return fmt.Errorf("session %s not found", oldName)
	}

	// Check if new name already exists
	if c.Exists(newName) {
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
func (c *DefaultClient) GetAttachCommand(sessionName string) *exec.Cmd {
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
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
