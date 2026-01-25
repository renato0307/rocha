package application

import (
	"context"
	"fmt"

	"rocha/internal/logging"
	"rocha/internal/config"
	"rocha/internal/ports"
)

// SettingsService handles session settings updates
type SettingsService struct {
	sessionRepo ports.SessionStateUpdater
}

// NewSettingsService creates a new SettingsService
func NewSettingsService(sessionRepo ports.SessionStateUpdater) *SettingsService {
	return &SettingsService{
		sessionRepo: sessionRepo,
	}
}

// SetClaudeDir updates ClaudeDir for a session
func (s *SettingsService) SetClaudeDir(
	ctx context.Context,
	sessionName string,
	claudeDir string,
) error {
	logging.Logger.Info("Setting ClaudeDir for session", "session", sessionName, "claudeDir", claudeDir)

	// Expand path (~ to home directory)
	expandedPath := config.ExpandPath(claudeDir)
	logging.Logger.Debug("Path expanded", "original", claudeDir, "expanded", expandedPath)

	// Update in database
	if err := s.sessionRepo.UpdateClaudeDir(ctx, sessionName, expandedPath); err != nil {
		logging.Logger.Error("Failed to update ClaudeDir", "session", sessionName, "error", err)
		return fmt.Errorf("failed to update ClaudeDir: %w", err)
	}

	logging.Logger.Info("ClaudeDir updated successfully", "session", sessionName)
	return nil
}

// SetSkipPermissions updates AllowDangerouslySkipPermissions flag for a session
func (s *SettingsService) SetSkipPermissions(
	ctx context.Context,
	sessionName string,
	skipPermissions bool,
) error {
	logging.Logger.Info("Setting skip permissions flag for session",
		"session", sessionName,
		"skipPermissions", skipPermissions)

	// Update in database
	if err := s.sessionRepo.UpdateSkipPermissions(ctx, sessionName, skipPermissions); err != nil {
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
