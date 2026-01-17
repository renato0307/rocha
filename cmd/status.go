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
	if err != nil || st.ExecutionID == "" {
		// No state file or no current execution
		fmt.Printf("%s:? %s:? %s:?", state.SymbolWaitingUser, state.SymbolIdle, state.SymbolWorking)
		return nil
	}

	// Count only sessions from current execution
	waiting, idle, working := st.GetCounts(st.ExecutionID)

	// If no sessions reported yet for this execution, show unknown
	if waiting == 0 && idle == 0 && working == 0 {
		fmt.Printf("%s:? %s:? %s:?", state.SymbolWaitingUser, state.SymbolIdle, state.SymbolWorking)
	} else {
		fmt.Printf("%s:%d %s:%d %s:%d", state.SymbolWaitingUser, waiting, state.SymbolIdle, idle, state.SymbolWorking, working)
	}

	return nil
}
