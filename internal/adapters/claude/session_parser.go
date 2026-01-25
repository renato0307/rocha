package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// SessionParser parses Claude session JSONL files to extract token usage
type SessionParser struct {
	claudeProjectsDir string
}

// NewSessionParser creates a new SessionParser
func NewSessionParser() *SessionParser {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return &SessionParser{
		claudeProjectsDir: filepath.Join(homeDir, ".claude", "projects"),
	}
}

// NewSessionParserWithDir creates a new SessionParser with a custom directory (for testing)
func NewSessionParserWithDir(dir string) *SessionParser {
	return &SessionParser{
		claudeProjectsDir: dir,
	}
}

// GetTodayUsage returns all token usage entries for today
func (p *SessionParser) GetTodayUsage() ([]ports.TokenUsage, error) {
	var allUsage []ports.TokenUsage

	// Get today's date at midnight (local time)
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Check if projects directory exists
	if _, err := os.Stat(p.claudeProjectsDir); os.IsNotExist(err) {
		logging.Logger.Debug("Claude projects directory does not exist", "path", p.claudeProjectsDir)
		return allUsage, nil
	}

	// Find all project directories
	projectDirs, err := filepath.Glob(filepath.Join(p.claudeProjectsDir, "*"))
	if err != nil {
		logging.Logger.Warn("Failed to glob project directories", "error", err)
		return allUsage, nil
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
			usage, err := p.parseJSONLFile(jsonlFile, todayStart)
			if err != nil {
				logging.Logger.Debug("Failed to parse JSONL file", "file", jsonlFile, "error", err)
				continue
			}
			allUsage = append(allUsage, usage...)
		}
	}

	logging.Logger.Debug("Parsed token usage", "total_entries", len(allUsage))
	return allUsage, nil
}

// jsonlEntry represents a single entry in the JSONL file
type jsonlEntry struct {
	Message   *jsonlMessage `json:"message"`
	Timestamp string        `json:"timestamp"`
	Type      string        `json:"type"`
}

type jsonlMessage struct {
	Usage *jsonlUsage `json:"usage"`
}

type jsonlUsage struct {
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// parseJSONLFile parses a single JSONL file and extracts token usage for today
func (p *SessionParser) parseJSONLFile(filePath string, todayStart time.Time) ([]ports.TokenUsage, error) {
	var usage []ports.TokenUsage

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

		var entry jsonlEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		// Skip non-assistant messages
		if entry.Type != "assistant" {
			continue
		}

		// Skip entries without usage data
		if entry.Message == nil || entry.Message.Usage == nil {
			continue
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			continue
		}

		// Skip entries not from today
		if timestamp.Before(todayStart) {
			continue
		}

		usage = append(usage, ports.TokenUsage{
			CacheCreation: entry.Message.Usage.CacheCreationInputTokens,
			CacheRead:     entry.Message.Usage.CacheReadInputTokens,
			InputTokens:   entry.Message.Usage.InputTokens,
			OutputTokens:  entry.Message.Usage.OutputTokens,
			Timestamp:     timestamp,
		})
	}

	if err := scanner.Err(); err != nil {
		return usage, err
	}

	return usage, nil
}
