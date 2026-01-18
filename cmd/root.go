package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"rocha/config"
	"rocha/logging"
	"rocha/state"
	"rocha/storage"
	"rocha/tmux"
	"rocha/ui"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
)

// CLI represents the command-line interface structure
type CLI struct {
	Version      kong.VersionFlag `help:"Show version information"`
	Debug        bool             `help:"Enable debug logging to file" short:"d"`
	DebugFile    string           `help:"Custom path for debug log file (disables automatic cleanup)"`
	MaxLogFiles  int              `help:"Maximum number of log files to keep (0 = unlimited)" default:"1000"`
	DBPath       string           `help:"Path to SQLite database" type:"path" default:"~/.rocha/state.db" env:"ROCHA_DB_PATH"`

	Run         RunCmd         `cmd:"" help:"Start the rocha TUI (default)" default:"1"`
	Setup       SetupCmd       `cmd:"setup" help:"Configure tmux status bar integration automatically"`
	Status      StatusCmd      `cmd:"status" help:"Show session state counts for tmux status bar" hidden:""`
	Attach      AttachCmd      `cmd:"attach" help:"Attach to tmux session (creates if needed)"`
	StartClaude StartClaudeCmd `cmd:"start-claude" help:"Start Claude Code with hooks configured" hidden:""`
	PlaySound   PlaySoundCmd   `cmd:"play-sound" help:"Play notification sound (cross-platform)" hidden:""`
	Notify      NotifyCmd      `cmd:"notify" help:"Handle notification event from Claude hooks" hidden:""`
	Sessions    SessionsCmd    `cmd:"sessions" help:"Manage sessions (list, view, add, del)"`

	// Internal field for settings (not a flag)
	settings *config.Settings `kong:"-"`
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
		// Apply DBPath setting
		if c.DBPath == "~/.rocha/state.db" {
			if _, hasEnv := os.LookupEnv("ROCHA_DB_PATH"); !hasEnv {
				if c.settings.DBPath != "" {
					c.DBPath = c.settings.DBPath
				}
			}
		}

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

	return nil
}

// RunCmd starts the TUI application
type RunCmd struct {
	Dev             bool   `help:"Enable development mode (shows version info in dialogs)"`
	Editor          string `help:"Editor to open sessions in (overrides $ROCHA_EDITOR, $VISUAL, $EDITOR)" default:"code"`
	ErrorClearDelay int    `help:"Seconds before error messages auto-clear" default:"10"`
	StatusColors    string `help:"Comma-separated ANSI color codes for statuses (e.g., '141,33,214,226,46')" default:"141,33,214,226,46"`
	StatusIcons     string `help:"Comma-separated status icons (optional, colors are used for display)" default:""`
	Statuses        string `help:"Comma-separated status names (e.g., 'spec,plan,implement,review,done')" default:"spec,plan,implement,review,done"`
	WorktreePath    string `help:"Base directory for git worktrees" type:"path" default:"~/.rocha/worktrees"`
}

// Run executes the TUI
func (r *RunCmd) Run(tmuxClient tmux.Client, cli *CLI) error {
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

		// Apply WorktreePath setting
		if r.WorktreePath == "~/.rocha/worktrees" {
			if cli.settings.WorktreePath != "" {
				r.WorktreePath = cli.settings.WorktreePath
			}
		}
	}

	logging.Logger.Info("Starting rocha TUI")

	// Generate new execution ID for this TUI run
	executionID := state.NewExecutionID()
	// Set environment variable for child processes (including tmux sessions)
	// Hooks will also receive it explicitly via --execution-id flag
	os.Setenv("ROCHA_EXECUTION_ID", executionID)
	logging.Logger.Info("Generated new execution ID", "execution_id", executionID)

	// Open database
	dbPath := expandPath(cli.DBPath)
	store, err := storage.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Load state for initial session info
	st, err := store.Load(context.Background())
	if err != nil {
		log.Printf("Warning: failed to load session state: %v", err)
		logging.Logger.Warn("Failed to load session state", "error", err)
		st = &storage.SessionState{Sessions: make(map[string]storage.SessionInfo)}
	}
	logging.Logger.Debug("State loaded", "existing_sessions", len(st.Sessions))

	// Sync with running tmux sessions
	runningSessions, err := tmuxClient.List()
	if err != nil {
		logging.Logger.Warn("Failed to list tmux sessions", "error", err)
	} else {
		runningNames := make([]string, len(runningSessions))
		for i, sess := range runningSessions {
			runningNames[i] = sess.Name
		}
		logging.Logger.Info("Syncing with running tmux sessions", "count", len(runningNames))

		// Update execution ID for running sessions directly in database
		for _, sessionName := range runningNames {
			if sessionInfo, exists := st.Sessions[sessionName]; exists {
				if err := store.UpdateSession(context.Background(), sessionName, sessionInfo.State, executionID); err != nil {
					logging.Logger.Error("Failed to update session", "error", err, "session", sessionName)
				} else {
					logging.Logger.Debug("Updated session execution ID", "session", sessionName)
				}
			}
		}

		// Detect sessions where Claude has exited
		for _, sessionName := range runningNames {
			if !isClaudeRunningInSession(sessionName) {
				logging.Logger.Info("Claude has exited from session", "name", sessionName)
				if err := store.UpdateSession(context.Background(), sessionName, state.StateExited, executionID); err != nil {
					logging.Logger.Error("Failed to update exited state", "error", err, "session", sessionName)
				}
			}
		}
	}

	// Set terminal to raw mode for proper input handling
	logging.Logger.Debug("Initializing Bubble Tea program")
	errorClearDelay := time.Duration(r.ErrorClearDelay) * time.Second
	statusConfig := ui.NewStatusConfig(r.Statuses, r.StatusIcons, r.StatusColors)
	p := tea.NewProgram(
		ui.NewModel(tmuxClient, store, r.WorktreePath, r.Editor, errorClearDelay, statusConfig, r.Dev),
		tea.WithAltScreen(), // Use alternate screen buffer
	)

	logging.Logger.Info("Starting TUI program")
	if _, err := p.Run(); err != nil {
		logging.Logger.Error("TUI program error", "error", err)
		return fmt.Errorf("error running program: %w", err)
	}

	logging.Logger.Info("TUI program exited normally")
	return nil
}

// isClaudeRunningInSession checks if Claude Code is running in the given tmux session
func isClaudeRunningInSession(sessionName string) bool {
	cmd := exec.Command("pgrep", "-f", fmt.Sprintf("claude.*notify %s", sessionName))
	err := cmd.Run()
	return err == nil // Exit code 0 means process found
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[1:])
		}
	}
	return path
}
