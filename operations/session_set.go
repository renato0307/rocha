package operations

import (
	"context"
	"fmt"

	"rocha/logging"
	"rocha/paths"
	"rocha/ports"
	"rocha/storage"
)

// SetSessionClaudeDir updates ClaudeDir for a single session
func SetSessionClaudeDir(
	ctx context.Context,
	sessionName string,
	claudeDir string,
	store *storage.Store,
) error {
	logging.Logger.Info("Setting ClaudeDir for session", "session", sessionName, "claudeDir", claudeDir)

	// Expand path (~ to home directory)
	expandedPath := paths.ExpandPath(claudeDir)
	logging.Logger.Debug("Path expanded", "original", claudeDir, "expanded", expandedPath)

	// Update in database
	if err := store.UpdateSessionClaudeDir(ctx, sessionName, expandedPath); err != nil {
		logging.Logger.Error("Failed to update ClaudeDir", "session", sessionName, "error", err)
		return fmt.Errorf("failed to update ClaudeDir: %w", err)
	}

	logging.Logger.Info("ClaudeDir updated successfully", "session", sessionName)
	return nil
}

// SetSessionSkipPermissions updates AllowDangerouslySkipPermissions flag for a single session
func SetSessionSkipPermissions(
	ctx context.Context,
	sessionName string,
	skipPermissions bool,
	store *storage.Store,
) error {
	logging.Logger.Info("Setting skip permissions flag for session",
		"session", sessionName,
		"skipPermissions", skipPermissions)

	// Update in database
	if err := store.UpdateSessionSkipPermissions(ctx, sessionName, skipPermissions); err != nil {
		logging.Logger.Error("Failed to update skip permissions flag",
			"session", sessionName,
			"error", err)
		return fmt.Errorf("failed to update skip permissions flag: %w", err)
	}

	logging.Logger.Info("Skip permissions flag updated successfully",
		"session", sessionName,
		"skipPermissions", skipPermissions)
	return nil
}

// GetRunningTmuxSessions returns list of tmux sessions that are currently running
func GetRunningTmuxSessions(sessionNames []string, tmuxClient ports.TmuxSessionLifecycle) ([]string, error) {
	logging.Logger.Debug("Checking for running tmux sessions", "sessions", sessionNames)

	// Get all running tmux sessions
	sessions, err := tmuxClient.ListSessions()
	if err != nil {
		// List returns error if no sessions exist
		logging.Logger.Debug("No tmux sessions running or tmux error", "error", err)
		return []string{}, nil
	}

	// Build map for quick lookup
	runningSessions := make(map[string]bool)
	for _, session := range sessions {
		runningSessions[session.Name] = true
	}

	// Filter requested sessions that are running
	var running []string
	for _, name := range sessionNames {
		if runningSessions[name] {
			running = append(running, name)
		}
		// Also check for shell session
		shellName := name + "-shell"
		if runningSessions[shellName] {
			running = append(running, shellName)
		}
	}

	logging.Logger.Debug("Found running tmux sessions", "count", len(running), "sessions", running)
	return running, nil
}
