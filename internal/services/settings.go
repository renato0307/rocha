package services

import (
	"context"
	"fmt"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
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

// SetDebugClaude updates DebugClaude flag for a session
func (s *SettingsService) SetDebugClaude(
	ctx context.Context,
	sessionName string,
	debug bool,
) error {
	logging.Logger.Info("Setting debug Claude flag for session",
		"session", sessionName,
		"debug", debug)

	// Update in database
	if err := s.sessionRepo.UpdateDebugClaude(ctx, sessionName, debug); err != nil {
		logging.Logger.Error("Failed to update debug Claude flag",
			"session", sessionName,
			"error", err)
		return fmt.Errorf("failed to update debug Claude flag: %w", err)
	}

	logging.Logger.Info("Debug Claude flag updated successfully",
		"session", sessionName,
		"debug", debug)
	return nil
}

// GetAvailableStatuses returns the list of configured session statuses
func (s *SettingsService) GetAvailableStatuses() ([]string, error) {
	logging.Logger.Debug("Getting available statuses")

	settings, err := config.LoadSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	statusConfig := config.NewStatusConfig(
		joinStrings(settings.Statuses),
		"",
		joinStrings(settings.StatusColors),
	)

	return statusConfig.Statuses, nil
}

// GetTmuxStatusPosition returns the configured tmux status bar position with default applied
func (s *SettingsService) GetTmuxStatusPosition() string {
	logging.Logger.Debug("Getting tmux status position")

	settings, err := config.LoadSettings()
	if err != nil {
		logging.Logger.Warn("Failed to load settings, using default", "error", err)
		return config.DefaultTmuxStatusPosition
	}

	if settings.TmuxStatusPosition == "" {
		return config.DefaultTmuxStatusPosition
	}

	return settings.TmuxStatusPosition
}

// joinStrings joins a string slice with commas
func joinStrings(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		result += v
	}
	return result
}
