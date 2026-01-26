package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// HookParser parses Claude session JSONL files to extract hook events
type HookParser struct {
	sessionReader ports.SessionReader
}

// NewHookParser creates a new HookParser
func NewHookParser(sessionReader ports.SessionReader) *HookParser {
	return &HookParser{
		sessionReader: sessionReader,
	}
}

// hookProgressEntry represents a JSONL entry with hook progress data
type hookProgressEntry struct {
	CWD       string              `json:"cwd"`
	Data      *hookProgressData   `json:"data"`
	Timestamp string              `json:"timestamp"`
	Type      string              `json:"type"`
}

type hookProgressData struct {
	Command   string `json:"command"`
	HookEvent string `json:"hookEvent"`
	HookName  string `json:"hookName"`
	Type      string `json:"type"`
}

// GetHookEvents returns hook events matching the filter
func (p *HookParser) GetHookEvents(filter ports.HookFilter) ([]domain.HookEvent, error) {
	// Build session name → worktree path map for correlation
	sessionMap, err := p.buildSessionMap()
	if err != nil {
		logging.Logger.Warn("Failed to build session map", "error", err)
		return nil, err
	}

	// Get all Claude directories to scan
	claudeDirs := p.getClaudeDirs()

	var allEvents []domain.HookEvent

	// Scan each Claude directory
	for _, claudeDir := range claudeDirs {
		events, err := p.scanClaudeDir(claudeDir, sessionMap, filter)
		if err != nil {
			logging.Logger.Debug("Failed to scan Claude directory", "dir", claudeDir, "error", err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	// Sort by timestamp (newest first)
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.After(allEvents[j].Timestamp)
	})

	// Apply limit
	if filter.Limit > 0 && len(allEvents) > filter.Limit {
		allEvents = allEvents[:filter.Limit]
	}

	logging.Logger.Debug("Parsed hook events", "total", len(allEvents))
	return allEvents, nil
}

// buildSessionMap creates a map from worktree path to session name
func (p *HookParser) buildSessionMap() (map[string]string, error) {
	ctx := context.Background()
	sessions, err := p.sessionReader.List(ctx, true) // include archived sessions
	if err != nil {
		return nil, err
	}

	sessionMap := make(map[string]string)
	for _, session := range sessions {
		if session.WorktreePath != "" {
			sessionMap[session.WorktreePath] = session.Name
		}
	}

	logging.Logger.Debug("Built session map", "entries", len(sessionMap))
	return sessionMap, nil
}

// getClaudeDirs returns unique Claude directories from sessions, plus defaults
func (p *HookParser) getClaudeDirs() []string {
	dirs := make(map[string]bool)

	// Add default Claude directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultDir := filepath.Join(homeDir, ".claude", "projects")
		dirs[defaultDir] = true
	}

	// Add CLAUDE_CONFIG_DIR if set
	if envDir := os.Getenv("CLAUDE_CONFIG_DIR"); envDir != "" {
		projectsDir := filepath.Join(config.ExpandPath(envDir), "projects")
		dirs[projectsDir] = true
	}

	// Add session-specific Claude directories
	ctx := context.Background()
	sessions, err := p.sessionReader.List(ctx, true) // include archived sessions
	if err == nil {
		for _, session := range sessions {
			if session.ClaudeDir != "" {
				projectsDir := filepath.Join(config.ExpandPath(session.ClaudeDir), "projects")
				dirs[projectsDir] = true
			}
		}
	}

	// Convert to slice
	var result []string
	for dir := range dirs {
		result = append(result, dir)
	}

	logging.Logger.Debug("Identified Claude directories", "count", len(result))
	return result
}

// scanClaudeDir scans a Claude projects directory for hook events
func (p *HookParser) scanClaudeDir(claudeDir string, sessionMap map[string]string, filter ports.HookFilter) ([]domain.HookEvent, error) {
	var events []domain.HookEvent

	// Check if directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		logging.Logger.Debug("Claude projects directory does not exist", "path", claudeDir)
		return events, nil
	}

	// Find all project directories
	projectDirs, err := filepath.Glob(filepath.Join(claudeDir, "*"))
	if err != nil {
		logging.Logger.Warn("Failed to glob project directories", "error", err)
		return events, nil
	}

	// Process each project directory
	for _, projectDir := range projectDirs {
		info, err := os.Stat(projectDir)
		if err != nil || !info.IsDir() {
			continue
		}

		// Find all JSONL files in this project
		jsonlFiles, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
		if err != nil {
			continue
		}

		// Process each JSONL file
		for _, jsonlFile := range jsonlFiles {
			fileEvents, err := p.parseJSONLFile(jsonlFile, sessionMap, filter)
			if err != nil {
				logging.Logger.Debug("Failed to parse JSONL file", "file", jsonlFile, "error", err)
				continue
			}
			events = append(events, fileEvents...)
		}
	}

	return events, nil
}

// parseJSONLFile parses a single JSONL file and extracts hook events
func (p *HookParser) parseJSONLFile(filePath string, sessionMap map[string]string, filter ports.HookFilter) ([]domain.HookEvent, error) {
	var events []domain.HookEvent

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry hookProgressEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		// Skip non-progress entries
		if entry.Type != "progress" {
			continue
		}

		// Skip entries without hook progress data
		if entry.Data == nil || entry.Data.Type != "hook_progress" {
			continue
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			continue
		}

		// Apply time range filter
		if !filter.From.IsZero() && timestamp.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && timestamp.After(filter.To) {
			continue
		}

		// Correlate cwd to session name
		sessionName := sessionMap[entry.CWD]
		if sessionName == "" {
			sessionName = "unknown"
		}

		// Apply session name filter
		if filter.SessionName != "" && sessionName != filter.SessionName {
			continue
		}

		// Apply event type filter
		if filter.EventType != "" && entry.Data.HookEvent != filter.EventType {
			continue
		}

		events = append(events, domain.HookEvent{
			Command:     entry.Data.Command,
			HookEvent:   entry.Data.HookEvent,
			HookName:    entry.Data.HookName,
			SessionName: sessionName,
			Timestamp:   timestamp,
		})
	}

	if err := scanner.Err(); err != nil {
		return events, err
	}

	return events, nil
}
