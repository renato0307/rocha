package services

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// ShellService handles shell session management and tmux pane operations
type ShellService struct {
	editorOpener  ports.EditorOpener
	sessionReader ports.SessionReader
	sessionWriter ports.SessionWriter
	tmuxClient    ports.TmuxClient
}

// NewShellService creates a new ShellService
func NewShellService(
	sessionReader ports.SessionReader,
	sessionWriter ports.SessionWriter,
	tmuxClient ports.TmuxClient,
	editorOpener ports.EditorOpener,
) *ShellService {
	return &ShellService{
		editorOpener:  editorOpener,
		sessionReader: sessionReader,
		sessionWriter: sessionWriter,
		tmuxClient:    tmuxClient,
	}
}

// GetOrCreateShellSession returns shell session name, creating if needed
// Returns empty string and error if operation fails
func (s *ShellService) GetOrCreateShellSession(
	ctx context.Context,
	parentSessionName string,
	tmuxStatusPosition string,
) (string, error) {
	// Get parent session info
	session, err := s.sessionReader.Get(ctx, parentSessionName)
	if err != nil {
		return "", fmt.Errorf("session info not found: %s: %w", parentSessionName, err)
	}

	// Check if shell session already exists
	if session.ShellSession != nil {
		// Check if tmux session exists
		if s.tmuxClient.SessionExists(session.ShellSession.Name) {
			return session.ShellSession.Name, nil
		}
	}

	// Create shell session name
	shellSessionName := fmt.Sprintf("%s-shell", parentSessionName)

	// Determine working directory
	workingDir := session.WorktreePath
	if workingDir == "" {
		workingDir = session.RepoPath
	}

	// Create shell session in tmux
	_, err = s.tmuxClient.CreateShellSession(shellSessionName, workingDir, tmuxStatusPosition)
	if err != nil {
		return "", fmt.Errorf("failed to create shell session: %w", err)
	}

	// Create domain session for shell
	shellSession := domain.Session{
		BranchName:   session.BranchName,
		DisplayName:  "", // No display name for shell sessions
		ExecutionID:  session.ExecutionID,
		LastUpdated:  time.Now().UTC(),
		Name:         shellSessionName,
		RepoInfo:     session.RepoInfo,
		RepoPath:     session.RepoPath,
		ShellSession: nil, // Shell sessions don't have their own shells
		State:        domain.StateIdle,
		WorktreePath: session.WorktreePath,
	}

	// Add shell session to repository
	if err := s.sessionWriter.Add(ctx, shellSession); err != nil {
		logging.Logger.Warn("Failed to save shell session to state", "error", err)
		// Don't return error - tmux session was created successfully
	} else {
		// Link shell session to parent
		if err := s.sessionWriter.LinkShellSession(ctx, parentSessionName, shellSessionName); err != nil {
			logging.Logger.Warn("Failed to link shell session to parent", "error", err)
		}
	}

	logging.Logger.Info("Shell session created", "name", shellSessionName, "parent", parentSessionName)
	return shellSessionName, nil
}

// GetRunningTmuxSessions returns a map of session names that are currently running in tmux
func (s *ShellService) GetRunningTmuxSessions(ctx context.Context) (map[string]bool, error) {
	logging.Logger.Debug("Getting running tmux sessions")

	// Get all running tmux sessions
	sessions, err := s.tmuxClient.ListSessions()
	if err != nil {
		// List returns error if no sessions exist
		logging.Logger.Debug("No tmux sessions running or tmux error", "error", err)
		return make(map[string]bool), nil
	}

	// Build map for quick lookup
	runningSessions := make(map[string]bool)
	for _, session := range sessions {
		runningSessions[session.Name] = true
	}

	logging.Logger.Debug("Found running tmux sessions", "count", len(runningSessions))
	return runningSessions, nil
}

// SendKeys sends keys to a tmux session
func (s *ShellService) SendKeys(sessionName string, keys ...string) error {
	logging.Logger.Debug("Sending keys to tmux session", "session", sessionName)
	return s.tmuxClient.SendKeys(sessionName, keys...)
}

// OpenEditor opens the specified path in the configured editor
func (s *ShellService) OpenEditor(path, editor string) error {
	logging.Logger.Debug("Opening editor", "path", path, "editor", editor)
	return s.editorOpener.Open(path, editor)
}

// SourceFile reloads tmux configuration from the specified file
func (s *ShellService) SourceFile(configPath string) error {
	logging.Logger.Debug("Sourcing tmux config file", "path", configPath)
	return s.tmuxClient.SourceFile(configPath)
}

// GetAttachCommand returns an exec.Cmd configured for attaching to a tmux session
func (s *ShellService) GetAttachCommand(sessionName string) *exec.Cmd {
	logging.Logger.Debug("Getting attach command for session", "session", sessionName)
	return s.tmuxClient.GetAttachCommand(sessionName)
}

// CapturePane captures the content of a tmux session pane
// lines specifies how many lines to capture (negative means from end of scrollback)
func (s *ShellService) CapturePane(sessionName string, lines int) (string, error) {
	logging.Logger.Debug("Capturing pane content", "session", sessionName, "lines", lines)
	return s.tmuxClient.CapturePane(sessionName, -lines)
}
