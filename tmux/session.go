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

// Session represents a tmux session
type Session struct {
	Name      string
	CreatedAt time.Time
	ptmx      *os.File
	attached  bool
	attachCh  chan struct{}
	mu        sync.Mutex
}

// NewSession creates a new tmux session with the given name
func NewSession(name string) (*Session, error) {
	logging.Logger.Info("Creating new tmux session", "name", name)

	s := &Session{
		Name:      name,
		CreatedAt: time.Now(),
	}

	// Check if session already exists
	if s.Exists() {
		logging.Logger.Warn("Session already exists", "name", name)
		return nil, fmt.Errorf("session %s already exists", name)
	}

	// Create a new detached tmux session starting with a shell
	// This ensures the session stays alive even when you detach
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	logging.Logger.Debug("Creating tmux session with shell", "shell", shell)

	cmd := exec.Command("tmux", "new-session", "-d", "-s", name, shell)
	if err := cmd.Run(); err != nil {
		logging.Logger.Error("Failed to create tmux session", "error", err, "name", name)
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Bind Ctrl+Q as an additional detach shortcut for this session (easier to use than Ctrl+B D)
	bindCmd := exec.Command("tmux", "bind-key", "-T", "root", "C-q", "detach-client")
	_ = bindCmd.Run() // Ignore errors - this sets it globally which is fine

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
			if s.Exists() {
				logging.Logger.Debug("Tmux session ready", "name", name)
				// Session created successfully
				// Now automatically start claude with hooks in the session
				// Get path to rocha binary
				rochaBin, err := os.Executable()
				if err != nil {
					rochaBin = "rocha" // Fallback to PATH
					logging.Logger.Warn("Could not get rocha executable path, using PATH", "error", err)
				}
				logging.Logger.Debug("Rocha binary path", "path", rochaBin)

				// Set session name in environment and start claude with hooks
				// Clear the screen first to hide the command being typed
				startCmd := fmt.Sprintf("clear && ROCHA_SESSION_NAME=%s %s start-claude", name, rochaBin)
				logging.Logger.Debug("Sending start command to session", "command", startCmd)
				sendCmd := exec.Command("tmux", "send-keys", "-t", name, startCmd, "Enter")
				if err := sendCmd.Run(); err != nil {
					logging.Logger.Error("Failed to send start command", "error", err)
				} else {
					logging.Logger.Info("Session created and Claude started", "name", name)
				}
				return s, nil
			}
		}
	}
}

// Exists checks if the tmux session exists
func (s *Session) Exists() bool {
	cmd := exec.Command("tmux", "has-session", "-t="+s.Name)
	return cmd.Run() == nil
}

// Attach attaches to the tmux session. Returns a channel that will be closed when detached.
func (s *Session) Attach() (chan struct{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.attached {
		return nil, fmt.Errorf("already attached")
	}

	// Start tmux attach command with PTY
	cmd := exec.Command("tmux", "attach-session", "-t", s.Name)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to attach to session: %w", err)
	}

	s.ptmx = ptmx
	s.attached = true
	s.attachCh = make(chan struct{})

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
					s.Detach()
					return
				}
			}

			// Forward to tmux
			ptmx.Write(buf[:n])
		}
	}()

	return s.attachCh, nil
}

// Detach detaches from the tmux session
func (s *Session) Detach() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.attached {
		return nil
	}

	if s.ptmx != nil {
		s.ptmx.Close()
		s.ptmx = nil
	}

	s.attached = false
	close(s.attachCh)
	s.attachCh = nil

	return nil
}

// Kill terminates the tmux session
func (s *Session) Kill() error {
	cmd := exec.Command("tmux", "kill-session", "-t", s.Name)
	return cmd.Run()
}

// List returns all active tmux sessions
func List() ([]*Session, error) {
	cmd := exec.Command("tmux", "ls", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// Check if it's because there are no sessions (exit code 1)
		// vs an actual error
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
