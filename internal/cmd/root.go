package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ui"
)

// CLI represents the command-line interface structure
type CLI struct {
	Version     kong.VersionFlag `help:"Show version information"`
	Debug       bool             `help:"Enable debug logging to file" short:"d"`
	DebugFile   string           `help:"Custom path for debug log file (disables automatic cleanup)"`
	MaxLogFiles int              `help:"Maximum number of log files to keep (0 = unlimited)" default:"1000"`

	Run         RunCmd         `cmd:"" help:"Start the rocha TUI (default)" default:"1"`
	Setup       SetupCmd       `cmd:"setup" help:"Configure tmux status bar integration automatically"`
	Stats       StatsCmd       `cmd:"stats" help:"Show token usage statistics"`
	Hooks       HooksCmd       `cmd:"hooks" help:"View Claude Code hook events"`
	Status      StatusCmd      `cmd:"status" help:"Show session state counts for tmux status bar" hidden:""`
	Attach      AttachCmd      `cmd:"attach" help:"Attach to tmux session (creates if needed)"`
	StartClaude StartClaudeCmd `cmd:"start-claude" help:"Start Claude Code with hooks configured" hidden:""`
	PlaySound   PlaySoundCmd   `cmd:"play-sound" help:"Play notification sound (cross-platform)" hidden:""`
	Notify      NotifyCmd      `cmd:"notify" help:"Handle notification event from Claude hooks" hidden:""`
	Sessions    SessionsCmd    `cmd:"sessions" help:"Manage sessions (list, view, add, del)"`
	Settings    SettingsCmd    `cmd:"settings" help:"Manage settings (meta)"`

	// Internal fields (not flags)
	Container *Container       `kong:"-"`
	settings  *config.Settings `kong:"-"`
}

// SetSettings sets the settings on the CLI struct
func (c *CLI) SetSettings(settings *config.Settings) {
	c.settings = settings
}

// AfterApply initializes logging after CLI parsing and applies settings
func (c *CLI) AfterApply() error {
	// Apply settings with proper precedence: CLI flags > env vars > settings.json > defaults
	// Only apply if flag is at default value and env var is not set

	if c.settings != nil {
		// Apply MaxLogFiles setting
		if c.MaxLogFiles == 1000 {
			if _, hasEnv := os.LookupEnv("ROCHA_MAX_LOG_FILES"); !hasEnv {
				if c.settings.MaxLogFiles != nil {
					c.MaxLogFiles = *c.settings.MaxLogFiles
				}
			}
		}

		// Apply Debug setting
		if !c.Debug {
			if _, hasEnv := os.LookupEnv("ROCHA_DEBUG"); !hasEnv {
				if c.settings.Debug != nil && *c.settings.Debug {
					c.Debug = true
				}
			}
		}
	}

	// Initialize logging first and get the log file path
	logFilePath, err := logging.Initialize(c.Debug, c.DebugFile, c.MaxLogFiles)
	if err != nil {
		return err
	}

	// Set environment variables AFTER initialization so child processes inherit debug settings
	// and use the SAME log file (important for correlating parent/child process logs)
	if c.Debug || c.DebugFile != "" {
		os.Setenv("ROCHA_DEBUG", "1")
		// Share the log file path with subprocesses so they append to the same file
		if logFilePath != "" {
			os.Setenv("ROCHA_DEBUG_FILE", logFilePath)
		}
	}
	if c.MaxLogFiles != 1000 {
		os.Setenv("ROCHA_MAX_LOG_FILES", fmt.Sprintf("%d", c.MaxLogFiles))
	}

	// Create container AFTER logging is initialized
	// This fixes the nil pointer panic when GORM's logger calls logging.Logger.Debug()
	container, err := NewContainer()
	if err != nil {
		return fmt.Errorf("failed to initialize container: %w", err)
	}
	c.Container = container

	return nil
}

// Close closes all resources held by the CLI
func (c *CLI) Close() error {
	if c.Container != nil {
		return c.Container.Close()
	}
	return nil
}

// RunCmd starts the TUI application
type RunCmd struct {
	Dev                        bool   `help:"Enable development mode (shows version info in dialogs)"`
	Editor                     string `help:"Editor to open sessions in (overrides $ROCHA_EDITOR, $VISUAL, $EDITOR)" default:"code"`
	ErrorClearDelay            int    `help:"Seconds before error messages auto-clear" default:"10"`
	ShowTimestamps             bool   `help:"Show relative timestamps for last state changes" default:"false"`
	ShowTokenChart             bool   `help:"Show token usage chart by default" default:"false"`
	StatusColors               string `help:"Comma-separated ANSI color codes for statuses (e.g., '141,33,214,226,46')" default:"141,33,214,226,46"`
	StatusIcons                string `help:"Comma-separated status icons (optional, colors are used for display)" default:""`
	Statuses                   string `help:"Comma-separated status names (e.g., 'spec,plan,implement,review,done')" default:"spec,plan,implement,review,done"`
	TimestampRecentColor       string `help:"ANSI color code for recent timestamps" default:"241"`
	TimestampRecentMinutes     int    `help:"Minutes threshold for recent timestamps (gray color)" default:"5"`
	TimestampStaleColor        string `help:"ANSI color code for stale timestamps (matches waiting state ◐)" default:"1"`
	TimestampWarningColor      string `help:"ANSI color code for warning timestamps (matches idle state ○)" default:"3"`
	TimestampWarningMinutes    int    `help:"Minutes threshold for warning timestamps (yellow color)" default:"20"`
	TipsDisplayDurationSeconds int    `help:"Seconds to display each tip" default:"90"`
	TipsEnabled                bool   `help:"Enable rotating tips display" default:"true"`
	TipsShowIntervalSeconds    int    `help:"Seconds between tips" default:"2"`
	TmuxStatusPosition         string `help:"Tmux status bar position (top or bottom)" default:"bottom" enum:"top,bottom"`
}

// Run executes the TUI
func (r *RunCmd) Run(cli *CLI) error {
	// Apply RunCmd-specific settings with proper precedence
	// Only apply if flag is at default value and env var is not set

	if cli.settings != nil {
		// Apply Editor setting
		if r.Editor == "code" {
			if _, hasEnv := os.LookupEnv("ROCHA_EDITOR"); !hasEnv {
				if cli.settings.Editor != "" {
					r.Editor = cli.settings.Editor
				}
			}
		}

		// Apply ErrorClearDelay setting
		if r.ErrorClearDelay == 10 {
			if cli.settings.ErrorClearDelay != nil {
				r.ErrorClearDelay = *cli.settings.ErrorClearDelay
			}
		}

		// Apply Statuses setting
		if r.Statuses == "spec,plan,implement,review,done" {
			if len(cli.settings.Statuses) > 0 {
				// Convert StringArray to comma-separated string
				r.Statuses = strings.Join(cli.settings.Statuses, ",")
			}
		}

		// Apply StatusColors setting
		if r.StatusColors == "141,33,214,226,46" {
			if len(cli.settings.StatusColors) > 0 {
				// Convert StringArray to comma-separated string
				r.StatusColors = strings.Join(cli.settings.StatusColors, ",")
			}
		}

		// Apply ShowTimestamps setting
		if !r.ShowTimestamps {
			if _, hasEnv := os.LookupEnv("ROCHA_SHOW_TIMESTAMPS"); !hasEnv {
				if cli.settings.ShowTimestamps != nil && *cli.settings.ShowTimestamps {
					r.ShowTimestamps = true
				}
			}
		}

		// Apply TmuxStatusPosition setting
		if r.TmuxStatusPosition == config.DefaultTmuxStatusPosition {
			if _, hasEnv := os.LookupEnv("ROCHA_TMUX_STATUS_POSITION"); !hasEnv {
				if cli.settings.TmuxStatusPosition != "" {
					r.TmuxStatusPosition = cli.settings.TmuxStatusPosition
				}
			}
		}

		// Apply Tips settings
		if r.TipsEnabled {
			if cli.settings.TipsEnabled != nil && !*cli.settings.TipsEnabled {
				r.TipsEnabled = false
			}
		}
		if r.TipsDisplayDurationSeconds == 90 {
			if cli.settings.TipsDisplayDurationSeconds != nil {
				r.TipsDisplayDurationSeconds = *cli.settings.TipsDisplayDurationSeconds
			}
		}
		if r.TipsShowIntervalSeconds == 2 {
			if cli.settings.TipsShowIntervalSeconds != nil {
				r.TipsShowIntervalSeconds = *cli.settings.TipsShowIntervalSeconds
			}
		}

		// Apply ShowTokenChart setting
		if !r.ShowTokenChart {
			if cli.settings.ShowTokenChart != nil && *cli.settings.ShowTokenChart {
				r.ShowTokenChart = true
			}
		}
	}

	logging.Logger.Info("Starting rocha TUI")

	// Generate new execution ID for this TUI run
	executionID := uuid.New().String()
	// Set environment variable for child processes (including tmux sessions)
	// Hooks will also receive it explicitly via --execution-id flag
	os.Setenv("ROCHA_EXECUTION_ID", executionID)
	logging.Logger.Info("Generated new execution ID", "execution_id", executionID)

	// Load state for initial session info
	st, err := cli.Container.SessionService.LoadState(context.Background(), false)
	if err != nil {
		log.Printf("Warning: failed to load session state: %v", err)
		logging.Logger.Warn("Failed to load session state", "error", err)
		st = &domain.SessionCollection{Sessions: make(map[string]domain.Session)}
	}
	logging.Logger.Debug("State loaded", "existing_sessions", len(st.Sessions))

	// Sync with running tmux sessions
	runningSessions, err := cli.Container.SessionService.ListTmuxSessions()
	if err != nil {
		logging.Logger.Warn("Failed to list tmux sessions", "error", err)
	} else {
		runningNames := make([]string, len(runningSessions))
		for i, sess := range runningSessions {
			runningNames[i] = sess.Name
		}
		logging.Logger.Info("Syncing with running tmux sessions", "count", len(runningNames))

		// Update execution ID for running sessions without changing last_updated timestamp
		for _, sessionName := range runningNames {
			if _, exists := st.Sessions[sessionName]; exists {
				if err := cli.Container.SessionService.UpdateState(context.Background(), sessionName, domain.StateIdle, executionID); err != nil {
					logging.Logger.Error("Failed to update execution ID", "error", err, "session", sessionName)
				} else {
					logging.Logger.Debug("Updated session execution ID", "session", sessionName)
				}
			}
		}

	}

	// Extract allow dangerously skip permissions default from settings
	allowDangerouslySkipPermissionsDefault := false
	if cli.settings != nil && cli.settings.AllowDangerouslySkipPermissions != nil {
		allowDangerouslySkipPermissionsDefault = *cli.settings.AllowDangerouslySkipPermissions
	}
	logging.Logger.Debug("Allow dangerously skip permissions default from settings",
		"value", allowDangerouslySkipPermissionsDefault)

	// Validate key bindings if configured
	var keysConfig config.KeyBindingsConfig
	if cli.settings != nil && cli.settings.Keys != nil {
		if err := cli.settings.Keys.Validate(ui.GetValidKeyNames()); err != nil {
			return fmt.Errorf("invalid key bindings in settings.json: %w", err)
		}
		keysConfig = cli.settings.Keys
		logging.Logger.Debug("Custom key bindings loaded and validated")
	}

	// Set terminal to raw mode for proper input handling
	logging.Logger.Debug("Initializing Bubble Tea program")
	errorClearDelay := time.Duration(r.ErrorClearDelay) * time.Second
	statusConfig := config.NewStatusConfig(r.Statuses, r.StatusIcons, r.StatusColors)
	timestampConfig := config.NewTimestampColorConfig(
		r.TimestampRecentMinutes,
		r.TimestampWarningMinutes,
		r.TimestampRecentColor,
		r.TimestampWarningColor,
		r.TimestampStaleColor,
	)
	tipsConfig := ui.TipsConfig{
		DisplayDurationSeconds: r.TipsDisplayDurationSeconds,
		Enabled:                r.TipsEnabled,
		ShowIntervalSeconds:    r.TipsShowIntervalSeconds,
	}
	p := tea.NewProgram(
		ui.NewModel(
			r.Editor,
			errorClearDelay,
			statusConfig,
			timestampConfig,
			r.Dev,
			r.ShowTimestamps,
			r.ShowTokenChart,
			r.TmuxStatusPosition,
			allowDangerouslySkipPermissionsDefault,
			tipsConfig,
			keysConfig,
			cli.Container.GitService,
			cli.Container.SessionService,
			cli.Container.ShellService,
			cli.Container.TokenStatsService,
		),
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	logging.Logger.Info("Starting TUI program")
	if _, err := p.Run(); err != nil {
		logging.Logger.Error("TUI program error", "error", err)
		return fmt.Errorf("error running program: %w", err)
	}

	logging.Logger.Info("TUI program exited normally")
	return nil
}
