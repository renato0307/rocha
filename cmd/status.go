package cmd

import (
	"fmt"
	"rocha/state"
)

// StatusCmd displays session state counts for tmux status bar
type StatusCmd struct{}

// Run executes the status command
func (s *StatusCmd) Run() error {
	st, err := state.Load()
	if err != nil {
		// No state file
		fmt.Printf("%s:? %s:? %s:?", state.SymbolWaitingUser, state.SymbolIdle, state.SymbolWorking)
		return nil
	}

	// Count ALL sessions regardless of execution ID
	// This ensures the status bar shows the global state across all rocha instances
	waiting, idle, working := st.GetAllCounts()

	// If no sessions at all, show zeros (not unknown)
	fmt.Printf("%s:%d %s:%d %s:%d", state.SymbolWaitingUser, waiting, state.SymbolIdle, idle, state.SymbolWorking, working)

	return nil
}
