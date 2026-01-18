package cmd

import (
	"context"
	"fmt"
	"rocha/state"
	"rocha/storage"
)

// StatusCmd displays session state counts for tmux status bar
type StatusCmd struct{}

// Run executes the status command
func (s *StatusCmd) Run(cli *CLI) error {
	// Open database
	dbPath := expandPath(cli.DBPath)
	store, err := storage.NewStore(dbPath)
	if err != nil {
		// Database doesn't exist or can't be opened
		fmt.Printf("%s:? %s:? %s:?", state.SymbolWaitingUser, state.SymbolIdle, state.SymbolWorking)
		return nil
	}
	defer store.Close()

	st, err := store.Load(context.Background())
	if err != nil {
		// No state
		fmt.Printf("%s:? %s:? %s:?", state.SymbolWaitingUser, state.SymbolIdle, state.SymbolWorking)
		return nil
	}

	// Count ALL sessions regardless of execution ID
	// This ensures the status bar shows the global state across all rocha instances
	waiting, idle, working := 0, 0, 0
	for _, sess := range st.Sessions {
		switch sess.State {
		case state.StateWaitingUser:
			waiting++
		case state.StateIdle:
			idle++
		case state.StateWorking:
			working++
		}
	}

	// If no sessions at all, show zeros (not unknown)
	fmt.Printf("%s:%d %s:%d %s:%d", state.SymbolWaitingUser, waiting, state.SymbolIdle, idle, state.SymbolWorking, working)

	return nil
}
