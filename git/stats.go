package git

import (
	"context"
	"fmt"
	"os/exec"
	"rocha/logging"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"
)

// GitStats holds detailed git statistics for a worktree
type GitStats struct {
	Ahead         int       // Commits ahead of tracking branch
	Behind        int       // Commits behind tracking branch
	Additions     int       // Lines added in working directory
	Deletions     int       // Lines deleted in working directory
	CommitHash    string    // Last commit hash (short)
	CommitMessage string    // Last commit message (truncated)
	PRNumber      string    // PR number if branch has associated PR (optional)
	Error         error     // Error during fetching (if any)
	FetchedAt     time.Time // When these stats were fetched
}

// GitStatsRequest represents a request to fetch git stats
type GitStatsRequest struct {
	SessionName  string
	WorktreePath string
	Priority     int // Higher priority = fetched first
}

// GitStatsReadyMsg is sent when git stats are successfully fetched
type GitStatsReadyMsg struct {
	SessionName string
	Stats       *GitStats
}

// GitStatsErrorMsg is sent when git stats fetching fails
type GitStatsErrorMsg struct {
	SessionName string
	Err         error
}

// FetchGitStats fetches all git statistics for the given worktree path
// Uses errgroup for concurrent fetching with context cancellation
func FetchGitStats(ctx context.Context, worktreePath string) (*GitStats, error) {
	logging.Logger.Debug("Fetching git stats", "path", worktreePath)

	stats := &GitStats{
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
		additions, deletions, err := getFileStats(ctx, worktreePath)
		if err != nil {
			logging.Logger.Debug("Failed to get file stats", "error", err)
			// Non-fatal - continue with other stats
			return nil
		}
		stats.Additions = additions
		stats.Deletions = deletions
		return nil
	})

	// Fetch last commit
	g.Go(func() error {
		hash, message, err := getLastCommit(ctx, worktreePath)
		if err != nil {
			logging.Logger.Debug("Failed to get last commit", "error", err)
			// Non-fatal - continue with other stats
			return nil
		}
		stats.CommitHash = hash
		stats.CommitMessage = message
		return nil
	})

	// Fetch PR number (optional, requires gh CLI)
	g.Go(func() error {
		prNumber, err := getPRNumber(ctx, worktreePath)
		if err != nil {
			logging.Logger.Debug("Failed to get PR number", "error", err)
			// Non-fatal - PR info is optional
			return nil
		}
		stats.PRNumber = prNumber
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
		"additions", stats.Additions,
		"deletions", stats.Deletions,
		"commit", stats.CommitHash,
		"pr", stats.PRNumber)

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

// getFileStats returns lines added and deleted in working directory
func getFileStats(ctx context.Context, path string) (additions int, deletions int, err error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--numstat", "HEAD")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("git diff failed: %w", err)
	}

	// Parse output: each line is "ADDED	DELETED	filename"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
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

	return additions, deletions, nil
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

// getPRNumber returns the PR number for the current branch (requires gh CLI)
func getPRNumber(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", "--json", "number", "-q", ".number")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		// gh CLI not installed or no PR for this branch
		return "", fmt.Errorf("gh pr view failed: %w", err)
	}

	prNumber := strings.TrimSpace(string(output))
	if prNumber == "" {
		return "", fmt.Errorf("no PR number found")
	}

	return prNumber, nil
}

// StartGitStatsFetcher starts an async worker that fetches git stats
// Returns a tea.Cmd that will send GitStatsReadyMsg or GitStatsErrorMsg
func StartGitStatsFetcher(request GitStatsRequest) tea.Cmd {
	return func() tea.Msg {
		// Create context with 3 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Fetch stats
		stats, err := FetchGitStats(ctx, request.WorktreePath)
		if err != nil {
			logging.Logger.Warn("Failed to fetch git stats",
				"session", request.SessionName,
				"error", err)
			return GitStatsErrorMsg{
				SessionName: request.SessionName,
				Err:         err,
			}
		}

		return GitStatsReadyMsg{
			SessionName: request.SessionName,
			Stats:       stats,
		}
	}
}
