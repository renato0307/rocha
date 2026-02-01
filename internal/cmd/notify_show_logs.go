package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/renato0307/rocha/internal/logging"
)

// NotifyShowLogsCmd displays hook execution logs
type NotifyShowLogsCmd struct {
	Session string `arg:"" optional:"" help:"Filter by session name"`
	Since   string `help:"Show logs since duration (e.g., '1h', '30m')" default:"1h"`
	Format  string `help:"Output format (table or json)" default:"table" enum:"table,json"`
}

// hookLogEntry represents a single log entry from a hook log file
type hookLogEntry struct {
	Event     string    `json:"event"`
	Level     string    `json:"level"`
	Message   string    `json:"msg"`
	Session   string    `json:"session"`
	Source    string    `json:"source"` // filename
	Timestamp time.Time `json:"time"`
}

// Run executes the show-logs command
func (s *NotifyShowLogsCmd) Run(cli *CLI) error {
	logDir, err := logging.GetLogDir()
	if err != nil {
		return fmt.Errorf("failed to get log directory: %w", err)
	}

	// Parse --since duration
	sinceDuration, err := time.ParseDuration(s.Since)
	if err != nil {
		return fmt.Errorf("invalid --since duration: %w", err)
	}
	sinceTime := time.Now().Add(-sinceDuration)

	// Find all hook log files
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No hook logs found.")
			return nil
		}
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	// Collect and filter log entries
	var allEntries []hookLogEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "hook-") || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		// Parse filename: hook-<session>-<event>-<timestamp>.log
		parsed := parseHookLogFilename(entry.Name())
		if parsed == nil {
			continue
		}

		// Filter by session if specified
		if s.Session != "" && parsed.session != s.Session {
			continue
		}

		// Filter by time - check file timestamp first for efficiency
		if parsed.timestamp.Before(sinceTime) {
			continue
		}

		// Read and parse log entries from file
		logPath := filepath.Join(logDir, entry.Name())
		fileEntries, err := parseHookLogFile(logPath, parsed.session, parsed.event)
		if err != nil {
			logging.Logger.Warn("Failed to parse hook log file", "file", entry.Name(), "error", err)
			continue
		}

		// Filter entries by time
		for _, logEntry := range fileEntries {
			if !logEntry.Timestamp.Before(sinceTime) {
				logEntry.Source = entry.Name()
				allEntries = append(allEntries, logEntry)
			}
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.After(allEntries[j].Timestamp)
	})

	// Output based on format
	switch s.Format {
	case "json":
		return s.outputJSON(allEntries)
	default:
		return s.outputTable(allEntries)
	}
}

func (s *NotifyShowLogsCmd) outputTable(entries []hookLogEntry) error {
	if len(entries) == 0 {
		fmt.Println("No hook logs found for the specified criteria.")
		return nil
	}

	fmt.Printf("%-20s %-15s %-20s %-7s %s\n", "TIMESTAMP", "SESSION", "EVENT", "LEVEL", "MESSAGE")
	fmt.Println(strings.Repeat("-", 100))

	for _, e := range entries {
		timestamp := e.Timestamp.Format("2006-01-02 15:04:05")
		session := truncate(e.Session, 15)
		event := truncate(e.Event, 20)
		level := truncate(e.Level, 7)
		message := truncate(e.Message, 35)

		fmt.Printf("%-20s %-15s %-20s %-7s %s\n", timestamp, session, event, level, message)
	}

	fmt.Printf("\nTotal: %d log entries\n", len(entries))
	return nil
}

func (s *NotifyShowLogsCmd) outputJSON(entries []hookLogEntry) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

// parsedHookLogFilename holds parsed components from a hook log filename
type parsedHookLogFilename struct {
	event     string
	session   string
	timestamp time.Time
}

// parseHookLogFilename parses a hook log filename
// Format: hook-<session>-<event>-<timestamp>.log
// Example: hook-my-session-stop-20260201-143022.log
func parseHookLogFilename(filename string) *parsedHookLogFilename {
	// Remove prefix and suffix
	name := strings.TrimPrefix(filename, "hook-")
	name = strings.TrimSuffix(name, ".log")

	// Find timestamp at the end (format: 20060102-150405)
	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		return nil
	}

	// Last two parts are the timestamp (YYYYMMDD-HHMMSS)
	timestampStr := parts[len(parts)-2] + "-" + parts[len(parts)-1]
	timestamp, err := time.ParseInLocation("20060102-150405", timestampStr, time.Local)
	if err != nil {
		return nil
	}

	// Event is the part before the timestamp
	event := parts[len(parts)-3]

	// Session is everything before the event
	session := strings.Join(parts[:len(parts)-3], "-")

	return &parsedHookLogFilename{
		event:     event,
		session:   session,
		timestamp: timestamp,
	}
}

// parseHookLogFile reads a hook log file and returns parsed entries
func parseHookLogFile(path, session, event string) ([]hookLogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []hookLogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue // Skip malformed lines
		}

		entry := hookLogEntry{
			Event:   event,
			Session: session,
		}

		// Extract time
		if timeStr, ok := raw["time"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
				entry.Timestamp = t
			} else if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				entry.Timestamp = t
			}
		}

		// Extract level
		if level, ok := raw["level"].(string); ok {
			entry.Level = level
		}

		// Extract message
		if msg, ok := raw["msg"].(string); ok {
			entry.Message = msg
		}

		// Override session if present in log entry
		if s, ok := raw["session"].(string); ok {
			entry.Session = s
		}

		// Override event if present in log entry
		if e, ok := raw["event"].(string); ok {
			entry.Event = e
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// truncate truncates a string to maxLen, adding "..." if needed
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
