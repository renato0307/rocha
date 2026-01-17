package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Logger is the public logger instance accessible from all packages
var Logger *slog.Logger

// Initialize sets up the logger based on the debug flag and configuration
func Initialize(debug bool, debugFile string, maxLogFiles int) error {
	// Check environment variables for inherited debug settings
	if os.Getenv("ROCHA_DEBUG") == "1" {
		debug = true
	}
	if envDebugFile := os.Getenv("ROCHA_DEBUG_FILE"); envDebugFile != "" && debugFile == "" {
		debugFile = envDebugFile
	}
	if envMaxLogFiles := os.Getenv("ROCHA_MAX_LOG_FILES"); envMaxLogFiles != "" && maxLogFiles == 1000 {
		// Only override if not explicitly set
		if parsed, err := strconv.Atoi(envMaxLogFiles); err == nil {
			maxLogFiles = parsed
		}
	}

	if !debug && debugFile == "" {
		// Discard all logs when debug is false and no custom file
		Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
		return nil
	}

	var logFilePath string

	if debugFile != "" {
		// Use custom debug file path (no rotation)
		logFilePath = debugFile

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
	} else {
		// Use OS-specific log directory with rotation
		logDir, err := getLogDir()
		if err != nil {
			return fmt.Errorf("failed to get log directory: %w", err)
		}

		// Create log directory if it doesn't exist
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Rotate logs if needed (only when not using custom file)
		if maxLogFiles > 0 {
			if err := rotateLogs(logDir, maxLogFiles); err != nil {
				// Log rotation failure shouldn't prevent logging
				fmt.Fprintf(os.Stderr, "Warning: log rotation failed: %v\n", err)
			}
		}

		// Create log file with UUID name
		logFileName := fmt.Sprintf("%s.log", uuid.New().String())
		logFilePath = filepath.Join(logDir, logFileName)
	}

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Create JSON handler with options
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewJSONHandler(logFile, opts)
	Logger = slog.New(handler)

	// Log the log file location and print to stdout
	// Only do this if debug was explicitly enabled (not inherited from env)
	// This prevents spam from hooks and status bar updates
	wasExplicit := os.Getenv("ROCHA_DEBUG") == ""
	if wasExplicit {
		Logger.Info("Debug logging initialized", "log_file", logFilePath)
		fmt.Printf("Debug mode enabled. Logs: %s\n", logFilePath)
	}

	return nil
}

// rotateLogs removes old log files if there are more than maxLogFiles
func rotateLogs(logDir string, maxLogFiles int) error {
	// Read all log files in directory
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	// Filter for .log files and get their info
	type logFileInfo struct {
		path    string
		modTime time.Time
	}
	var logFiles []logFileInfo

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".log" {
			continue
		}

		fullPath := filepath.Join(logDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		logFiles = append(logFiles, logFileInfo{
			path:    fullPath,
			modTime: info.ModTime(),
		})
	}

	// If we're under the limit, nothing to do
	if len(logFiles) < maxLogFiles {
		return nil
	}

	// Sort by modification time (oldest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.Before(logFiles[j].modTime)
	})

	// Delete oldest files to get under the limit
	numToDelete := len(logFiles) - maxLogFiles + 1 // +1 to make room for the new log
	for i := 0; i < numToDelete && i < len(logFiles); i++ {
		if err := os.Remove(logFiles[i].path); err != nil {
			// Continue even if deletion fails
			fmt.Fprintf(os.Stderr, "Warning: failed to delete old log file %s: %v\n", logFiles[i].path, err)
		}
	}

	return nil
}

// getLogDir returns the OS-specific log directory
func getLogDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Logs/rocha
		return filepath.Join(homeDir, "Library", "Logs", "rocha"), nil
	case "linux":
		// Linux: ~/.local/state/rocha or XDG_STATE_HOME
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			stateHome = filepath.Join(homeDir, ".local", "state")
		}
		return filepath.Join(stateHome, "rocha"), nil
	case "windows":
		// Windows: %LOCALAPPDATA%\rocha\logs
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		return filepath.Join(localAppData, "rocha", "logs"), nil
	default:
		// Fallback to home directory
		return filepath.Join(homeDir, ".rocha", "logs"), nil
	}
}
