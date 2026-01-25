package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/services"
	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
)

// GitStatsRequest represents a request to fetch git stats
type GitStatsRequest struct {
	Priority     int // Higher priority = fetched first
	SessionName  string
	WorktreePath string
}

// GitStatsReadyMsg is sent when git stats are successfully fetched
type GitStatsReadyMsg struct {
	SessionName string
	Stats       *domain.GitStats
}

// GitStatsErrorMsg is sent when git stats fetching fails
type GitStatsErrorMsg struct {
	Err         error
	SessionName string
}

// StartGitStatsFetcher starts an async worker that fetches git stats
// Returns a tea.Cmd that will send GitStatsReadyMsg or GitStatsErrorMsg
func StartGitStatsFetcher(gitService *services.GitService, request GitStatsRequest) tea.Cmd {
	return func() tea.Msg {
		// Create context with 3 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Fetch stats
		stats, err := gitService.FetchGitStats(ctx, request.WorktreePath)
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
