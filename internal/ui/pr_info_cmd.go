package ui

import (
	"context"
	"sync"
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

// BatchPRInfoSession represents a session to fetch PR info for
type BatchPRInfoSession struct {
	BranchName  string
	SessionName string
}

// BatchPRInfoRequest groups sessions by repo for batch fetching
type BatchPRInfoRequest struct {
	RepoPath string
	Sessions []BatchPRInfoSession
}

// BatchPRInfoReadyMsg contains PR info for multiple sessions
type BatchPRInfoReadyMsg struct {
	Results map[string]*domain.PRInfo // sessionName -> PRInfo
}

// StartBatchPRInfoFetcher fetches PRs for multiple repos in parallel.
// Each repo gets one `gh pr list` call, and results are matched to sessions by branch name.
func StartBatchPRInfoFetcher(gitService *services.GitService, requests []BatchPRInfoRequest) tea.Cmd {
	return func() tea.Msg {
		results := make(map[string]*domain.PRInfo)
		var wg sync.WaitGroup
		var mu sync.Mutex

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for _, req := range requests {
			wg.Add(1)
			go func(r BatchPRInfoRequest) {
				defer wg.Done()

				logging.Logger.Debug("Batch fetching PRs for repo", "repo", r.RepoPath, "sessions", len(r.Sessions))

				prMap, err := gitService.FetchAllPRs(ctx, r.RepoPath)
				if err != nil {
					logging.Logger.Warn("Failed to batch fetch PRs", "repo", r.RepoPath, "error", err)
					return
				}
				if prMap == nil {
					logging.Logger.Debug("gh CLI not available, skipping batch fetch")
					return
				}

				mu.Lock()
				for _, sess := range r.Sessions {
					if pr, ok := prMap[sess.BranchName]; ok {
						results[sess.SessionName] = pr
						logging.Logger.Debug("Matched PR to session",
							"session", sess.SessionName,
							"branch", sess.BranchName,
							"pr", pr.Number)
					}
				}
				mu.Unlock()
			}(req)
		}

		wg.Wait()
		logging.Logger.Debug("Batch PR fetch complete", "results", len(results))
		return BatchPRInfoReadyMsg{Results: results}
	}
}

// GroupSessionsByRepo groups sessions needing PR fetch by repository.
// Returns batch requests ready for StartBatchPRInfoFetcher.
func GroupSessionsByRepo(sessions map[string]domain.Session) []BatchPRInfoRequest {
	byRepo := make(map[string][]BatchPRInfoSession)

	for name, sess := range sessions {
		if sess.RepoPath == "" || sess.BranchName == "" {
			continue
		}
		byRepo[sess.RepoPath] = append(byRepo[sess.RepoPath], BatchPRInfoSession{
			BranchName:  sess.BranchName,
			SessionName: name,
		})
	}

	requests := make([]BatchPRInfoRequest, 0, len(byRepo))
	for repoPath, sessionList := range byRepo {
		requests = append(requests, BatchPRInfoRequest{
			RepoPath: repoPath,
			Sessions: sessionList,
		})
	}
	return requests
}
