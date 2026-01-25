package git

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"rocha/domain"
	"rocha/logging"

	"golang.org/x/sync/errgroup"
)

// FetchGitStats fetches all git statistics for the given worktree path
// Uses errgroup for concurrent fetching with context cancellation
func FetchGitStats(ctx context.Context, worktreePath string) (*domain.GitStats, error) {
	logging.Logger.Debug("Fetching git stats", "path", worktreePath)

	stats := &domain.GitStats{
		FetchedAt: time.Now(),
	}

	// Use errgroup for concurrent fetching
	g, ctx := errgroup.WithContext(ctx)

	// Fetch ahead/behind
	g.Go(func() error {
		ahead, behind, err := getAheadBehind(ctx, worktreePath)
		if err != nil {
			logging.Logger.Debug("Failed to get ahead/behind", "error", err)
			// Non-fatal - continue with other stats
			return nil
		}
		stats.Ahead = ahead
		stats.Behind = behind
		return nil
	})

	// Fetch file stats
	g.Go(func() error {
		additions, deletions, fileCount, err := getFileStats(ctx, worktreePath)
		if err != nil {
			logging.Logger.Debug("Failed to get file stats", "error", err)
			// Non-fatal - continue with other stats
			return nil
		}
		stats.Additions = additions
		stats.ChangedFiles = fileCount
		stats.Deletions = deletions
		return nil
	})


	// Wait for all fetches to complete
	if err := g.Wait(); err != nil {
		stats.Error = err
		return stats, err
	}

	logging.Logger.Debug("Git stats fetched successfully",
		"ahead", stats.Ahead,
		"behind", stats.Behind,
		"changedFiles", stats.ChangedFiles,
		"additions", stats.Additions,
		"deletions", stats.Deletions)

	return stats, nil
}

// getAheadBehind returns how many commits ahead and behind the tracking branch
func getAheadBehind(ctx context.Context, path string) (ahead int, behind int, err error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		// No tracking branch or other error
		return 0, 0, fmt.Errorf("git rev-list failed: %w", err)
	}

	// Parse output: "AHEAD	BEHIND"
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %s", output)
	}

	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse ahead count: %w", err)
	}

	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse behind count: %w", err)
	}

	return ahead, behind, nil
}

// getFileStats returns lines added, deleted, and number of changed files in working directory
func getFileStats(ctx context.Context, path string) (additions, deletions, fileCount int, err error) {
	// Get additions/deletions from git diff
	diffCmd := exec.CommandContext(ctx, "git", "diff", "--numstat", "HEAD")
	diffCmd.Dir = path

	diffOutput, err := diffCmd.Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("git diff failed: %w", err)
	}

	// Parse output: each line is "ADDED	DELETED	filename"
	lines := strings.Split(strings.TrimSpace(string(diffOutput)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// Handle binary files (shown as "-")
		if parts[0] != "-" {
			added, err := strconv.Atoi(parts[0])
			if err == nil {
				additions += added
			}
		}

		if parts[1] != "-" {
			deleted, err := strconv.Atoi(parts[1])
			if err == nil {
				deletions += deleted
			}
		}
	}

	// Get file count from git status (includes untracked files)
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = path

	statusOutput, err := statusCmd.Output()
	if err != nil {
		return additions, deletions, 0, fmt.Errorf("git status failed: %w", err)
	}

	statusLines := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
	for _, line := range statusLines {
		if line != "" {
			fileCount++
		}
	}

	return additions, deletions, fileCount, nil
}

// getLastCommit returns the last commit hash and message
func getLastCommit(ctx context.Context, path string) (hash string, message string, err error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%h %s")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("git log failed: %w", err)
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return "", "", fmt.Errorf("no commits found")
	}

	// Parse: "abc1234 commit message here"
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return parts[0], "", nil
	}

	hash = parts[0]
	message = parts[1]

	// Truncate message to 50 chars
	if len(message) > 50 {
		message = message[:47] + "..."
	}

	return hash, message, nil
}


