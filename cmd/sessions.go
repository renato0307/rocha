package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"rocha/git"
	"rocha/storage"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
)

// SessionsCmd manages sessions
type SessionsCmd struct {
	Archive SessionsArchiveCmd `cmd:"archive" help:"Archive or unarchive a session"`
	List    SessionsListCmd    `cmd:"list" help:"List all sessions" default:"1"`
	View    SessionsViewCmd    `cmd:"view" help:"View a specific session"`
	Add     SessionsAddCmd     `cmd:"add" help:"Add a new session"`
	Del     SessionsDelCmd     `cmd:"del" help:"Delete a session"`
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
	store, err := storage.NewStore(expandPath(cli.DBPath))
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
	store, err := storage.NewStore(expandPath(cli.DBPath))
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
	store, err := storage.NewStore(expandPath(cli.DBPath))
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
	Name         string `arg:"" help:"Name of the session to add"`
	DisplayName  string `help:"Display name for the session" default:""`
	State        string `help:"Initial state" enum:"idle,working,waiting,exited" default:"idle"`
	RepoPath     string `help:"Repository path" default:""`
	RepoInfo     string `help:"Repository info" default:""`
	BranchName   string `help:"Branch name" default:""`
	WorktreePath string `help:"Worktree path" default:""`
}

// Run executes the add command
func (s *SessionsAddCmd) Run(cli *CLI) error {
	store, err := storage.NewStore(expandPath(cli.DBPath))
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	displayName := s.DisplayName
	if displayName == "" {
		displayName = s.Name
	}

	sessInfo := storage.SessionInfo{
		Name:         s.Name,
		DisplayName:  displayName,
		State:        s.State,
		ExecutionID:  uuid.New().String(),
		LastUpdated:  time.Now().UTC(),
		RepoPath:     s.RepoPath,
		RepoInfo:     s.RepoInfo,
		BranchName:   s.BranchName,
		WorktreePath: s.WorktreePath,
	}

	if err := store.AddSession(context.Background(), sessInfo); err != nil {
		return fmt.Errorf("failed to add session: %w", err)
	}

	fmt.Printf("✓ Session '%s' added successfully\n", s.Name)
	return nil
}

// SessionsDelCmd deletes a session
type SessionsDelCmd struct {
	Name  string `arg:"" help:"Name of the session to delete"`
	Force bool   `help:"Force deletion without confirmation" short:"f"`
}

// Run executes the del command
func (s *SessionsDelCmd) Run(cli *CLI) error {
	store, err := storage.NewStore(expandPath(cli.DBPath))
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Check if session exists
	_, err = store.GetSession(context.Background(), s.Name)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Ask for confirmation unless --force is used
	if !s.Force {
		fmt.Printf("Are you sure you want to delete session '%s'? (y/N): ", s.Name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := store.DeleteSession(context.Background(), s.Name); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	fmt.Printf("✓ Session '%s' deleted successfully\n", s.Name)
	return nil
}
