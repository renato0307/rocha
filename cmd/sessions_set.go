package cmd

import (
	"context"
	"fmt"
	"strings"

	"rocha/logging"
	"rocha/ports"
	"rocha/tmux"
)

// SessionSetCmd sets configuration for a session
type SessionSetCmd struct {
	All      bool   `help:"Apply to all sessions" short:"a"`
	KillTmux bool   `help:"Kill tmux sessions to apply changes immediately" short:"k"`
	Name     string `arg:"" optional:"" help:"Name of the session (omit when using --all)"`
	Value    string `help:"Value to set (empty string to clear)" required:""`
	Variable string `help:"Variable to set" short:"v" enum:"claudedir,allow-dangerously-skip-permissions" required:""`
}

// AfterApply validates that either Name or All is provided, but not both
func (s *SessionSetCmd) AfterApply() error {
	hasName := s.Name != ""
	hasAll := s.All

	if hasName && hasAll {
		return fmt.Errorf("cannot specify both <name> and --all")
	}
	if !hasName && !hasAll {
		return fmt.Errorf("must specify either <name> or --all")
	}

	return nil
}

// Run executes the set command
func (s *SessionSetCmd) Run(cli *CLI) error {
	ctx := context.Background()
	logging.Logger.Info("Executing session set command",
		"name", s.Name, "variable", s.Variable, "value", s.Value, "all", s.All, "killTmux", s.KillTmux)

	tmuxClient := tmux.NewClient()
	container, err := NewContainer(tmuxClient)
	if err != nil {
		logging.Logger.Error("Failed to create container", "error", err)
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	sessionNames, err := s.getSessionNames(ctx, container)
	if err != nil {
		return err
	}

	updater, err := s.createUpdater(container)
	if err != nil {
		return err
	}

	successCount, failedSessions := updateAllSessions(ctx, sessionNames, updater)

	s.handleTmuxSessions(container.TmuxClient, sessionNames, failedSessions)

	s.printSummary(successCount, len(sessionNames))

	return nil
}

func (s *SessionSetCmd) getSessionNames(ctx context.Context, container *Container) ([]string, error) {
	if !s.All {
		logging.Logger.Debug("Updating single session", "session", s.Name)
		return []string{s.Name}, nil
	}

	logging.Logger.Info("Updating all sessions", "variable", s.Variable)
	sessions, err := container.SessionRepository.List(ctx, false)
	if err != nil {
		logging.Logger.Error("Failed to list sessions", "error", err)
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var names []string
	for _, sess := range sessions {
		names = append(names, sess.Name)
	}

	logging.Logger.Debug("Retrieved sessions to update", "count", len(names))
	fmt.Printf("Updating %s for %d sessions...\n", s.Variable, len(names))
	return names, nil
}

func (s *SessionSetCmd) createUpdater(container *Container) (sessionUpdater, error) {
	switch s.Variable {
	case "claudedir":
		return func(ctx context.Context, name string) error {
			return container.SettingsService.SetClaudeDir(ctx, name, s.Value)
		}, nil

	case "allow-dangerously-skip-permissions":
		skipPermissions, err := parseBoolValue(s.Value)
		if err != nil {
			logging.Logger.Error("Invalid boolean value", "value", s.Value, "error", err)
			return nil, fmt.Errorf("invalid value for allow-dangerously-skip-permissions: %w (use: true/false, yes/no, 1/0)", err)
		}
		return func(ctx context.Context, name string) error {
			return container.SettingsService.SetSkipPermissions(ctx, name, skipPermissions)
		}, nil

	default:
		return nil, fmt.Errorf("unknown variable type: %s", s.Variable)
	}
}

func (s *SessionSetCmd) handleTmuxSessions(tmuxClient ports.TmuxClient, sessionNames, failedSessions []string) {
	successfulSessions := filterSuccessfulSessions(sessionNames, failedSessions)

	if s.KillTmux {
		killTmuxSessions(tmuxClient, successfulSessions)
		return
	}

	restartCmd := s.buildRestartCommand()
	warnAboutRunningSessions(tmuxClient, successfulSessions, restartCmd)
}

func (s *SessionSetCmd) buildRestartCommand() string {
	if s.All {
		return fmt.Sprintf("rocha sessions set --all --variable=%s --value=%q --kill-tmux", s.Variable, s.Value)
	}
	return fmt.Sprintf("rocha sessions set %s --variable=%s --value=%q --kill-tmux", s.Name, s.Variable, s.Value)
}

func (s *SessionSetCmd) printSummary(successCount, totalCount int) {
	logging.Logger.Info("Session set command completed", "successCount", successCount, "totalCount", totalCount)
	fmt.Println()
	if successCount == totalCount {
		fmt.Printf("Updated %s for %d session(s)\n", s.Variable, successCount)
	} else {
		fmt.Printf("Updated %d of %d session(s)\n", successCount, totalCount)
	}
}

// sessionUpdater is a function that updates a single session
type sessionUpdater func(ctx context.Context, name string) error

// updateAllSessions applies an updater function to all sessions and tracks success/failures
func updateAllSessions(ctx context.Context, sessionNames []string, updater sessionUpdater) (successCount int, failedSessions []string) {
	logging.Logger.Info("Starting session updates", "count", len(sessionNames))

	for _, name := range sessionNames {
		logging.Logger.Debug("Updating session", "session", name)
		if err := updater(ctx, name); err != nil {
			logging.Logger.Warn("Failed to update session", "session", name, "error", err)
			fmt.Printf("Failed to update '%s': %v\n", name, err)
			failedSessions = append(failedSessions, name)
			continue
		}
		logging.Logger.Debug("Session updated successfully", "session", name)
		fmt.Printf("Updated '%s'\n", name)
		successCount++
	}

	logging.Logger.Info("Session updates completed", "successCount", successCount, "failedCount", len(failedSessions))
	return successCount, failedSessions
}

// filterSuccessfulSessions returns sessions that are not in the failed list
func filterSuccessfulSessions(sessionNames, failedSessions []string) []string {
	failedSet := make(map[string]bool)
	for _, name := range failedSessions {
		failedSet[name] = true
	}

	var successful []string
	for _, name := range sessionNames {
		if !failedSet[name] {
			successful = append(successful, name)
		}
	}
	return successful
}

// killTmuxSessions kills tmux sessions for the given session names
func killTmuxSessions(tmuxClient ports.TmuxSessionLifecycle, sessionNames []string) {
	logging.Logger.Info("Killing tmux sessions", "count", len(sessionNames))

	for _, name := range sessionNames {
		logging.Logger.Debug("Killing main tmux session", "session", name)
		if err := tmuxClient.KillSession(name); err != nil {
			logging.Logger.Warn("Failed to kill tmux session", "session", name, "error", err)
			fmt.Printf("Warning: Failed to kill tmux session '%s': %v\n", name, err)
		} else {
			logging.Logger.Debug("Main tmux session killed", "session", name)
			fmt.Printf("Killed tmux session '%s'\n", name)
		}

		shellName := name + "-shell"
		logging.Logger.Debug("Attempting to kill shell session", "session", shellName)
		if err := tmuxClient.KillSession(shellName); err != nil {
			logging.Logger.Debug("Shell session not found or already killed", "session", shellName)
		} else {
			logging.Logger.Debug("Shell tmux session killed", "session", shellName)
			fmt.Printf("Killed tmux session '%s'\n", shellName)
		}
	}

	logging.Logger.Info("Tmux session kills completed")
}

// warnAboutRunningSessions checks for running tmux sessions and prints a warning with restart instructions
func warnAboutRunningSessions(tmuxClient ports.TmuxSessionLifecycle, sessionNames []string, restartCmd string) {
	logging.Logger.Debug("Checking for running tmux sessions")

	// Get all running tmux sessions
	allRunningSessions, err := tmuxClient.ListSessions()
	if err != nil {
		logging.Logger.Debug("No tmux sessions running or tmux error", "error", err)
		return
	}

	// Build map of running session names
	runningMap := make(map[string]bool)
	for _, sess := range allRunningSessions {
		runningMap[sess.Name] = true
	}

	// Filter to only sessions we care about
	var runningSessions []string
	for _, name := range sessionNames {
		if runningMap[name] {
			runningSessions = append(runningSessions, name)
		}
		// Also check for shell session
		shellName := name + "-shell"
		if runningMap[shellName] {
			runningSessions = append(runningSessions, shellName)
		}
	}

	if len(runningSessions) == 0 {
		logging.Logger.Debug("No running tmux sessions found for updated sessions")
		return
	}

	logging.Logger.Info("Found running tmux sessions", "count", len(runningSessions), "sessions", runningSessions)
	fmt.Println()
	fmt.Printf("Warning: %d tmux session(s) are still running and need restart:\n", len(runningSessions))
	for _, name := range runningSessions {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()
	fmt.Println("Restart them to apply changes:")
	fmt.Printf("  %s\n", restartCmd)
	fmt.Println()
	fmt.Println("Or manually:")
	for _, name := range runningSessions {
		fmt.Printf("  tmux kill-session -t %s\n", name)
	}
}

// parseBoolValue parses a boolean value from various string formats
func parseBoolValue(value string) (bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))

	switch normalized {
	case "true", "yes", "1":
		return true, nil
	case "false", "no", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %q", value)
	}
}
