package cmd

import (
	"context"
	"fmt"

	"rocha/internal/domain"
)

// StatusCmd displays session state counts for tmux status bar
type StatusCmd struct{}

// Run executes the status command
func (s *StatusCmd) Run(cli *CLI) error {
	// Initialize container
	container, err := NewContainer(nil)
	if err != nil {
		// Database doesn't exist or can't be opened
		fmt.Printf("%s:? %s:? %s:?", domain.SymbolWaiting, domain.SymbolIdle, domain.SymbolWorking)
		return nil
	}
	defer container.Close()

	st, err := container.SessionService.LoadState(context.Background(), false)
	if err != nil {
		// No state
		fmt.Printf("%s:? %s:? %s:?", domain.SymbolWaiting, domain.SymbolIdle, domain.SymbolWorking)
		return nil
	}

	// Count ALL sessions regardless of execution ID
	// This ensures the status bar shows the global state across all rocha instances
	waiting, idle, working := 0, 0, 0
	for _, sess := range st.Sessions {
		switch sess.State {
		case domain.StateWaiting:
			waiting++
		case domain.StateIdle:
			idle++
		case domain.StateWorking:
			working++
		}
	}

	// If no sessions at all, show zeros (not unknown)
	fmt.Printf("%s:%d %s:%d %s:%d", domain.SymbolWaiting, waiting, domain.SymbolIdle, idle, domain.SymbolWorking, working)

	return nil
}
