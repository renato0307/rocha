package state

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"rocha/logging"
)

const (
	StateWaitingUser = "waiting" // Red - waiting for user input/prompt
	StateWorking     = "working" // Green - actively working
	StateIdle        = "idle"    // Yellow - finished/idle
	StateExited      = "exited"  // Gray - tmux session exists but Claude has exited

	// Status symbols (Unicode)
	SymbolWorking     = "●" // Green - actively working
	SymbolIdle        = "○" // Yellow - finished/idle
	SymbolWaitingUser = "◐" // Red - waiting for user input/prompt
	SymbolExited      = "■" // Gray - Claude has exited
)

// SessionState represents the persistent state of all Claude sessions
type SessionState struct {
	ExecutionID         string                 `json:"execution_id"` // UUID for current rocha run
	Sessions            map[string]SessionInfo `json:"sessions"`
	UpdatedAt           time.Time              `json:"updated_at"`
	SortOrder           string                 `json:"sort_order"`                  // Session list sort order: "name", "updated", "created", "state"
	OrderedSessionNames []string               `json:"ordered_session_names,omitempty"` // Manual order of sessions
}

// SessionInfo represents the state of a single session
type SessionInfo struct {
	Name         string        `json:"name"`                    // Tmux session name (no spaces)
	ShellSession *SessionInfo  `json:"shell_session,omitempty"` // Nested shell session info (optional)
	DisplayName  string        `json:"display_name"`            // Display name (with spaces)
	State        string        `json:"state"`                   // "waiting" or "working"
	ExecutionID  string        `json:"execution_id"`            // Which rocha run owns this state
	LastUpdated  time.Time     `json:"last_updated"`
	RepoPath     string        `json:"repo_path"`     // Original repository path (if in git repo)
	RepoInfo     string        `json:"repo_info"`     // GitHub owner/repo (e.g., "owner/repo")
	BranchName   string        `json:"branch_name"`   // Git branch name (if worktree created)
	WorktreePath string        `json:"worktree_path"` // Path to worktree if created
}

// StateEvent represents an event to be applied to state
type StateEvent struct {
	Type      string    `json:"type"`       // "update_session" or "sync_running"
	Timestamp time.Time `json:"timestamp"`

	// For update_session events
	SessionName string `json:"session_name,omitempty"`
	State       string `json:"state,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`

	// For sync_running events
	RunningSessionNames []string `json:"running_session_names,omitempty"`
	NewExecutionID      string   `json:"new_execution_id,omitempty"`
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

// getQueuePath returns the path to the event queue file
func getQueuePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "rocha")
	return filepath.Join(configDir, "state-queue.jsonl"), nil
}

// appendEvent appends an event to the queue file (thread-safe across processes)
func appendEvent(event StateEvent) error {
	queuePath, err := getQueuePath()
	if err != nil {
		return err
	}

	logging.Logger.Debug("Attempting to append event to queue",
		"type", event.Type,
		"session", event.SessionName,
		"state", event.State,
		"queue_path", queuePath,
		"pid", os.Getpid())

	if err := os.MkdirAll(filepath.Dir(queuePath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Open file for append with create
	file, err := os.OpenFile(queuePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		logging.Logger.Error("Failed to open queue file", "error", err, "path", queuePath)
		return fmt.Errorf("failed to open queue file: %w", err)
	}
	defer file.Close()

	logging.Logger.Debug("Acquiring lock on queue file")

	// Lock for append (OS-specific)
	if err := lockFile(file); err != nil {
		logging.Logger.Error("Failed to acquire lock", "error", err)
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() {
		logging.Logger.Debug("Releasing lock on queue file")
		unlockFile(file)
	}()

	// Marshal and append
	event.Timestamp = time.Now()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	logging.Logger.Debug("Writing event to queue", "data_size", len(data))

	if _, err := file.Write(append(data, '\n')); err != nil {
		logging.Logger.Error("Failed to write event", "error", err)
		return fmt.Errorf("failed to write event: %w", err)
	}

	logging.Logger.Info("Queue event written successfully",
		"type", event.Type,
		"session", event.SessionName,
		"new_state", event.State,
		"execution_id", event.ExecutionID)

	return nil
}

// processQueueEvents reads and applies all queued events, then clears the queue
func processQueueEvents(state *SessionState) error {
	queuePath, err := getQueuePath()
	if err != nil {
		return err
	}

	// Open queue file
	file, err := os.OpenFile(queuePath, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No queue file, nothing to process
		}
		return fmt.Errorf("failed to open queue file: %w", err)
	}
	defer file.Close()

	// Lock queue file (OS-specific)
	if err := lockFile(file); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlockFile(file)

	// Read all events
	scanner := bufio.NewScanner(file)
	var events []StateEvent

	for scanner.Scan() {
		var event StateEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			logging.Logger.Warn("Failed to unmarshal queue event, skipping", "error", err)
			continue
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read queue: %w", err)
	}

	// Apply events in order
	for _, event := range events {
		if err := applyEvent(state, event); err != nil {
			logging.Logger.Warn("Failed to apply event", "type", event.Type, "error", err)
		}
	}

	// Save state after applying events (before clearing queue for safety)
	if len(events) > 0 {
		if err := state.Save(); err != nil {
			logging.Logger.Warn("Failed to save state after applying events", "error", err)
		}
	}

	// Clear queue after processing and saving
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	return nil
}

// applyEvent applies a single event to the state
func applyEvent(state *SessionState, event StateEvent) error {
	if state.Sessions == nil {
		state.Sessions = make(map[string]SessionInfo)
	}

	switch event.Type {
	case "update_session":
		existing, exists := state.Sessions[event.SessionName]
		if exists {
			existing.State = event.State
			existing.ExecutionID = event.ExecutionID
			existing.LastUpdated = event.Timestamp
			state.Sessions[event.SessionName] = existing
		} else {
			// Don't create new sessions from update events - sessions should only be created
			// via UpdateSessionWithGit() or git detection during startup. This prevents
			// race conditions where hook events arrive before git metadata is saved.
			logging.Logger.Warn("Ignoring update event for non-existent session",
				"session", event.SessionName,
				"state", event.State)
		}

	case "sync_running":
		runningMap := make(map[string]bool)
		for _, name := range event.RunningSessionNames {
			runningMap[name] = true
		}

		for name, session := range state.Sessions {
			if runningMap[name] {
				// Only update ExecutionID, preserve current state
				// (Don't reset state - if Claude is working, it should stay working)
				session.ExecutionID = event.NewExecutionID
				session.LastUpdated = event.Timestamp
				state.Sessions[name] = session
			}
		}

	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}

	return nil
}

// QueueUpdateSession queues a session state update
func QueueUpdateSession(name, state, executionID string) error {
	return appendEvent(StateEvent{
		Type:        "update_session",
		SessionName: name,
		State:       state,
		ExecutionID: executionID,
	})
}

// QueueSyncRunning queues a sync operation
func QueueSyncRunning(runningSessionNames []string, newExecutionID string) error {
	return appendEvent(StateEvent{
		Type:                "sync_running",
		RunningSessionNames: runningSessionNames,
		NewExecutionID:      newExecutionID,
	})
}

// Load reads the state from disk and applies any queued events
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

	if state.Sessions == nil {
		state.Sessions = make(map[string]SessionInfo)
	}

	// Initialize default sort order if not set
	if state.SortOrder == "" {
		state.SortOrder = "name"
	}

	// Process queued events before returning (this will save state if events were processed)
	if err := processQueueEvents(&state); err != nil {
		logging.Logger.Warn("Failed to process queue events", "error", err)
	}

	// Clean up orphaned shell sessions (sessions ending with "-shell")
	// that exist as top-level entries (shouldn't exist with new nested structure)
	hadOrphans := false
	for name := range state.Sessions {
		if len(name) > 6 && name[len(name)-6:] == "-shell" {
			logging.Logger.Warn("Removing orphaned shell session from state", "name", name)
			delete(state.Sessions, name)
			hadOrphans = true
		}
	}

	// Save state if we cleaned up orphaned sessions
	if hadOrphans {
		if err := state.Save(); err != nil {
			logging.Logger.Warn("Failed to save state after cleanup", "error", err)
		}
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
	if err := lockFile(file); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlockFile(file)

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

	// Get existing session to preserve ShellSession field
	existing, exists := s.Sessions[name]

	sessionInfo := SessionInfo{
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

	// Preserve ShellSession if it exists
	if exists {
		sessionInfo.ShellSession = existing.ShellSession
	}

	s.Sessions[name] = sessionInfo

	return s.Save()
}

// RenameSession renames a session by updating both tmux name (map key) and display name
// The old session is removed and a new entry is created with the new name as key
func (s *SessionState) RenameSession(oldName, newName, newDisplayName string) error {
	if s.Sessions == nil {
		return fmt.Errorf("no sessions in state")
	}

	// Get existing session info
	oldInfo, exists := s.Sessions[oldName]
	if !exists {
		return fmt.Errorf("session %s not found in state", oldName)
	}

	// Check if new name already exists
	if _, exists := s.Sessions[newName]; exists && newName != oldName {
		return fmt.Errorf("session %s already exists", newName)
	}

	// Create new entry with updated names
	newInfo := SessionInfo{
		Name:         newName,
		DisplayName:  newDisplayName,
		State:        oldInfo.State,
		ExecutionID:  oldInfo.ExecutionID,
		LastUpdated:  time.Now(),
		RepoPath:     oldInfo.RepoPath,
		RepoInfo:     oldInfo.RepoInfo,
		BranchName:   oldInfo.BranchName,
		WorktreePath: oldInfo.WorktreePath,
		ShellSession: oldInfo.ShellSession, // Preserve shell session
	}

	// Remove old entry and add new entry
	delete(s.Sessions, oldName)
	s.Sessions[newName] = newInfo

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

// GetCounts returns the number of waiting, idle, working, and exited sessions for the given execution ID
func (s *SessionState) GetCounts(executionID string) (waiting int, idle int, working int, exited int) {
	if s.Sessions == nil {
		return 0, 0, 0, 0
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
		case StateExited:
			exited++
		}
	}

	return waiting, idle, working, exited
}

// GetAllCounts returns the number of waiting, idle, working, and exited sessions across all execution IDs
// This is useful for the status bar to show global state regardless of which rocha instance is active
func (s *SessionState) GetAllCounts() (waiting int, idle int, working int, exited int) {
	if s.Sessions == nil {
		return 0, 0, 0, 0
	}

	for _, session := range s.Sessions {
		switch session.State {
		case StateWaitingUser:
			waiting++
		case StateIdle:
			idle++
		case StateWorking:
			working++
		case StateExited:
			exited++
		}
	}

	return waiting, idle, working, exited
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

// AddSessionToTop adds a session to the top of the manual order list
// If the session is already in the list, it moves it to the top
func (s *SessionState) AddSessionToTop(sessionName string) error {
	// Remove from current position if it exists
	for i, name := range s.OrderedSessionNames {
		if name == sessionName {
			s.OrderedSessionNames = append(s.OrderedSessionNames[:i], s.OrderedSessionNames[i+1:]...)
			break
		}
	}

	// Add to top
	s.OrderedSessionNames = append([]string{sessionName}, s.OrderedSessionNames...)

	return s.Save()
}

// MoveSessionUp moves a session up one position in the manual order
// Returns error if session is already at the top or not found
func (s *SessionState) MoveSessionUp(sessionName string) error {
	// Find session in order list
	idx := -1
	for i, name := range s.OrderedSessionNames {
		if name == sessionName {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("session %s not found in order list", sessionName)
	}

	if idx == 0 {
		return nil // Already at top, no-op
	}

	// Swap with previous
	s.OrderedSessionNames[idx], s.OrderedSessionNames[idx-1] = s.OrderedSessionNames[idx-1], s.OrderedSessionNames[idx]

	return s.Save()
}

// MoveSessionDown moves a session down one position in the manual order
// Returns error if session is already at the bottom or not found
func (s *SessionState) MoveSessionDown(sessionName string) error {
	// Find session in order list
	idx := -1
	for i, name := range s.OrderedSessionNames {
		if name == sessionName {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("session %s not found in order list", sessionName)
	}

	if idx == len(s.OrderedSessionNames)-1 {
		return nil // Already at bottom, no-op
	}

	// Swap with next
	s.OrderedSessionNames[idx], s.OrderedSessionNames[idx+1] = s.OrderedSessionNames[idx+1], s.OrderedSessionNames[idx]

	return s.Save()
}
