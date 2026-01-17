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
		fmt.Printf("W:? A:?")
		return nil
	}

	// Count only sessions from current execution
	waiting, working := st.GetCounts(st.ExecutionID)

	// If no sessions reported yet for this execution, show unknown
	if waiting == 0 && working == 0 {
		fmt.Printf("W:? A:?")
	} else {
		fmt.Printf("W:%d A:%d", waiting, working)
	}

	return nil
}
