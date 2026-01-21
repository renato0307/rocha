package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"rocha/git"
	"rocha/logging"
	"rocha/operations"
	"rocha/paths"
	"rocha/storage"

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

	// Delete session using operations package
	logging.Logger.Info("Deleting session", "session", s.Name)
	err = operations.DeleteSession(ctx, s.Name, store, operations.DeleteSessionOptions{
		KillTmux:       killTmux,
		RemoveWorktree: removeWorktree,
	})
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
	Names []string `arg:"" help:"Names of sessions to move" required:"true"`
	To    string   `help:"Destination ROCHA_HOME path" required:"true"`
}

// Run executes the move command
func (s *SessionsMoveCmd) Run(cli *CLI) error {
	logging.Logger.Info("Executing sessions move command", "sessions", s.Names, "from", s.From, "to", s.To, "force", s.Force)

	ctx := context.Background()

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

	// Display warning and ask for confirmation
	if !s.Force {
		logging.Logger.Debug("Prompting user for confirmation", "sessions", s.Names)
		fmt.Println("⚠ WARNING: This operation will:")
		fmt.Println("  • Kill tmux sessions for the selected sessions")
		fmt.Println("  • Move worktrees to the new ROCHA_HOME location")
		fmt.Printf("  • Move %d session(s) from %s to %s\n", len(s.Names), sourceHome, destHome)
		fmt.Println("\nSessions to move:")
		for _, name := range s.Names {
			fmt.Printf("  • %s\n", name)
		}
		fmt.Print("\nContinue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			logging.Logger.Info("User cancelled session move", "sessions", s.Names)
			fmt.Println("Cancelled")
			return nil
		}
		logging.Logger.Info("User confirmed session move", "sessions", s.Names)
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

	// PHASE 1: COPY
	logging.Logger.Info("Starting Phase 1: Copy", "sessions", s.Names)
	fmt.Println("\n📦 Phase 1: Copying sessions to destination...")
	copiedSessions := []string{}
	for _, name := range s.Names {
		logging.Logger.Debug("Copying session", "session", name)
		fmt.Printf("Copying session '%s'...\n", name)
		err := operations.MoveSession(ctx, name, sourceStore, destStore, sourceHome, destHome)
		if err != nil {
			logging.Logger.Error("Failed to copy session", "session", name, "error", err)
			return fmt.Errorf("failed to copy session %s: %w", name, err)
		}
		copiedSessions = append(copiedSessions, name)
		fmt.Printf("✓ Copied '%s'\n", name)
	}
	logging.Logger.Info("Phase 1 complete", "copiedCount", len(copiedSessions))

	// PHASE 2: VERIFY
	logging.Logger.Info("Starting Phase 2: Verify", "sessions", copiedSessions)
	fmt.Println("\n✅ Phase 2: Verifying sessions at destination...")
	for _, name := range copiedSessions {
		logging.Logger.Debug("Verifying session", "session", name)
		fmt.Printf("Verifying session '%s'...\n", name)
		err := operations.VerifySession(ctx, name, destStore)
		if err != nil {
			logging.Logger.Error("Verification failed", "session", name, "error", err)
			return fmt.Errorf("verification failed: %w", err)
		}
		fmt.Printf("✓ Verified '%s'\n", name)
	}
	logging.Logger.Info("Phase 2 complete - all sessions verified")

	// PHASE 3: DELETE
	logging.Logger.Info("Starting Phase 3: Delete from source", "sessions", copiedSessions)
	fmt.Println("\n🗑️  Phase 3: Deleting sessions from source...")
	successCount := 0
	for _, name := range copiedSessions {
		logging.Logger.Debug("Deleting session from source", "session", name)
		fmt.Printf("Deleting session '%s' from source...\n", name)
		// Note: tmux was already killed in MoveSession, only need to clean up worktree
		err := operations.DeleteSession(ctx, name, sourceStore, operations.DeleteSessionOptions{
			KillTmux:       false, // Already killed in Phase 1
			RemoveWorktree: true,  // Clean up source worktree
		})
		if err != nil {
			logging.Logger.Warn("Failed to delete session from source", "session", name, "error", err)
			fmt.Printf("⚠ Warning: Failed to delete session %s from source: %v\n", name, err)
			continue
		}
		successCount++
		fmt.Printf("✓ Deleted '%s' from source\n", name)
	}
	logging.Logger.Info("Phase 3 complete", "successCount", successCount, "totalCount", len(copiedSessions))

	// Report results
	fmt.Printf("\n✓ Successfully moved %d session(s)\n", successCount)
	if successCount < len(copiedSessions) {
		failedCount := len(copiedSessions) - successCount
		logging.Logger.Warn("Some sessions may need manual cleanup", "failedCount", failedCount)
		fmt.Printf("⚠ %d session(s) may need manual cleanup from source\n", failedCount)
	}

	logging.Logger.Info("Sessions move command completed successfully", "movedCount", successCount)
	return nil
}
