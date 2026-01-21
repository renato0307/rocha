package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rocha/git"
	"rocha/logging"
	"rocha/operations"
	"rocha/paths"
	"rocha/storage"
	"rocha/tmux"

	"text/tabwriter"
	"time"

	"github.com/google/uuid"
)

// SessionsCmd manages sessions
type SessionsCmd struct {
	Add     SessionsAddCmd     `cmd:"add" help:"Add a new session"`
	Archive SessionsArchiveCmd `cmd:"archive" help:"Archive or unarchive a session"`
	Del     SessionsDelCmd     `cmd:"del" help:"Delete a session"`
	List    SessionsListCmd    `cmd:"list" help:"List all sessions" default:"1"`
	Move    SessionsMoveCmd    `cmd:"move" aliases:"mv" help:"Move sessions between ROCHA_HOME directories"`
	Set     SessionSetCmd      `cmd:"set" help:"Set session configuration"`
	View    SessionsViewCmd    `cmd:"view" help:"View a specific session"`
}

// SessionsArchiveCmd archives or unarchives a session
type SessionsArchiveCmd struct {
	Name                string `arg:"" help:"Name of the session to archive/unarchive"`
	Force               bool   `help:"Skip confirmation prompt" short:"f"`
	RemoveWorktree      bool   `help:"Remove associated git worktree" short:"w"`
	SkipWorktreePrompt  bool   `help:"Don't prompt about worktree removal" short:"s"`
}

// Run executes the archive command
func (s *SessionsArchiveCmd) Run(cli *CLI) error {
	store, err := storage.NewStore(paths.GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Check if session exists
	session, err := store.GetSession(context.Background(), s.Name)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Determine if archiving or unarchiving
	isArchiving := !session.IsArchived

	if isArchiving {
		// Archiving workflow
		if !s.Force {
			fmt.Printf("Are you sure you want to archive session '%s'? (y/N): ", s.Name)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		// Handle worktree removal
		removeWorktree := s.RemoveWorktree
		if session.WorktreePath != "" && !s.SkipWorktreePrompt && !s.RemoveWorktree {
			fmt.Printf("Remove associated worktree at '%s'? (y/N): ", session.WorktreePath)
			var response string
			fmt.Scanln(&response)
			removeWorktree = (response == "y" || response == "Y")
		}

		// Remove worktree if requested
		if removeWorktree && session.WorktreePath != "" {
			if err := git.RemoveWorktree(session.RepoPath, session.WorktreePath); err != nil {
				fmt.Printf("⚠ Warning: Failed to remove worktree: %v\n", err)
				fmt.Println("Continuing with archiving...")
			} else {
				fmt.Printf("✓ Removed worktree at '%s'\n", session.WorktreePath)
			}
		}

		// Archive the session
		if err := store.ToggleArchive(context.Background(), s.Name); err != nil {
			return fmt.Errorf("failed to archive session: %w", err)
		}

		fmt.Printf("✓ Session '%s' archived successfully\n", s.Name)
	} else {
		// Unarchiving workflow
		if err := store.ToggleArchive(context.Background(), s.Name); err != nil {
			return fmt.Errorf("failed to unarchive session: %w", err)
		}

		fmt.Printf("✓ Session '%s' unarchived successfully\n", s.Name)
	}

	return nil
}

// SessionsListCmd lists all sessions
type SessionsListCmd struct {
	Format       string `help:"Output format: table or json" enum:"table,json" default:"table"`
	ShowArchived bool   `help:"Show archived sessions" short:"a"`
}

// Run executes the list command
func (s *SessionsListCmd) Run(cli *CLI) error {
	store, err := storage.NewStore(paths.GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	sessions, err := store.ListSessions(context.Background(), s.ShowArchived)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if s.Format == "json" {
		data, err := json.MarshalIndent(sessions, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDISPLAY NAME\tSTATE\tBRANCH\tREPO\tARCHIVED\tLAST UPDATED")
	for _, sess := range sessions {
		archived := ""
		if sess.IsArchived {
			archived = "✓"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			sess.Name,
			sess.DisplayName,
			sess.State,
			sess.BranchName,
			sess.RepoInfo,
			archived,
			sess.LastUpdated.Format("2006-01-02 15:04:05"))
	}
	w.Flush()

	fmt.Printf("\nTotal: %d sessions\n", len(sessions))
	return nil
}

// SessionsViewCmd views a specific session
type SessionsViewCmd struct {
	Name   string `arg:"" help:"Name of the session to view"`
	Format string `help:"Output format: table or json" enum:"table,json" default:"table"`
}

// Run executes the view command
func (s *SessionsViewCmd) Run(cli *CLI) error {
	store, err := storage.NewStore(paths.GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	session, err := store.GetSession(context.Background(), s.Name)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if s.Format == "json" {
		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("Session: %s\n", session.Name)
	fmt.Printf("Display Name: %s\n", session.DisplayName)
	fmt.Printf("State: %s\n", session.State)
	fmt.Printf("Execution ID: %s\n", session.ExecutionID)
	fmt.Printf("Archived: %t\n", session.IsArchived)
	fmt.Printf("Flagged: %t\n", session.IsFlagged)
	fmt.Printf("Last Updated: %s\n", session.LastUpdated.Format("2006-01-02 15:04:05"))
	fmt.Printf("Repo Path: %s\n", session.RepoPath)
	fmt.Printf("Repo Info: %s\n", session.RepoInfo)
	fmt.Printf("Branch Name: %s\n", session.BranchName)
	fmt.Printf("Worktree Path: %s\n", session.WorktreePath)
	if session.ClaudeDir != "" {
		fmt.Printf("Claude Dir: %s\n", session.ClaudeDir)
	} else {
		fmt.Printf("Claude Dir: <default>\n")
	}

	if session.ShellSession != nil {
		fmt.Printf("\nShell Session:\n")
		fmt.Printf("  Name: %s\n", session.ShellSession.Name)
		fmt.Printf("  Display Name: %s\n", session.ShellSession.DisplayName)
		fmt.Printf("  State: %s\n", session.ShellSession.State)
		fmt.Printf("  Last Updated: %s\n", session.ShellSession.LastUpdated.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// SessionsAddCmd adds a new session
type SessionsAddCmd struct {
	AllowDangerouslySkipPermissions bool   `help:"Skip permission prompts in Claude (DANGEROUS)"`
	BranchName                      string `help:"Branch name" default:""`
	DisplayName                     string `help:"Display name for the session" default:""`
	Name                            string `arg:"" help:"Name of the session to add"`
	RepoInfo                        string `help:"Repository info" default:""`
	RepoPath                        string `help:"Repository path" default:""`
	State                           string `help:"Initial state" enum:"idle,working,waiting,exited" default:"idle"`
	WorktreePath                    string `help:"Worktree path" default:""`
}

// Run executes the add command
func (s *SessionsAddCmd) Run(cli *CLI) error {
	store, err := storage.NewStore(paths.GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	displayName := s.DisplayName
	if displayName == "" {
		displayName = s.Name
	}

	sessInfo := storage.SessionInfo{
		AllowDangerouslySkipPermissions: s.AllowDangerouslySkipPermissions,
		BranchName:                      s.BranchName,
		DisplayName:                     displayName,
		ExecutionID:                     uuid.New().String(),
		LastUpdated:                     time.Now().UTC(),
		Name:                            s.Name,
		RepoInfo:                        s.RepoInfo,
		RepoPath:                        s.RepoPath,
		State:                           s.State,
		WorktreePath:                    s.WorktreePath,
	}

	if err := store.AddSession(context.Background(), sessInfo); err != nil {
		return fmt.Errorf("failed to add session: %w", err)
	}

	fmt.Printf("✓ Session '%s' added successfully\n", s.Name)
	return nil
}

// SessionsDelCmd deletes a session
type SessionsDelCmd struct {
	Force              bool   `help:"Force deletion without confirmation" short:"f"`
	Name               string `arg:"" help:"Name of the session to delete"`
	SkipKillTmux       bool   `help:"Skip killing tmux session" short:"k"`
	SkipRemoveWorktree bool   `help:"Skip removing associated git worktree" short:"w"`
}

// Run executes the del command
func (s *SessionsDelCmd) Run(cli *CLI) error {
	// Calculate actual actions (inverted from skip flags)
	killTmux := !s.SkipKillTmux
	removeWorktree := !s.SkipRemoveWorktree

	logging.Logger.Info("Executing sessions del command", "session", s.Name, "killTmux", killTmux, "removeWorktree", removeWorktree, "force", s.Force)

	ctx := context.Background()
	store, err := storage.NewStore(paths.GetDBPath())
	if err != nil {
		logging.Logger.Error("Failed to open database", "error", err)
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Check if session exists
	logging.Logger.Debug("Checking if session exists", "session", s.Name)
	sessInfo, err := store.GetSession(ctx, s.Name)
	if err != nil {
		logging.Logger.Error("Session not found", "session", s.Name, "error", err)
		return fmt.Errorf("session not found: %w", err)
	}
	logging.Logger.Debug("Session found", "session", s.Name, "worktreePath", sessInfo.WorktreePath)

	// Display warning and ask for confirmation unless --force is used
	if !s.Force {
		logging.Logger.Debug("Prompting user for confirmation", "session", s.Name)
		fmt.Printf("⚠ WARNING: This will delete session '%s'\n", s.Name)
		if killTmux {
			fmt.Println("  • Kill tmux session")
		}
		if removeWorktree && sessInfo.WorktreePath != "" {
			fmt.Printf("  • Remove worktree at '%s'\n", sessInfo.WorktreePath)
		}
		fmt.Print("\nContinue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			logging.Logger.Info("User cancelled session deletion", "session", s.Name)
			fmt.Println("Cancelled")
			return nil
		}
		logging.Logger.Info("User confirmed session deletion", "session", s.Name)
	}

	// Create tmux client
	tmuxClient := tmux.NewClient()

	// Delete session using operations package
	logging.Logger.Info("Deleting session", "session", s.Name)
	err = operations.DeleteSession(ctx, s.Name, store, operations.DeleteSessionOptions{
		KillTmux:       killTmux,
		RemoveWorktree: removeWorktree,
	}, tmuxClient)
	if err != nil {
		logging.Logger.Error("Failed to delete session", "session", s.Name, "error", err)
		return fmt.Errorf("failed to delete session: %w", err)
	}

	logging.Logger.Info("Session deleted successfully via CLI", "session", s.Name)
	fmt.Printf("✓ Session '%s' deleted successfully\n", s.Name)
	return nil
}

// SessionsMoveCmd moves sessions between ROCHA_HOME directories
type SessionsMoveCmd struct {
	Force bool     `help:"Skip confirmation prompt" short:"f"`
	From  string   `help:"Source ROCHA_HOME path" required:"true"`
	Repos []string `help:"Repository identifiers (owner/repo format)" short:"r" required:"true"`
	To    string   `help:"Destination ROCHA_HOME path" required:"true"`
}

// Run executes the move command
func (s *SessionsMoveCmd) Run(cli *CLI) error {
	logging.Logger.Info("Executing sessions move command", "repos", s.Repos, "from", s.From, "to", s.To, "force", s.Force)

	ctx := context.Background()

	// Validate repo format
	for _, repo := range s.Repos {
		if !strings.Contains(repo, "/") {
			return fmt.Errorf("invalid repo format '%s': must be in owner/repo format", repo)
		}
	}

	// Expand paths
	sourceHome := paths.ExpandPath(s.From)
	destHome := paths.ExpandPath(s.To)
	logging.Logger.Debug("Paths expanded", "sourceHome", sourceHome, "destHome", destHome)

	// Validate source path exists
	logging.Logger.Debug("Validating source path", "path", sourceHome)
	if _, err := os.Stat(sourceHome); os.IsNotExist(err) {
		logging.Logger.Error("Source ROCHA_HOME does not exist", "path", sourceHome)
		return fmt.Errorf("source ROCHA_HOME does not exist: %s", sourceHome)
	}

	// Create destination path if it doesn't exist
	logging.Logger.Debug("Creating destination directory if needed", "path", destHome)
	if err := os.MkdirAll(destHome, 0755); err != nil {
		logging.Logger.Error("Failed to create destination ROCHA_HOME", "path", destHome, "error", err)
		return fmt.Errorf("failed to create destination ROCHA_HOME: %w", err)
	}

	// Open both databases
	sourceDBPath := filepath.Join(sourceHome, "state.db")
	destDBPath := filepath.Join(destHome, "state.db")
	logging.Logger.Debug("Opening databases", "source", sourceDBPath, "dest", destDBPath)

	sourceStore, err := storage.NewStore(sourceDBPath)
	if err != nil {
		logging.Logger.Error("Failed to open source database", "path", sourceDBPath, "error", err)
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceStore.Close()

	destStore, err := storage.NewStore(destDBPath)
	if err != nil {
		logging.Logger.Error("Failed to open destination database", "path", destDBPath, "error", err)
		return fmt.Errorf("failed to open destination database: %w", err)
	}
	defer destStore.Close()

	// Create tmux client
	tmuxClient := tmux.NewClient()

	// Get session counts per repo for confirmation message
	repoSessionCounts := make(map[string]int)
	for _, repo := range s.Repos {
		sessions, err := sourceStore.ListSessions(ctx, false)
		if err != nil {
			logging.Logger.Error("Failed to list sessions", "error", err)
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		count := 0
		for _, sess := range sessions {
			if sess.RepoInfo == repo {
				count++
			}
		}
		repoSessionCounts[repo] = count
	}

	// Display warning and ask for confirmation
	if !s.Force {
		logging.Logger.Debug("Prompting user for confirmation", "repos", s.Repos)
		fmt.Println("⚠ WARNING: This operation will:")
		fmt.Println("  • Kill tmux sessions for all sessions in the specified repositories")
		fmt.Println("  • Move .main directories and all worktrees to the new ROCHA_HOME location")
		fmt.Println("  • Repair git worktree references")
		fmt.Printf("  • Move sessions from %s to %s\n", sourceHome, destHome)
		fmt.Println("\nRepositories to move:")
		totalSessions := 0
		for _, repo := range s.Repos {
			count := repoSessionCounts[repo]
			totalSessions += count
			fmt.Printf("  • %s (%d session(s))\n", repo, count)
		}
		fmt.Printf("\nTotal: %d session(s) across %d repositor(y/ies)\n", totalSessions, len(s.Repos))
		fmt.Print("\nContinue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			logging.Logger.Info("User cancelled session move", "repos", s.Repos)
			fmt.Println("Cancelled")
			return nil
		}
		logging.Logger.Info("User confirmed session move", "repos", s.Repos)
	}

	// Move each repository
	allMovedSessions := []string{}
	for _, repo := range s.Repos {
		fmt.Printf("\n📦 Moving repository: %s\n", repo)
		logging.Logger.Info("Starting repository move", "repo", repo)

		movedSessions, err := operations.MoveRepository(ctx, repo, sourceStore, destStore, sourceHome, destHome, tmuxClient)
		if err != nil {
			logging.Logger.Error("Failed to move repository", "repo", repo, "error", err)
			return fmt.Errorf("failed to move repository %s: %w", repo, err)
		}

		allMovedSessions = append(allMovedSessions, movedSessions...)
		fmt.Printf("✓ Moved repository '%s' (%d session(s))\n", repo, len(movedSessions))
	}

	// Report results
	fmt.Printf("\n✓ Successfully moved %d session(s) across %d repositor(y/ies)\n", len(allMovedSessions), len(s.Repos))
	logging.Logger.Info("Sessions move command completed successfully", "movedCount", len(allMovedSessions), "repoCount", len(s.Repos))
	return nil
}

// SessionSetCmd sets configuration for a session
type SessionSetCmd struct {
	Name     string `arg:"" optional:"" help:"Name of the session (omit when using --all)"`
	All      bool   `help:"Apply to all sessions" short:"a"`
	KillTmux bool   `help:"Kill tmux sessions to apply changes immediately" short:"k"`
	Value    string `help:"Value to set (empty string to clear)" required:""`
	Variable string `help:"Variable to set" short:"v" enum:"claudedir" required:""`
}

// AfterApply validates that either Name or All is provided, but not both
func (s *SessionSetCmd) AfterApply() error {
	hasName := s.Name != ""
	hasAll := s.All

	// XOR validation: must have exactly one of Name or All
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
		"name", s.Name,
		"variable", s.Variable,
		"value", s.Value,
		"all", s.All,
		"killTmux", s.KillTmux)

	// Note: AfterApply() has already validated Name/All XOR
	// Note: Variable enum validation is handled by Kong
	logging.Logger.Debug("Arguments validated by Kong and AfterApply")

	// Open database
	dbPath := paths.GetDBPath()
	logging.Logger.Debug("Opening database", "path", dbPath)
	store, err := storage.NewStore(dbPath)
	if err != nil {
		logging.Logger.Error("Failed to open database", "path", dbPath, "error", err)
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()
	logging.Logger.Debug("Database opened successfully")

	// Create tmux client
	tmuxClient := tmux.NewClient()

	// Get sessions to update
	var sessionNames []string
	if s.All {
		logging.Logger.Info("Updating ClaudeDir for all sessions", "value", s.Value)
		sessions, err := store.ListSessions(ctx, false) // Skip archived
		if err != nil {
			logging.Logger.Error("Failed to list sessions", "error", err)
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		for _, sess := range sessions {
			sessionNames = append(sessionNames, sess.Name)
		}
		logging.Logger.Debug("Retrieved sessions to update", "count", len(sessionNames), "sessions", sessionNames)
		fmt.Printf("Updating ClaudeDir for %d sessions...\n", len(sessionNames))
	} else {
		sessionNames = []string{s.Name}
		logging.Logger.Debug("Updating single session", "session", s.Name)
	}

	// Update each session
	logging.Logger.Info("Starting session updates", "count", len(sessionNames))
	successCount := 0
	var failedSessions []string
	for _, name := range sessionNames {
		logging.Logger.Debug("Updating session", "session", name, "value", s.Value)
		if err := operations.SetSessionClaudeDir(ctx, name, s.Value, store); err != nil {
			logging.Logger.Warn("Failed to update session", "session", name, "error", err)
			fmt.Printf("⚠ Failed to update '%s': %v\n", name, err)
			failedSessions = append(failedSessions, name)
			continue
		}
		successCount++
		logging.Logger.Debug("Session updated successfully", "session", name)
		fmt.Printf("✓ Updated '%s'\n", name)
	}
	logging.Logger.Info("Session updates completed", "successCount", successCount, "failedCount", len(failedSessions))

	// Kill tmux sessions if requested
	if s.KillTmux {
		logging.Logger.Info("Killing tmux sessions", "count", successCount)
		for _, name := range sessionNames {
			// Skip failed sessions
			skip := false
			for _, failed := range failedSessions {
				if failed == name {
					skip = true
					break
				}
			}
			if skip {
				logging.Logger.Debug("Skipping tmux kill for failed session", "session", name)
				continue
			}

			// Kill main session
			logging.Logger.Debug("Killing main tmux session", "session", name)
			if err := tmuxClient.Kill(name); err != nil {
				logging.Logger.Warn("Failed to kill tmux session", "session", name, "error", err)
				fmt.Printf("⚠ Warning: Failed to kill tmux session '%s': %v\n", name, err)
			} else {
				logging.Logger.Debug("Main tmux session killed", "session", name)
				fmt.Printf("✓ Killed tmux session '%s'\n", name)
			}

			// Kill shell session if exists
			shellName := name + "-shell"
			logging.Logger.Debug("Attempting to kill shell session", "session", shellName)
			if err := tmuxClient.Kill(shellName); err != nil {
				// Shell session might not exist, just log debug
				logging.Logger.Debug("Shell session not found or already killed", "session", shellName)
			} else {
				logging.Logger.Debug("Shell tmux session killed", "session", shellName)
				fmt.Printf("✓ Killed tmux session '%s'\n", shellName)
			}
		}
		logging.Logger.Info("Tmux session kills completed")
	}

	// Show warning if tmux sessions are still running
	if !s.KillTmux {
		logging.Logger.Debug("Checking for running tmux sessions")
		// Check which sessions are running
		successfulSessions := []string{}
		for _, name := range sessionNames {
			// Check if this session was updated successfully
			failed := false
			for _, f := range failedSessions {
				if f == name {
					failed = true
					break
				}
			}
			if !failed {
				successfulSessions = append(successfulSessions, name)
			}
		}

		runningSessions, err := operations.GetRunningTmuxSessions(successfulSessions, tmuxClient)
		if err != nil {
			logging.Logger.Warn("Failed to check running tmux sessions", "error", err)
		} else if len(runningSessions) > 0 {
			logging.Logger.Info("Found running tmux sessions with old CLAUDE_CONFIG_DIR", "count", len(runningSessions), "sessions", runningSessions)
			fmt.Println()
			fmt.Printf("⚠ Warning: %d tmux session(s) are still running with old CLAUDE_CONFIG_DIR:\n", len(runningSessions))
			for _, name := range runningSessions {
				fmt.Printf("  • %s\n", name)
			}
			fmt.Println()
			fmt.Println("Restart them to apply changes:")
			if s.All {
				fmt.Printf("  rocha session set --all claudedir %q --kill-tmux\n", s.Value)
			} else {
				fmt.Printf("  rocha session set %s claudedir %q --kill-tmux\n", s.Name, s.Value)
			}
			fmt.Println()
			fmt.Println("Or manually:")
			for _, name := range runningSessions {
				fmt.Printf("  tmux kill-session -t %s\n", name)
			}
		} else {
			logging.Logger.Debug("No running tmux sessions found for updated sessions")
		}
	}

	// Print summary
	logging.Logger.Info("Session set command completed", "successCount", successCount, "totalCount", len(sessionNames))
	fmt.Println()
	if successCount == len(sessionNames) {
		if s.Value == "" {
			fmt.Printf("✓ Cleared ClaudeDir for %d session(s) (will use default)\n", successCount)
		} else {
			fmt.Printf("✓ Updated ClaudeDir for %d session(s)\n", successCount)
		}
	} else {
		fmt.Printf("✓ Updated %d of %d session(s)\n", successCount, len(sessionNames))
	}

	return nil
}
