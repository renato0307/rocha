package operations

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"rocha/git"
	"rocha/logging"
	"rocha/storage"
)

// MoveSession handles moving a single session between stores
// It copies the session data and moves the worktree (if it exists)
func MoveSession(
	ctx context.Context,
	sessionName string,
	sourceStore *storage.Store,
	destStore *storage.Store,
	sourceRochaHome string,
	destRochaHome string,
) error {
	logging.Logger.Info("Moving session", "session", sessionName, "from", sourceRochaHome, "to", destRochaHome)

	// Get session from source
	sessInfo, err := sourceStore.GetSession(ctx, sessionName)
	if err != nil {
		logging.Logger.Error("Failed to get session from source", "session", sessionName, "error", err)
		return fmt.Errorf("failed to get session %s: %w", sessionName, err)
	}
	logging.Logger.Debug("Retrieved session info", "session", sessionName, "worktreePath", sessInfo.WorktreePath)

	// Kill tmux session (graceful failure - session might not be running)
	logging.Logger.Debug("Killing tmux session", "session", sessionName)
	if err := KillTmuxSession(sessionName); err != nil {
		// Log warning but continue - tmux session might not exist
		logging.Logger.Warn("Failed to kill tmux session", "session", sessionName, "error", err)
		fmt.Printf("⚠ Warning: Failed to kill tmux session %s: %v\n", sessionName, err)
	}

	// Kill shell session if exists
	if sessInfo.ShellSession != nil {
		shellName := sessInfo.ShellSession.Name
		logging.Logger.Debug("Killing shell session", "session", shellName)
		if err := KillTmuxSession(shellName); err != nil {
			logging.Logger.Warn("Failed to kill shell session", "session", shellName, "error", err)
			fmt.Printf("⚠ Warning: Failed to kill shell session %s: %v\n", shellName, err)
		}
	}

	// Update paths in session info
	logging.Logger.Debug("Updating session paths", "session", sessionName)
	updateSessionPaths(sessInfo, sourceRochaHome, destRochaHome)
	logging.Logger.Debug("Updated paths", "session", sessionName, "newWorktreePath", sessInfo.WorktreePath, "newClaudeDir", sessInfo.ClaudeDir)

	// Move worktree if it exists
	if sessInfo.WorktreePath != "" {
		sourceWorktree := strings.Replace(sessInfo.WorktreePath, destRochaHome, sourceRochaHome, 1)
		destWorktree := sessInfo.WorktreePath

		logging.Logger.Info("Moving worktree", "session", sessionName, "from", sourceWorktree, "to", destWorktree)
		if err := moveWorktree(sourceWorktree, destWorktree); err != nil {
			// Log warning but continue - worktree might not exist
			logging.Logger.Warn("Failed to move worktree", "session", sessionName, "error", err)
			fmt.Printf("⚠ Warning: Failed to move worktree for %s: %v\n", sessionName, err)
		} else {
			logging.Logger.Info("Worktree moved successfully", "session", sessionName)
		}
	}

	// Add session to destination store
	logging.Logger.Debug("Adding session to destination store", "session", sessionName)
	if err := destStore.AddSession(ctx, *sessInfo); err != nil {
		logging.Logger.Error("Failed to add session to destination", "session", sessionName, "error", err)
		return fmt.Errorf("failed to add session %s to destination: %w", sessionName, err)
	}

	logging.Logger.Info("Session moved successfully", "session", sessionName)
	return nil
}

// VerifySession confirms session exists in destination store
func VerifySession(
	ctx context.Context,
	sessionName string,
	destStore *storage.Store,
) error {
	logging.Logger.Debug("Verifying session in destination", "session", sessionName)
	_, err := destStore.GetSession(ctx, sessionName)
	if err != nil {
		logging.Logger.Error("Verification failed - session not found in destination", "session", sessionName, "error", err)
		return fmt.Errorf("verification failed - session %s not found in destination: %w", sessionName, err)
	}
	logging.Logger.Info("Session verified successfully", "session", sessionName)
	return nil
}

// DeleteSessionOptions configures session deletion behavior
type DeleteSessionOptions struct {
	KillTmux       bool // Kill tmux sessions before deleting
	RemoveWorktree bool // Remove worktree from filesystem
}

// DeleteSession removes session from database with optional tmux kill and worktree removal
func DeleteSession(
	ctx context.Context,
	sessionName string,
	store *storage.Store,
	opts DeleteSessionOptions,
) error {
	logging.Logger.Info("Deleting session", "session", sessionName, "killTmux", opts.KillTmux, "removeWorktree", opts.RemoveWorktree)

	// Get session info before deleting (to get worktree path and shell session)
	sessInfo, err := store.GetSession(ctx, sessionName)
	if err != nil {
		logging.Logger.Error("Failed to get session for deletion", "session", sessionName, "error", err)
		return fmt.Errorf("failed to get session %s: %w", sessionName, err)
	}

	// Kill tmux sessions if requested
	if opts.KillTmux {
		logging.Logger.Debug("Killing tmux sessions", "session", sessionName)
		// Kill shell session if exists
		if sessInfo.ShellSession != nil {
			logging.Logger.Debug("Killing shell session", "session", sessInfo.ShellSession.Name)
			if err := KillTmuxSession(sessInfo.ShellSession.Name); err != nil {
				logging.Logger.Warn("Failed to kill shell session", "session", sessInfo.ShellSession.Name, "error", err)
				fmt.Printf("⚠ Warning: Failed to kill shell session %s: %v\n", sessInfo.ShellSession.Name, err)
			}
		}

		// Kill main session
		if err := KillTmuxSession(sessionName); err != nil {
			logging.Logger.Warn("Failed to kill tmux session", "session", sessionName, "error", err)
			fmt.Printf("⚠ Warning: Failed to kill tmux session %s: %v\n", sessionName, err)
		}
	}

	// Delete from database (cascade deletes extension tables)
	logging.Logger.Debug("Deleting session from database", "session", sessionName)
	if err := store.DeleteSession(ctx, sessionName); err != nil {
		logging.Logger.Error("Failed to delete session from database", "session", sessionName, "error", err)
		return fmt.Errorf("failed to delete session %s from database: %w", sessionName, err)
	}

	// Remove worktree if requested and exists
	if opts.RemoveWorktree && sessInfo.WorktreePath != "" && sessInfo.RepoPath != "" {
		logging.Logger.Info("Removing worktree", "session", sessionName, "path", sessInfo.WorktreePath)
		if err := git.RemoveWorktree(sessInfo.RepoPath, sessInfo.WorktreePath); err != nil {
			logging.Logger.Warn("Failed to remove worktree", "session", sessionName, "path", sessInfo.WorktreePath, "error", err)
			fmt.Printf("⚠ Warning: Failed to remove worktree for %s: %v\n", sessionName, err)
		} else {
			logging.Logger.Info("Worktree removed successfully", "session", sessionName)
		}
	}

	logging.Logger.Info("Session deleted successfully", "session", sessionName)
	return nil
}

// KillTmuxSession kills a tmux session
func KillTmuxSession(sessionName string) error {
	logging.Logger.Debug("Killing tmux session", "session", sessionName)
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		logging.Logger.Debug("Tmux kill command failed", "session", sessionName, "error", err)
		return fmt.Errorf("tmux kill failed (session may not exist): %w", err)
	}
	logging.Logger.Debug("Tmux session killed successfully", "session", sessionName)
	return nil
}

// updateSessionPaths updates WorktreePath and ClaudeDir in session info
func updateSessionPaths(sessInfo *storage.SessionInfo, sourceRochaHome, destRochaHome string) {
	logging.Logger.Debug("Updating session paths", "session", sessInfo.Name, "from", sourceRochaHome, "to", destRochaHome)

	oldWorktreePath := sessInfo.WorktreePath
	oldClaudeDir := sessInfo.ClaudeDir

	// Update WorktreePath
	if sessInfo.WorktreePath != "" {
		sessInfo.WorktreePath = strings.Replace(
			sessInfo.WorktreePath,
			sourceRochaHome,
			destRochaHome,
			1,
		)
	}

	// Update ClaudeDir if it references sourceRochaHome
	if sessInfo.ClaudeDir != "" && strings.Contains(sessInfo.ClaudeDir, sourceRochaHome) {
		sessInfo.ClaudeDir = strings.Replace(
			sessInfo.ClaudeDir,
			sourceRochaHome,
			destRochaHome,
			1,
		)
	}

	logging.Logger.Debug("Paths updated",
		"session", sessInfo.Name,
		"worktreePath", oldWorktreePath, "→", sessInfo.WorktreePath,
		"claudeDir", oldClaudeDir, "→", sessInfo.ClaudeDir)

	// Update shell session paths if exists
	if sessInfo.ShellSession != nil {
		updateSessionPaths(sessInfo.ShellSession, sourceRochaHome, destRochaHome)
	}
}

// moveWorktree moves a worktree from source to destination
func moveWorktree(sourcePath, destPath string) error {
	logging.Logger.Info("Moving worktree", "from", sourcePath, "to", destPath)

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		logging.Logger.Warn("Source worktree does not exist", "path", sourcePath)
		return fmt.Errorf("source worktree does not exist: %s", sourcePath)
	}

	// Create parent directories for destination
	logging.Logger.Debug("Creating destination directory", "path", filepath.Dir(destPath))
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		logging.Logger.Error("Failed to create destination directory", "path", filepath.Dir(destPath), "error", err)
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Try atomic rename first (works if same filesystem)
	logging.Logger.Debug("Attempting atomic rename", "from", sourcePath, "to", destPath)
	err := os.Rename(sourcePath, destPath)
	if err == nil {
		logging.Logger.Info("Worktree moved using atomic rename", "from", sourcePath, "to", destPath)
		return nil
	}

	// If rename fails, fall back to copy + delete (for cross-filesystem moves)
	logging.Logger.Debug("Atomic rename failed, falling back to copy+delete", "error", err)
	if err := copyDirectory(sourcePath, destPath); err != nil {
		logging.Logger.Error("Failed to copy worktree", "from", sourcePath, "to", destPath, "error", err)
		return fmt.Errorf("failed to copy worktree: %w", err)
	}

	// Remove source after successful copy
	logging.Logger.Debug("Removing source worktree after copy", "path", sourcePath)
	if err := os.RemoveAll(sourcePath); err != nil {
		logging.Logger.Error("Failed to remove source worktree after copy", "path", sourcePath, "error", err)
		return fmt.Errorf("failed to remove source worktree after copy: %w", err)
	}

	logging.Logger.Info("Worktree moved using copy+delete", "from", sourcePath, "to", destPath)
	return nil
}

// copyDirectory recursively copies a directory
func copyDirectory(src, dst string) error {
	logging.Logger.Debug("Copying directory", "from", src, "to", dst)

	entries, err := os.ReadDir(src)
	if err != nil {
		logging.Logger.Error("Failed to read source directory", "path", src, "error", err)
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		logging.Logger.Error("Failed to create destination directory", "path", dst, "error", err)
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	logging.Logger.Debug("Directory copied successfully", "from", src, "to", dst)
	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	logging.Logger.Debug("Copying file", "from", src, "to", dst)

	sourceFile, err := os.Open(src)
	if err != nil {
		logging.Logger.Error("Failed to open source file", "path", src, "error", err)
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		logging.Logger.Error("Failed to create destination file", "path", dst, "error", err)
		return err
	}
	defer destFile.Close()

	bytesWritten, err := io.Copy(destFile, sourceFile)
	if err != nil {
		logging.Logger.Error("Failed to copy file contents", "from", src, "to", dst, "error", err)
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		logging.Logger.Error("Failed to get source file info", "path", src, "error", err)
		return err
	}

	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		logging.Logger.Error("Failed to set file permissions", "path", dst, "mode", sourceInfo.Mode(), "error", err)
		return err
	}

	logging.Logger.Debug("File copied successfully", "from", src, "to", dst, "bytes", bytesWritten)
	return nil
}
