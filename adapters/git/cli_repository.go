package git

import (
	"rocha/domain"
	gitpkg "rocha/git"
	"rocha/ports"
)

// CLIRepository implements ports.GitRepository by wrapping the existing git package
type CLIRepository struct{}

// Verify interface compliance at compile time
var _ ports.GitRepository = (*CLIRepository)(nil)

// NewCLIRepository creates a new CLIRepository
func NewCLIRepository() *CLIRepository {
	return &CLIRepository{}
}

// RepoInspector methods

// IsGitRepo implements RepoInspector.IsGitRepo
func (r *CLIRepository) IsGitRepo(path string) (bool, string) {
	return gitpkg.IsGitRepo(path)
}

// GetMainRepoPath implements RepoInspector.GetMainRepoPath
func (r *CLIRepository) GetMainRepoPath(path string) (string, error) {
	return gitpkg.GetMainRepoPath(path)
}

// GetRepoInfo implements RepoInspector.GetRepoInfo
func (r *CLIRepository) GetRepoInfo(repoPath string) string {
	return gitpkg.GetRepoInfo(repoPath)
}

// GetBranchName implements RepoInspector.GetBranchName
func (r *CLIRepository) GetBranchName(path string) string {
	return gitpkg.GetBranchName(path)
}

// GetRemoteURL implements RepoInspector.GetRemoteURL
func (r *CLIRepository) GetRemoteURL(repoPath string) string {
	return gitpkg.GetRemoteURL(repoPath)
}

// WorktreeManager methods

// CreateWorktree implements WorktreeManager.CreateWorktree
func (r *CLIRepository) CreateWorktree(repoPath, worktreePath, branchName string) error {
	return gitpkg.CreateWorktree(repoPath, worktreePath, branchName)
}

// RemoveWorktree implements WorktreeManager.RemoveWorktree
func (r *CLIRepository) RemoveWorktree(repoPath, worktreePath string) error {
	return gitpkg.RemoveWorktree(repoPath, worktreePath)
}

// ListWorktrees implements WorktreeManager.ListWorktrees
func (r *CLIRepository) ListWorktrees(repoPath string) ([]string, error) {
	return gitpkg.ListWorktrees(repoPath)
}

// RepairWorktrees implements WorktreeManager.RepairWorktrees
func (r *CLIRepository) RepairWorktrees(mainRepoPath string, worktreePaths []string) error {
	return gitpkg.RepairWorktrees(mainRepoPath, worktreePaths)
}

// BuildWorktreePath implements WorktreeManager.BuildWorktreePath
func (r *CLIRepository) BuildWorktreePath(base, repoInfo, sessionName string) string {
	return gitpkg.BuildWorktreePath(base, repoInfo, sessionName)
}

// RepoCloner methods

// GetOrCloneRepository implements RepoCloner.GetOrCloneRepository
func (r *CLIRepository) GetOrCloneRepository(source, worktreeBase string) (string, *domain.RepoSource, error) {
	localPath, gitRepoSource, err := gitpkg.GetOrCloneRepository(source, worktreeBase)
	if err != nil {
		return "", nil, err
	}
	// Convert git.RepoSource to domain.RepoSource
	domainSource := &domain.RepoSource{
		Branch:   gitRepoSource.Branch,
		IsRemote: gitRepoSource.IsRemote,
		Owner:    gitRepoSource.Owner,
		Path:     gitRepoSource.Path,
		Repo:     gitRepoSource.Repo,
	}
	return localPath, domainSource, nil
}

// BranchValidator methods

// ValidateBranchName implements BranchValidator.ValidateBranchName
func (r *CLIRepository) ValidateBranchName(name string) error {
	return gitpkg.ValidateBranchName(name)
}

// SanitizeBranchName implements BranchValidator.SanitizeBranchName
func (r *CLIRepository) SanitizeBranchName(name string) (string, error) {
	return gitpkg.SanitizeBranchName(name)
}
