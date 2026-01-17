package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

const (
	StateWaiting = "waiting"
	StateWorking = "working"
)

// SessionState represents the persistent state of all Claude sessions
type SessionState struct {
	ExecutionID string                 `json:"execution_id"` // UUID for current rocha run
	Sessions    map[string]SessionInfo `json:"sessions"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// SessionInfo represents the state of a single session
type SessionInfo struct {
	Name        string    `json:"name"`
	State       string    `json:"state"`        // "waiting" or "working"
	ExecutionID string    `json:"execution_id"` // Which rocha run owns this state
	LastUpdated time.Time `json:"last_updated"`
}

// NewExecutionID generates a new UUID for the current rocha run
func NewExecutionID() string {
	return uuid.New().String()
}

// statePathFunc is a function variable that returns the path to the state file
// Can be overridden in tests
var statePathFunc = getDefaultStatePath

// getDefaultStatePath returns the default state file path
func getDefaultStatePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "rocha")
	return filepath.Join(configDir, "state.json"), nil
}

// GetStatePath returns the path to the state file
func GetStatePath() (string, error) {
	return statePathFunc()
}

// Load reads the state from disk. Returns empty state if file doesn't exist.
func Load() (*SessionState, error) {
	path, err := GetStatePath()
	if err != nil {
		return &SessionState{Sessions: make(map[string]SessionInfo)}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty state if file doesn't exist
			return &SessionState{Sessions: make(map[string]SessionInfo)}, nil
		}
		return &SessionState{Sessions: make(map[string]SessionInfo)}, fmt.Errorf("failed to read state file: %w", err)
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return &SessionState{Sessions: make(map[string]SessionInfo)}, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Ensure Sessions map is initialized
	if state.Sessions == nil {
		state.Sessions = make(map[string]SessionInfo)
	}

	return &state, nil
}

// Save writes the state to disk with file locking
func (s *SessionState) Save() error {
	path, err := GetStatePath()
	if err != nil {
		return err
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Open file with O_RDWR|O_CREATE
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open state file: %w", err)
	}
	defer file.Close()

	// Acquire exclusive lock
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unix.Flock(int(file.Fd()), unix.LOCK_UN)

	// Update timestamp
	s.UpdatedAt = time.Now()

	// Marshal to JSON
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Truncate file and write atomically
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// UpdateSession adds or updates a session with the given state and execution ID
func (s *SessionState) UpdateSession(name, state, executionID string) error {
	if s.Sessions == nil {
		s.Sessions = make(map[string]SessionInfo)
	}

	s.Sessions[name] = SessionInfo{
		Name:        name,
		State:       state,
		ExecutionID: executionID,
		LastUpdated: time.Now(),
	}

	return s.Save()
}

// RemoveSession deletes a session from the state
func (s *SessionState) RemoveSession(name string) error {
	if s.Sessions == nil {
		return nil
	}

	delete(s.Sessions, name)
	return s.Save()
}

// GetCounts returns the number of waiting and working sessions for the given execution ID
func (s *SessionState) GetCounts(executionID string) (waiting int, working int) {
	if s.Sessions == nil {
		return 0, 0
	}

	for _, session := range s.Sessions {
		if session.ExecutionID != executionID {
			continue
		}

		switch session.State {
		case StateWaiting:
			waiting++
		case StateWorking:
			working++
		}
	}

	return waiting, working
}
