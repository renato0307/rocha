package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rocha/logging"
	"rocha/storage"
	"rocha/tmux"
	"rocha/ui"

	"github.com/charmbracelet/ssh"
	tea "github.com/charmbracelet/bubbletea"
)

// sessionModel wraps ui.Model to handle resource cleanup
type sessionModel struct {
	*ui.Model
	sessionID string
	startTime time.Time
	store     *storage.Store
}

func (s *sessionModel) Init() tea.Cmd {
	return s.Model.Init()
}

func (s *sessionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for quit message to trigger cleanup
	if _, ok := msg.(tea.QuitMsg); ok {
		// Calculate session duration
		duration := time.Since(s.startTime)

		// Close store before quitting
		if err := s.store.Close(); err != nil {
			logging.Logger.Error("Failed to close store for SSH session",
				"error", err,
				"session_id", s.sessionID,
				"duration", duration.String())
		} else {
			logging.Logger.Debug("Closed store for SSH session",
				"session_id", s.sessionID,
				"duration", duration.String())
		}

		// Log session end
		logging.Logger.Info("SSH session ended",
			"session_id", s.sessionID,
			"duration", duration.String())
	}

	updatedModel, cmd := s.Model.Update(msg)
	if m, ok := updatedModel.(*ui.Model); ok {
		s.Model = m
	}
	return s, cmd
}

func (s *sessionModel) View() string {
	return s.Model.View()
}

// teaHandler creates a Bubbletea model for each SSH session
func (s *Server) teaHandler(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
	// Get PTY info
	pty, _, _ := sess.Pty()
	sessionID := fmt.Sprintf("%s@%s", sess.User(), sess.RemoteAddr().String())

	logging.Logger.Info("New SSH session",
		"session_id", sessionID,
		"user", sess.User(),
		"remote_addr", sess.RemoteAddr().String(),
		"term", pty.Term,
		"window", fmt.Sprintf("%dx%d", pty.Window.Width, pty.Window.Height))

	// Open shared database
	store, err := storage.NewStore(s.dbPath)
	if err != nil {
		logging.Logger.Error("Failed to open database for SSH session",
			"error", err,
			"session_id", sessionID)
		return errorModel{err}, nil
	}

	// Create tmux client
	tmuxClient := tmux.NewClient()

	// Expand worktree path
	worktreePath := s.settings.WorktreePath
	if worktreePath == "" {
		worktreePath = "~/.rocha/worktrees"
	}
	worktreePath = expandPath(worktreePath)

	// Get editor
	editor := s.settings.Editor
	if editor == "" {
		editor = "code"
	}

	// Get error clear delay
	errorClearDelay := 10 * time.Second
	if s.settings.ErrorClearDelay != nil {
		errorClearDelay = time.Duration(*s.settings.ErrorClearDelay) * time.Second
	}

	// Get show timestamps
	showTimestamps := false
	if s.settings.ShowTimestamps != nil {
		showTimestamps = *s.settings.ShowTimestamps
	}

	// Get tmux status position
	tmuxStatusPosition := tmux.DefaultStatusPosition
	if s.settings.TmuxStatusPosition != "" {
		tmuxStatusPosition = s.settings.TmuxStatusPosition
	}

	// Build status config
	statuses := []string{"spec", "plan", "implement", "review", "done"}
	if len(s.settings.Statuses) > 0 {
		statuses = s.settings.Statuses
	}

	statusColors := []string{"141", "33", "214", "226", "46"}
	if len(s.settings.StatusColors) > 0 {
		statusColors = s.settings.StatusColors
	}

	statusConfig := ui.NewStatusConfig(
		joinStrings(statuses, ","),
		"", // No status icons by default
		joinStrings(statusColors, ","),
	)

	// Build timestamp config with defaults
	timestampConfig := ui.NewTimestampColorConfig(
		5,     // recentMinutes
		20,    // warningMinutes
		"241", // recentColor
		"3",   // warningColor
		"1",   // staleColor
	)

	// Build model config
	cfg := ui.ModelConfig{
		DevMode:            false, // SSH mode never uses dev mode
		Editor:             editor,
		ErrorClearDelay:    errorClearDelay,
		ShowTimestamps:     showTimestamps,
		StatusConfig:       statusConfig,
		Store:              store,
		TimestampConfig:    timestampConfig,
		TmuxClient:         tmuxClient,
		TmuxStatusPosition: tmuxStatusPosition,
		WorktreePath:       worktreePath,
	}

	// Create model using shared factory
	model := ui.NewModelFromConfig(cfg)

	// Wrap model to handle cleanup
	wrappedModel := &sessionModel{
		Model:     model,
		sessionID: sessionID,
		startTime: time.Now(),
		store:     store,
	}

	return wrappedModel, []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	}
}

// errorModel is a simple model that displays an error
type errorModel struct {
	err error
}

func (e errorModel) Init() tea.Cmd {
	return nil
}

func (e errorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return e, tea.Quit
}

func (e errorModel) View() string {
	return fmt.Sprintf("Error: %v\n", e.err)
}

// expandPath expands ~ to home directory in paths
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(path) == 1 {
			return homeDir
		}
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

// joinStrings joins string slice with separator
func joinStrings(strs []string, sep string) string {
	var result strings.Builder
	for i, s := range strs {
		if i > 0 {
			result.WriteString(sep)
		}
		result.WriteString(s)
	}
	return result.String()
}
