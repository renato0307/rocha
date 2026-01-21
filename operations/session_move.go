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
	// Get session from source
	sessInfo, err := sourceStore.GetSession(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session %s: %w", sessionName, err)
	}

	// Kill tmux session (graceful failure - session might not be running)
	if err := killTmuxSession(sessionName); err != nil {
		// Log warning but continue - tmux session might not exist
		fmt.Printf("⚠ Warning: Failed to kill tmux session %s: %v\n", sessionName, err)
	}

	// Kill shell session if exists
	if sessInfo.ShellSession != nil {
		shellName := sessInfo.ShellSession.Name
		if err := killTmuxSession(shellName); err != nil {
			fmt.Printf("⚠ Warning: Failed to kill shell session %s: %v\n", shellName, err)
		}
	}

	// Update paths in session info
	updateSessionPaths(sessInfo, sourceRochaHome, destRochaHome)

	// Move worktree if it exists
	if sessInfo.WorktreePath != "" {
		sourceWorktree := strings.Replace(sessInfo.WorktreePath, destRochaHome, sourceRochaHome, 1)
		destWorktree := sessInfo.WorktreePath

		if err := moveWorktree(sourceWorktree, destWorktree); err != nil {
			// Log warning but continue - worktree might not exist
			fmt.Printf("⚠ Warning: Failed to move worktree for %s: %v\n", sessionName, err)
		}
	}

	// Add session to destination store
	if err := destStore.AddSession(ctx, *sessInfo); err != nil {
		return fmt.Errorf("failed to add session %s to destination: %w", sessionName, err)
	}

	return nil
}

// VerifySession confirms session exists in destination store
func VerifySession(
	ctx context.Context,
	sessionName string,
	destStore *storage.Store,
) error {
	_, err := destStore.GetSession(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("verification failed - session %s not found in destination: %w", sessionName, err)
	}
	return nil
}

// DeleteSession removes session from source (DB + worktree + tmux)
func DeleteSession(
	ctx context.Context,
	sessionName string,
	sourceStore *storage.Store,
) error {
	// Get session info before deleting (to get worktree path)
	sessInfo, err := sourceStore.GetSession(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session %s: %w", sessionName, err)
	}

	// Delete from database (cascade deletes extension tables)
	if err := sourceStore.DeleteSession(ctx, sessionName); err != nil {
		return fmt.Errorf("failed to delete session %s from source database: %w", sessionName, err)
	}

	// Clean up worktree if still exists at source
	if sessInfo.WorktreePath != "" && sessInfo.RepoPath != "" {
		if err := git.RemoveWorktree(sessInfo.RepoPath, sessInfo.WorktreePath); err != nil {
			// Log warning but don't fail - worktree might have been moved already
			fmt.Printf("⚠ Warning: Failed to remove source worktree for %s: %v\n", sessionName, err)
		}
	}

	return nil
}

// killTmuxSession kills a tmux session
func killTmuxSession(sessionName string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux kill failed (session may not exist): %w", err)
	}
	return nil
}

// updateSessionPaths updates WorktreePath and ClaudeDir in session info
func updateSessionPaths(sessInfo *storage.SessionInfo, sourceRochaHome, destRochaHome string) {
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

	// Update shell session paths if exists
	if sessInfo.ShellSession != nil {
		updateSessionPaths(sessInfo.ShellSession, sourceRochaHome, destRochaHome)
	}
}

// moveWorktree moves a worktree from source to destination
func moveWorktree(sourcePath, destPath string) error {
	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source worktree does not exist: %s", sourcePath)
	}

	// Create parent directories for destination
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Try atomic rename first (works if same filesystem)
	err := os.Rename(sourcePath, destPath)
	if err == nil {
		return nil
	}

	// If rename fails, fall back to copy + delete (for cross-filesystem moves)
	if err := copyDirectory(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to copy worktree: %w", err)
	}

	// Remove source after successful copy
	if err := os.RemoveAll(sourcePath); err != nil {
		return fmt.Errorf("failed to remove source worktree after copy: %w", err)
	}

	return nil
}

// copyDirectory recursively copies a directory
func copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
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

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
