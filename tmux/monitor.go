package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Monitor watches a tmux session for Claude prompts
type Monitor struct {
	client       Client
	sessionName  string
	stopCh       chan struct{}
	notifyCh     chan string
	lastContent  string
	isWaiting    bool
}

// NewMonitor creates a new session monitor
func NewMonitor(client Client, sessionName string, notifyCh chan string) *Monitor {
	return &Monitor{
		client:      client,
		sessionName: sessionName,
		stopCh:      make(chan struct{}),
		notifyCh:    notifyCh,
		isWaiting:   false,
	}
}

// Start begins monitoring the session
func (m *Monitor) Start() {
	go m.monitorLoop()
}

// Stop halts the monitoring
func (m *Monitor) Stop() {
	close(m.stopCh)
}

// monitorLoop runs in the background checking for Claude prompts
func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkForPrompt()
		}
	}
}

// checkForPrompt captures the pane content and looks for Claude prompts
func (m *Monitor) checkForPrompt() {
	if !m.client.Exists(m.sessionName) {
		return
	}

	content, err := m.capturePane()
	if err != nil {
		return
	}

	// Check if content has changed
	if content == m.lastContent {
		return
	}
	m.lastContent = content

	// Check if Claude is waiting for input
	// Look for common Claude Code prompts
	isWaitingNow := m.detectPrompt(content)

	// If state changed from running to waiting, send notification
	if !m.isWaiting && isWaitingNow {
		m.isWaiting = true
		// Send notification with session name
		select {
		case m.notifyCh <- m.sessionName:
		default:
			// Channel full, skip notification
		}
	} else if m.isWaiting && !isWaitingNow {
		// Claude started working again
		m.isWaiting = false
	}
}

// capturePane gets the current content of the tmux pane
func (m *Monitor) capturePane() (string, error) {
	return m.client.CapturePane(m.sessionName, -30)
}

// detectPrompt checks if the content shows Claude waiting for input
func (m *Monitor) detectPrompt(content string) bool {
	// Look for common Claude Code prompt patterns
	prompts := []string{
		"Press Enter to continue",
		"would you like me to",
		"should i",
		"what would you like",
		"how can i help",
		"?", // Generic question mark on last line
	}

	// Get the last few lines (where prompts typically appear)
	lines := strings.Split(content, "\n")
	lastLines := ""
	if len(lines) > 5 {
		lastLines = strings.Join(lines[len(lines)-5:], "\n")
	} else {
		lastLines = content
	}
	lastLines = strings.ToLower(lastLines)

	for _, prompt := range prompts {
		if strings.Contains(lastLines, prompt) {
			return true
		}
	}

	return false
}

// PlaySound plays a notification sound
func PlaySound() {
	// Try different methods to play sound
	// Method 1: paplay (PulseAudio)
	if _, err := exec.LookPath("paplay"); err == nil {
		cmd := exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/complete.oga")
		if err := cmd.Run(); err == nil {
			return
		}
	}

	// Method 2: aplay (ALSA)
	if _, err := exec.LookPath("aplay"); err == nil {
		cmd := exec.Command("aplay", "/usr/share/sounds/freedesktop/stereo/complete.wav")
		if err := cmd.Run(); err == nil {
			return
		}
	}

	// Method 3: Terminal bell
	fmt.Fprintf(os.Stderr, "\a")
}
