package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestState creates a temporary directory and overrides statePathFunc for testing
func setupTestState(t *testing.T) (string, func()) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")

	// Override statePathFunc for testing
	origStatePathFunc := statePathFunc
	statePathFunc = func() (string, error) {
		return statePath, nil
	}

	cleanup := func() {
		statePathFunc = origStatePathFunc
	}

	return statePath, cleanup
}

func TestNewExecutionID(t *testing.T) {
	id1 := NewExecutionID()
	id2 := NewExecutionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "Each execution ID should be unique")
}

func TestLoadEmptyState(t *testing.T) {
	_, cleanup := setupTestState(t)
	defer cleanup()

	st, err := Load()

	require.NoError(t, err)
	assert.NotNil(t, st)
	assert.NotNil(t, st.Sessions)
	assert.Empty(t, st.Sessions)
}

func TestSaveAndLoad(t *testing.T) {
	_, cleanup := setupTestState(t)
	defer cleanup()

	executionID := NewExecutionID()
	st := &SessionState{
		ExecutionID: executionID,
		Sessions: map[string]SessionInfo{
			"session1": {
				Name:        "session1",
				State:       StateWorking,
				ExecutionID: executionID,
				LastUpdated: time.Now(),
			},
		},
	}

	err := st.Save()
	require.NoError(t, err)

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, executionID, loaded.ExecutionID)
	assert.Len(t, loaded.Sessions, 1)
	assert.Equal(t, StateWorking, loaded.Sessions["session1"].State)
}

func TestUpdateSession(t *testing.T) {
	tests := []struct {
		name          string
		sessionName   string
		state         string
		executionID   string
		expectedState string
	}{
		{"working state", "session1", StateWorking, "exec-1", StateWorking},
		{"waiting state", "session2", StateWaiting, "exec-1", StateWaiting},
		{"update existing", "session1", StateWaiting, "exec-1", StateWaiting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTestState(t)
			defer cleanup()

			st := &SessionState{
				ExecutionID: "exec-1",
				Sessions:    make(map[string]SessionInfo),
			}

			err := st.UpdateSession(tt.sessionName, tt.state, tt.executionID)
			require.NoError(t, err)

			// Reload to verify persistence
			loaded, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedState, loaded.Sessions[tt.sessionName].State)
			assert.Equal(t, tt.executionID, loaded.Sessions[tt.sessionName].ExecutionID)
		})
	}
}

func TestRemoveSession(t *testing.T) {
	_, cleanup := setupTestState(t)
	defer cleanup()

	executionID := NewExecutionID()
	st := &SessionState{
		ExecutionID: executionID,
		Sessions: map[string]SessionInfo{
			"session1": {Name: "session1", State: StateWorking, ExecutionID: executionID},
			"session2": {Name: "session2", State: StateWaiting, ExecutionID: executionID},
		},
	}

	err := st.Save()
	require.NoError(t, err)

	// Remove session1
	err = st.RemoveSession("session1")
	require.NoError(t, err)

	// Reload and verify
	loaded, err := Load()
	require.NoError(t, err)
	assert.Len(t, loaded.Sessions, 1)
	assert.Contains(t, loaded.Sessions, "session2")
	assert.NotContains(t, loaded.Sessions, "session1")
}

func TestGetCounts(t *testing.T) {
	tests := []struct {
		name            string
		sessions        map[string]SessionInfo
		executionID     string
		expectedWaiting int
		expectedWorking int
	}{
		{
			name:            "empty state",
			sessions:        map[string]SessionInfo{},
			executionID:     "exec-1",
			expectedWaiting: 0,
			expectedWorking: 0,
		},
		{
			name: "all waiting",
			sessions: map[string]SessionInfo{
				"s1": {State: StateWaiting, ExecutionID: "exec-1"},
				"s2": {State: StateWaiting, ExecutionID: "exec-1"},
			},
			executionID:     "exec-1",
			expectedWaiting: 2,
			expectedWorking: 0,
		},
		{
			name: "all working",
			sessions: map[string]SessionInfo{
				"s1": {State: StateWorking, ExecutionID: "exec-1"},
				"s2": {State: StateWorking, ExecutionID: "exec-1"},
			},
			executionID:     "exec-1",
			expectedWaiting: 0,
			expectedWorking: 2,
		},
		{
			name: "mixed states",
			sessions: map[string]SessionInfo{
				"s1": {State: StateWaiting, ExecutionID: "exec-1"},
				"s2": {State: StateWorking, ExecutionID: "exec-1"},
				"s3": {State: StateWaiting, ExecutionID: "exec-1"},
			},
			executionID:     "exec-1",
			expectedWaiting: 2,
			expectedWorking: 1,
		},
		{
			name: "filter by execution ID",
			sessions: map[string]SessionInfo{
				"s1": {State: StateWaiting, ExecutionID: "exec-1"},
				"s2": {State: StateWorking, ExecutionID: "exec-2"},
				"s3": {State: StateWaiting, ExecutionID: "exec-1"},
			},
			executionID:     "exec-1",
			expectedWaiting: 2,
			expectedWorking: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &SessionState{
				ExecutionID: tt.executionID,
				Sessions:    tt.sessions,
			}

			waiting, working := st.GetCounts(tt.executionID)
			assert.Equal(t, tt.expectedWaiting, waiting)
			assert.Equal(t, tt.expectedWorking, working)
		})
	}
}

func TestConcurrentUpdates(t *testing.T) {
	_, cleanup := setupTestState(t)
	defer cleanup()

	executionID := NewExecutionID()
	numGoroutines := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Simulate concurrent hook updates
	for i := 0; i < numGoroutines; i++ {
		go func(sessionNum int) {
			defer wg.Done()

			st, err := Load()
			if err != nil {
				// Load can fail if a write is happening simultaneously
				// This is acceptable given our design (no read locks)
				t.Logf("Load failed for session %d: %v", sessionNum, err)
				return
			}

			sessionName := fmt.Sprintf("session%d", sessionNum)
			sessionState := StateWorking
			if sessionNum%2 == 0 {
				sessionState = StateWaiting
			}

			err = st.UpdateSession(sessionName, sessionState, executionID)
			// It's OK if some updates fail or get overwritten - we're mainly testing
			// that file locking prevents corruption
			if err != nil {
				t.Logf("Update failed for session %d: %v", sessionNum, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify we can load the state without corruption
	st, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, st.Sessions)
	// Due to load-modify-save races, we may not have all sessions
	// The important thing is that the file is not corrupted
	assert.GreaterOrEqual(t, len(st.Sessions), 1, "At least one session should be saved")
}

func TestRemoveSessionNonexistent(t *testing.T) {
	_, cleanup := setupTestState(t)
	defer cleanup()

	st := &SessionState{
		ExecutionID: NewExecutionID(),
		Sessions:    make(map[string]SessionInfo),
	}

	// Should not error when removing nonexistent session
	err := st.RemoveSession("nonexistent")
	assert.NoError(t, err)
}

func TestLoadCorruptedFile(t *testing.T) {
	statePath, cleanup := setupTestState(t)
	defer cleanup()

	// Write invalid JSON
	err := os.WriteFile(statePath, []byte("{invalid json"), 0644)
	require.NoError(t, err)

	// Should return empty state on corrupted file
	st, err := Load()
	require.Error(t, err)
	assert.NotNil(t, st)
	assert.NotNil(t, st.Sessions)
	assert.Empty(t, st.Sessions)
}
