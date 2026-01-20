package paths

import (
	"os"
	"path/filepath"
)

// GetRochaHome returns ROCHA_HOME or ~/.rocha default
func GetRochaHome() string {
	rochaHome := os.Getenv("ROCHA_HOME")
	if rochaHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ".rocha"
		}
		return filepath.Join(homeDir, ".rocha")
	}
	return ExpandPath(rochaHome)
}

// GetDBPath returns $ROCHA_HOME/state.db
func GetDBPath() string {
	return filepath.Join(GetRochaHome(), "state.db")
}

// GetWorktreePath returns $ROCHA_HOME/worktrees
func GetWorktreePath() string {
	return filepath.Join(GetRochaHome(), "worktrees")
}

// GetSettingsPath returns $ROCHA_HOME/settings.json
func GetSettingsPath() string {
	return filepath.Join(GetRochaHome(), "settings.json")
}

// ExpandPath expands ~ to home directory
func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			if len(path) == 1 {
				return homeDir
			}
			return filepath.Join(homeDir, path[1:])
		}
	}
	return path
}
