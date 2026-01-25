package tmux

import (
	"strings"
	"time"

	"rocha/internal/ports"
)

// Monitor watches a tmux session for Claude prompts
type Monitor struct {
	client       ports.TmuxClient
	isWaiting    bool
	lastContent  string
	notifyCh     chan string
	sessionName  string
	stopCh       chan struct{}
}

// NewMonitor creates a new session monitor
func NewMonitor(client ports.TmuxClient, sessionName string, notifyCh chan string) *Monitor {
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
	if !m.client.SessionExists(m.sessionName) {
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
