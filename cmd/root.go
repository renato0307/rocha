package cmd

import (
	"fmt"
	"log"
	"os"
	"rocha/logging"
	"rocha/state"
	"rocha/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// CLI represents the command-line interface structure
type CLI struct {
	Debug        bool   `help:"Enable debug logging to file" short:"d"`
	DebugFile    string `help:"Custom path for debug log file (disables automatic cleanup)"`
	MaxLogFiles  int    `help:"Maximum number of log files to keep (0 = unlimited)" default:"1000"`

	Run         RunCmd         `cmd:"" help:"Start the rocha TUI (default)" default:"1"`
	Setup       SetupCmd       `cmd:"setup" help:"Configure tmux status bar integration automatically"`
	Status      StatusCmd      `cmd:"status" help:"Show session state counts for tmux status bar"`
	StartClaude StartClaudeCmd `cmd:"start-claude" help:"Start Claude Code with hooks configured" hidden:""`
	PlaySound   PlaySoundCmd   `cmd:"play-sound" help:"Play notification sound (cross-platform)" hidden:""`
	Notify      NotifyCmd      `cmd:"notify" help:"Handle notification event from Claude hooks" hidden:""`
}

// AfterApply initializes logging after CLI parsing
func (c *CLI) AfterApply() error {
	// Initialize logging first
	if err := logging.Initialize(c.Debug, c.DebugFile, c.MaxLogFiles); err != nil {
		return err
	}

	// Set environment variables AFTER initialization so child processes inherit debug settings
	// but we can detect if we're the main process or a child in Initialize
	if c.Debug || c.DebugFile != "" {
		os.Setenv("ROCHA_DEBUG", "1")
	}
	if c.DebugFile != "" {
		os.Setenv("ROCHA_DEBUG_FILE", c.DebugFile)
	}
	if c.MaxLogFiles != 1000 {
		os.Setenv("ROCHA_MAX_LOG_FILES", fmt.Sprintf("%d", c.MaxLogFiles))
	}

	return nil
}

// RunCmd starts the TUI application
type RunCmd struct{}

// Run executes the TUI
func (r *RunCmd) Run() error {
	logging.Logger.Info("Starting rocha TUI")

	// Generate new execution ID for this TUI run
	executionID := state.NewExecutionID()
	os.Setenv("ROCHA_EXECUTION_ID", executionID)
	logging.Logger.Info("Generated new execution ID", "execution_id", executionID)

	// Update state file with new execution ID
	st, err := state.Load()
	if err != nil {
		log.Printf("Warning: failed to load state: %v", err)
		logging.Logger.Warn("Failed to load state", "error", err)
		st = &state.SessionState{Sessions: make(map[string]state.SessionInfo)}
	}
	logging.Logger.Debug("State loaded", "existing_sessions", len(st.Sessions))

	st.ExecutionID = executionID
	if err := st.Save(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
		logging.Logger.Error("Failed to save state", "error", err)
	} else {
		logging.Logger.Debug("State saved with new execution ID")
	}

	// Set terminal to raw mode for proper input handling
	logging.Logger.Debug("Initializing Bubble Tea program")
	p := tea.NewProgram(
		ui.NewModel(),
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
