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
	StateWaitingUser = "waiting" // Red - waiting for user input/prompt
	StateWorking     = "working" // Green - actively working
	StateIdle        = "idle"    // Yellow - finished/idle

	// Status symbols (Unicode)
	SymbolWorking     = "●" // Green - actively working
	SymbolIdle        = "○" // Yellow - finished/idle
	SymbolWaitingUser = "◐" // Red - waiting for user input/prompt
)

// SessionState represents the persistent state of all Claude sessions
type SessionState struct {
	ExecutionID string                 `json:"execution_id"` // UUID for current rocha run
	Sessions    map[string]SessionInfo `json:"sessions"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// SessionInfo represents the state of a single session
type SessionInfo struct {
	Name         string    `json:"name"`          // Tmux session name (no spaces)
	DisplayName  string    `json:"display_name"`  // Display name (with spaces)
	State        string    `json:"state"`         // "waiting" or "working"
	ExecutionID  string    `json:"execution_id"`  // Which rocha run owns this state
	LastUpdated  time.Time `json:"last_updated"`
	RepoPath     string    `json:"repo_path"`     // Original repository path (if in git repo)
	RepoInfo     string    `json:"repo_info"`     // GitHub owner/repo (e.g., "owner/repo")
	BranchName   string    `json:"branch_name"`   // Git branch name (if worktree created)
	WorktreePath string    `json:"worktree_path"` // Path to worktree if created
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
// Preserves existing git metadata fields if the session already exists
func (s *SessionState) UpdateSession(name, state, executionID string) error {
	if s.Sessions == nil {
		s.Sessions = make(map[string]SessionInfo)
	}

	// Get existing session if it exists
	existing, exists := s.Sessions[name]

	// If session exists, preserve all existing fields and only update what we need
	if exists {
		existing.State = state
		existing.ExecutionID = executionID
		existing.LastUpdated = time.Now()
		s.Sessions[name] = existing
	} else {
		// New session - create with basic fields only
		s.Sessions[name] = SessionInfo{
			Name:        name,
			State:       state,
			ExecutionID: executionID,
			LastUpdated: time.Now(),
		}
	}

	return s.Save()
}

// UpdateSessionWithGit adds or updates a session with git metadata
func (s *SessionState) UpdateSessionWithGit(name, displayName, state, executionID, repoPath, repoInfo, branchName, worktreePath string) error {
	if s.Sessions == nil {
		s.Sessions = make(map[string]SessionInfo)
	}

	s.Sessions[name] = SessionInfo{
		Name:         name,
		DisplayName:  displayName,
		State:        state,
		ExecutionID:  executionID,
		LastUpdated:  time.Now(),
		RepoPath:     repoPath,
		RepoInfo:     repoInfo,
		BranchName:   branchName,
		WorktreePath: worktreePath,
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

// GetCounts returns the number of waiting, idle, and working sessions for the given execution ID
func (s *SessionState) GetCounts(executionID string) (waiting int, idle int, working int) {
	if s.Sessions == nil {
		return 0, 0, 0
	}

	for _, session := range s.Sessions {
		if session.ExecutionID != executionID {
			continue
		}

		switch session.State {
		case StateWaitingUser:
			waiting++
		case StateIdle:
			idle++
		case StateWorking:
			working++
		}
	}

	return waiting, idle, working
}

// SyncWithRunning updates execution IDs for sessions that are actually running in tmux
// Sessions not running in tmux are kept in state (so they can be restarted later)
// but their execution IDs are not updated
func (s *SessionState) SyncWithRunning(runningSessionNames []string, newExecutionID string) error {
	if s.Sessions == nil {
		s.Sessions = make(map[string]SessionInfo)
	}

	// Create a map of running session names for fast lookup
	runningMap := make(map[string]bool)
	for _, name := range runningSessionNames {
		runningMap[name] = true
	}

	// Update execution ID only for sessions that are currently running in tmux
	// Keep sessions that aren't running (they can be restarted later)
	for name, session := range s.Sessions {
		if runningMap[name] {
			// Session is running - update its execution ID to current
			session.ExecutionID = newExecutionID
			s.Sessions[name] = session
		}
		// If not running in tmux, keep the session in state unchanged
		// so it can be restarted later
	}

	return s.Save()
}
