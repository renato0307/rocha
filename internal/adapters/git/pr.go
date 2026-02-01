package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
)

const prInfoFetchTimeout = 5 * time.Second

// ghPRResponse represents the JSON response from gh pr view
type ghPRResponse struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// fetchPRInfo fetches PR information for a branch using gh CLI.
// Returns (nil, nil) if gh CLI is not installed.
// Returns (PRInfo with Number=0, nil) if no PR exists for the branch.
func fetchPRInfo(ctx context.Context, worktreePath, branchName string) (*domain.PRInfo, error) {
	logging.Logger.Debug("Fetching PR info", "path", worktreePath, "branch", branchName)

	// Check if gh is available
	if _, err := exec.LookPath("gh"); err != nil {
		logging.Logger.Debug("gh CLI not found, skipping PR fetch")
		return nil, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, prInfoFetchTimeout)
	defer cancel()

	// Run gh pr view for the branch
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", branchName, "--json", "number,state,url")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		// Check if it's just "no PR found" error (exit code 1)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				logging.Logger.Debug("No PR found for branch", "branch", branchName)
				return &domain.PRInfo{
					CheckedAt: time.Now().UTC(),
					Number:    0,
					State:     "",
					URL:       "",
				}, nil
			}
		}
		logging.Logger.Debug("gh pr view failed", "error", err)
		return nil, fmt.Errorf("gh pr view failed: %w", err)
	}

	var resp ghPRResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		logging.Logger.Debug("Failed to parse gh pr view output", "error", err, "output", string(output))
		return nil, fmt.Errorf("failed to parse gh response: %w", err)
	}

	logging.Logger.Debug("Fetched PR info", "number", resp.Number, "state", resp.State, "url", resp.URL)

	return &domain.PRInfo{
		CheckedAt: time.Now().UTC(),
		Number:    resp.Number,
		State:     resp.State,
		URL:       resp.URL,
	}, nil
}

// openPRInBrowser opens the PR URL in the default browser using gh CLI
func openPRInBrowser(worktreePath string) error {
	logging.Logger.Debug("Opening PR in browser", "path", worktreePath)

	// Check if gh is available
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found")
	}

	cmd := exec.Command("gh", "pr", "view", "--web")
	cmd.Dir = worktreePath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr view --web failed: %w", err)
	}

	return nil
}
