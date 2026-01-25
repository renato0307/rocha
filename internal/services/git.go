package services

import (
	"context"

	"rocha/internal/domain"
	"rocha/internal/ports"
)

// GitService provides git operations for the UI layer
type GitService struct {
	gitRepo ports.GitRepository
}

// NewGitService creates a new GitService
func NewGitService(gitRepo ports.GitRepository) *GitService {
	return &GitService{
		gitRepo: gitRepo,
	}
}

// IsGitRepo checks if path is a git repository
// Returns (isGit, repoRoot)
func (s *GitService) IsGitRepo(path string) (bool, string) {
	return s.gitRepo.IsGitRepo(path)
}

// GetRemoteURL gets the remote URL for a repository
func (s *GitService) GetRemoteURL(repoPath string) string {
	return s.gitRepo.GetRemoteURL(repoPath)
}

// GetRepoInfo extracts owner/repo from repository
func (s *GitService) GetRepoInfo(repoPath string) string {
	return s.gitRepo.GetRepoInfo(repoPath)
}

// ParseRepoSource parses a repository source (URL or path)
func (s *GitService) ParseRepoSource(source string) (*domain.RepoSource, error) {
	return s.gitRepo.ParseRepoSource(source)
}

// IsGitURL checks if source is a git URL
func (s *GitService) IsGitURL(source string) bool {
	return s.gitRepo.IsGitURL(source)
}

// SanitizeBranchName sanitizes a branch name
func (s *GitService) SanitizeBranchName(name string) (string, error) {
	return s.gitRepo.SanitizeBranchName(name)
}

// ValidateBranchName validates a branch name
func (s *GitService) ValidateBranchName(name string) error {
	return s.gitRepo.ValidateBranchName(name)
}

// RemoveWorktree removes a git worktree
func (s *GitService) RemoveWorktree(repoPath, worktreePath string) error {
	return s.gitRepo.RemoveWorktree(repoPath, worktreePath)
}

// FetchGitStats fetches git statistics for a path
func (s *GitService) FetchGitStats(ctx context.Context, worktreePath string) (*domain.GitStats, error) {
	return s.gitRepo.FetchGitStats(ctx, worktreePath)
}

// GetMainRepoPath gets the main repository path (handles worktrees correctly)
func (s *GitService) GetMainRepoPath(path string) (string, error) {
	return s.gitRepo.GetMainRepoPath(path)
}

// GetBranchName gets the current branch name for a path
func (s *GitService) GetBranchName(path string) string {
	return s.gitRepo.GetBranchName(path)
}
