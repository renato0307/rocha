package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
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

	Run         RunCmd         `cmd:"" help:"Start the rocha TUI (default)" default:"1"`
	Setup       SetupCmd       `cmd:"setup" help:"Configure tmux status bar integration automatically"`
	Status      StatusCmd      `cmd:"status" help:"Show session state counts for tmux status bar"`
	Attach      AttachCmd      `cmd:"attach" help:"Attach to tmux session (creates if needed)"`
	StartClaude StartClaudeCmd `cmd:"start-claude" help:"Start Claude Code with hooks configured" hidden:""`
	PlaySound   PlaySoundCmd   `cmd:"play-sound" help:"Play notification sound (cross-platform)" hidden:""`
	Notify      NotifyCmd      `cmd:"notify" help:"Handle notification event from Claude hooks" hidden:""`
}

// AfterApply initializes logging after CLI parsing
func (c *CLI) AfterApply() error {
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
	WorktreePath    string `help:"Base directory for git worktrees" type:"path" default:"~/.rocha/worktrees"`
	Editor          string `help:"Editor to open sessions in (overrides $ROCHA_EDITOR, $VISUAL, $EDITOR)" default:"code"`
	ErrorClearDelay int    `help:"Seconds before error messages auto-clear" default:"10"`
}

// Run executes the TUI
func (r *RunCmd) Run(tmuxClient tmux.Client) error {
	logging.Logger.Info("Starting rocha TUI")

	// Generate new execution ID for this TUI run
	executionID := state.NewExecutionID()
	// Set environment variable for child processes (including tmux sessions)
	// Hooks will also receive it explicitly via --execution-id flag
	os.Setenv("ROCHA_EXECUTION_ID", executionID)
	logging.Logger.Info("Generated new execution ID", "execution_id", executionID)

	// Load state
	st, err := state.Load()
	if err != nil {
		log.Printf("Warning: failed to load state: %v", err)
		logging.Logger.Warn("Failed to load state", "error", err)
		st = &state.SessionState{Sessions: make(map[string]state.SessionInfo)}
	}
	logging.Logger.Debug("State loaded", "existing_sessions", len(st.Sessions))

	// Sync with running tmux sessions
	// Update execution ID for sessions that are currently running
	runningSessions, err := tmuxClient.List()
	if err != nil {
		logging.Logger.Warn("Failed to list tmux sessions", "error", err)
	} else {
		runningNames := make([]string, len(runningSessions))
		for i, sess := range runningSessions {
			runningNames[i] = sess.Name
		}
		logging.Logger.Info("Syncing with running tmux sessions", "count", len(runningNames))
		if err := state.QueueSyncRunning(runningNames, executionID); err != nil {
			logging.Logger.Error("Failed to queue sync with running sessions", "error", err)
		} else {
			logging.Logger.Debug("Sync with running sessions queued")
		}

		// Detect sessions where Claude has exited
		for _, sessionName := range runningNames {
			if !isClaudeRunningInSession(sessionName) {
				logging.Logger.Info("Claude has exited from session", "name", sessionName)
				if err := state.QueueUpdateSession(sessionName, state.StateExited, executionID); err != nil {
					logging.Logger.Error("Failed to queue exited state", "error", err, "session", sessionName)
				}
			}
		}

		// Detect and update git metadata for running sessions with worktrees
		homeDir, err := os.UserHomeDir()
		if err == nil {
			worktreeBase := filepath.Join(homeDir, ".rocha", "worktrees")
			logging.Logger.Debug("Detecting git metadata for running sessions", "worktree_base", worktreeBase)

			for _, sessionName := range runningNames {
				session, exists := st.Sessions[sessionName]
				if !exists {
					continue
				}

				// Check if worktree exists for this session
				worktreePath := filepath.Join(worktreeBase, sessionName)
				if _, err := os.Stat(worktreePath); err == nil {
					// Worktree exists - detect git metadata
					logging.Logger.Debug("Detecting git metadata for session", "name", sessionName, "worktree_path", worktreePath)

					branchName := git.GetBranchName(worktreePath)
					isRepo, repoRoot := git.IsGitRepo(worktreePath)

					if isRepo && branchName != "" {
						repoInfo := git.GetRepoInfo(repoRoot)

						// Update session with detected metadata
						session.WorktreePath = worktreePath
						session.BranchName = branchName
						session.RepoPath = repoRoot
						session.RepoInfo = repoInfo

						st.Sessions[sessionName] = session
						logging.Logger.Info("Updated git metadata for session",
							"name", sessionName,
							"branch", branchName,
							"repo", repoInfo)
					}
				}
			}

			// Save the updated state with git metadata
			if err := st.Save(); err != nil {
				logging.Logger.Error("Failed to save state with git metadata", "error", err)
			}
		}
	}

	// Update state file with new execution ID
	st.ExecutionID = executionID
	if err := st.Save(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
		logging.Logger.Error("Failed to save state", "error", err)
	} else {
		logging.Logger.Debug("State saved with new execution ID")
	}

	// Set terminal to raw mode for proper input handling
	logging.Logger.Debug("Initializing Bubble Tea program")
	errorClearDelay := time.Duration(r.ErrorClearDelay) * time.Second
	p := tea.NewProgram(
		ui.NewModel(tmuxClient, r.WorktreePath, r.Editor, errorClearDelay),
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

// isClaudeRunningInSession checks if Claude Code is running in the given tmux session
func isClaudeRunningInSession(sessionName string) bool {
	cmd := exec.Command("pgrep", "-f", fmt.Sprintf("claude.*notify %s", sessionName))
	err := cmd.Run()
	return err == nil // Exit code 0 means process found
}
