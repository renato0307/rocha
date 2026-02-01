package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/renato0307/rocha/internal/domain"
)

func TestGroupSessionsByRepo(t *testing.T) {
	tests := []struct {
		name             string
		sessions         map[string]domain.Session
		expectedRepos    int
		expectedSessions map[string]int // repoPath -> expected session count
	}{
		{
			name:             "empty sessions",
			sessions:         map[string]domain.Session{},
			expectedRepos:    0,
			expectedSessions: map[string]int{},
		},
		{
			name: "session without repo path is skipped",
			sessions: map[string]domain.Session{
				"session1": {
					BranchName: "feature-1",
					Name:       "session1",
					RepoPath:   "",
				},
			},
			expectedRepos:    0,
			expectedSessions: map[string]int{},
		},
		{
			name: "session without branch name is skipped",
			sessions: map[string]domain.Session{
				"session1": {
					BranchName: "",
					Name:       "session1",
					RepoPath:   "/path/to/repo",
				},
			},
			expectedRepos:    0,
			expectedSessions: map[string]int{},
		},
		{
			name: "single valid session",
			sessions: map[string]domain.Session{
				"session1": {
					BranchName: "feature-1",
					Name:       "session1",
					RepoPath:   "/path/to/repo",
				},
			},
			expectedRepos:    1,
			expectedSessions: map[string]int{"/path/to/repo": 1},
		},
		{
			name: "multiple sessions same repo",
			sessions: map[string]domain.Session{
				"session1": {
					BranchName: "feature-1",
					Name:       "session1",
					RepoPath:   "/path/to/repo",
				},
				"session2": {
					BranchName: "feature-2",
					Name:       "session2",
					RepoPath:   "/path/to/repo",
				},
			},
			expectedRepos:    1,
			expectedSessions: map[string]int{"/path/to/repo": 2},
		},
		{
			name: "multiple repos",
			sessions: map[string]domain.Session{
				"session1": {
					BranchName: "feature-1",
					Name:       "session1",
					RepoPath:   "/path/to/repo1",
				},
				"session2": {
					BranchName: "feature-2",
					Name:       "session2",
					RepoPath:   "/path/to/repo2",
				},
				"session3": {
					BranchName: "feature-3",
					Name:       "session3",
					RepoPath:   "/path/to/repo1",
				},
			},
			expectedRepos: 2,
			expectedSessions: map[string]int{
				"/path/to/repo1": 2,
				"/path/to/repo2": 1,
			},
		},
		{
			name: "mixed valid and invalid sessions",
			sessions: map[string]domain.Session{
				"valid": {
					BranchName: "feature-1",
					Name:       "valid",
					RepoPath:   "/path/to/repo",
				},
				"no-branch": {
					BranchName: "",
					Name:       "no-branch",
					RepoPath:   "/path/to/repo",
				},
				"no-repo": {
					BranchName: "feature-2",
					Name:       "no-repo",
					RepoPath:   "",
				},
			},
			expectedRepos:    1,
			expectedSessions: map[string]int{"/path/to/repo": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupSessionsByRepo(tt.sessions)

			assert.Len(t, result, tt.expectedRepos)

			// Build map for easier assertion (order not guaranteed)
			byRepo := make(map[string]int)
			for _, req := range result {
				byRepo[req.RepoPath] = len(req.Sessions)
			}

			for repoPath, expectedCount := range tt.expectedSessions {
				assert.Equal(t, expectedCount, byRepo[repoPath], "repo %s", repoPath)
			}
		})
	}
}
