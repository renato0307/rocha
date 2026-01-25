package config

import (
	"context"
	"os"
	"path/filepath"

	"rocha/logging"
	"rocha/paths"
	"rocha/ports"
)

// DefaultClaudeDir returns the default Claude directory
// Checks CLAUDE_CONFIG_DIR environment variable first, then falls back to ~/.claude
func DefaultClaudeDir() string {
	// Check environment variable first
	if envDir := os.Getenv("CLAUDE_CONFIG_DIR"); envDir != "" {
		return paths.ExpandPath(envDir)
	}

	// Fall back to ~/.claude
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.Logger.Warn("Failed to get home directory for ClaudeDir", "error", err)
		return "~/.claude"
	}
	return filepath.Join(homeDir, ".claude")
}

// DetectClaudeDirForRepo finds ClaudeDir from existing sessions
// of the same repository (for consistency)
// Returns empty string if no existing sessions found or if they don't have ClaudeDir set
func DetectClaudeDirForRepo(sessionReader ports.SessionReader, repoInfo string) (string, error) {
	if repoInfo == "" {
		return "", nil
	}

	logging.Logger.Debug("Detecting ClaudeDir for repo", "repo_info", repoInfo)

	// List all sessions (including archived)
	sessions, err := sessionReader.List(context.Background(), true)
	if err != nil {
		logging.Logger.Warn("Failed to list sessions for ClaudeDir detection", "error", err)
		return "", err
	}

	// Find first session with matching repoInfo and non-empty ClaudeDir
	for _, session := range sessions {
		if session.RepoInfo == repoInfo && session.ClaudeDir != "" {
			logging.Logger.Info("Found existing ClaudeDir for repo",
				"repo_info", repoInfo,
				"claude_dir", session.ClaudeDir,
				"from_session", session.Name)
			return session.ClaudeDir, nil
		}
	}

	logging.Logger.Debug("No existing ClaudeDir found for repo", "repo_info", repoInfo)
	return "", nil
}

// ResolveClaudeDir determines ClaudeDir with precedence:
// 1. User override (if provided and non-empty)
// 2. Existing sessions' ClaudeDir (if repo has sessions with ClaudeDir set)
// 3. Default ~/.claude
//
// Always returns an expanded absolute path
func ResolveClaudeDir(sessionReader ports.SessionReader, repoInfo, userOverride string) string {
	logging.Logger.Debug("Resolving ClaudeDir",
		"repo_info", repoInfo,
		"user_override", userOverride)

	// 1. User override takes precedence
	if userOverride != "" {
		path := paths.ExpandPath(userOverride)
		logging.Logger.Info("Using user-provided ClaudeDir", "path", path)
		return path
	}

	// 2. Try to detect from existing sessions
	if sessionReader != nil && repoInfo != "" {
		detected, err := DetectClaudeDirForRepo(sessionReader, repoInfo)
		if err == nil && detected != "" {
			path := paths.ExpandPath(detected)
			logging.Logger.Info("Using detected ClaudeDir from existing sessions", "path", path)
			return path
		}
	}

	// 3. Fall back to default
	path := DefaultClaudeDir()
	logging.Logger.Info("Using default ClaudeDir", "path", path)
	return path
}
