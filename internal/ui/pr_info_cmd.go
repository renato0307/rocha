package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/services"
)

// PRInfoRequest represents a request to fetch PR info
type PRInfoRequest struct {
	BranchName   string
	SessionName  string
	WorktreePath string
}

// PRInfoReadyMsg is sent when PR info is successfully fetched
type PRInfoReadyMsg struct {
	PRInfo      *domain.PRInfo
	SessionName string
}

// PRInfoErrorMsg is sent when PR info fetching fails
type PRInfoErrorMsg struct {
	Err         error
	SessionName string
}

// StartPRInfoFetcher starts an async worker that fetches PR info
// Returns a tea.Cmd that will send PRInfoReadyMsg or PRInfoErrorMsg
func StartPRInfoFetcher(gitService *services.GitService, request PRInfoRequest) tea.Cmd {
	return func() tea.Msg {
		// Create context with 5 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Fetch PR info
		prInfo, err := gitService.FetchPRInfo(ctx, request.WorktreePath, request.BranchName)
		if err != nil {
			logging.Logger.Warn("Failed to fetch PR info",
				"session", request.SessionName,
				"branch", request.BranchName,
				"error", err)
			return PRInfoErrorMsg{
				SessionName: request.SessionName,
				Err:         err,
			}
		}

		logging.Logger.Debug("Fetched PR info",
			"session", request.SessionName,
			"number", prInfo.Number,
			"state", prInfo.State)

		return PRInfoReadyMsg{
			SessionName: request.SessionName,
			PRInfo:      prInfo,
		}
	}
}
