package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"rocha/domain"
	"rocha/adapters/tmux"
)

// SessionsListCmd lists all sessions
type SessionsListCmd struct {
	Format       string `help:"Output format: table or json" enum:"table,json" default:"table"`
	ShowArchived bool   `help:"Show archived sessions" short:"a"`
}

// Run executes the list command
func (s *SessionsListCmd) Run(cli *CLI) error {
	tmuxClient := tmux.NewClient()
	container, err := NewContainer(tmuxClient)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer container.Close()

	sessions, err := container.SessionRepository.List(context.Background(), s.ShowArchived)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if s.Format == "json" {
		return s.printJSON(sessions)
	}
	return s.printTable(sessions)
}

func (s *SessionsListCmd) printJSON(sessions []domain.Session) error {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func (s *SessionsListCmd) printTable(sessions []domain.Session) error {
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
