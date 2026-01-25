package git

import (
	"context"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/ports"
)

// CLIRepository implements ports.GitRepository using local git commands
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
	return isGitRepo(path)
}

// GetMainRepoPath implements RepoInspector.GetMainRepoPath
func (r *CLIRepository) GetMainRepoPath(path string) (string, error) {
	return getMainRepoPath(path)
}

// GetRepoInfo implements RepoInspector.GetRepoInfo
func (r *CLIRepository) GetRepoInfo(repoPath string) string {
	return getRepoInfo(repoPath)
}

// GetBranchName implements RepoInspector.GetBranchName
func (r *CLIRepository) GetBranchName(path string) string {
	return getBranchName(path)
}

// GetRemoteURL implements RepoInspector.GetRemoteURL
func (r *CLIRepository) GetRemoteURL(repoPath string) string {
	return getRemoteURL(repoPath)
}

// WorktreeManager methods

// CreateWorktree implements WorktreeManager.CreateWorktree
func (r *CLIRepository) CreateWorktree(repoPath, worktreePath, branchName string) error {
	return createWorktree(repoPath, worktreePath, branchName)
}

// RemoveWorktree implements WorktreeManager.RemoveWorktree
func (r *CLIRepository) RemoveWorktree(repoPath, worktreePath string) error {
	return removeWorktree(repoPath, worktreePath)
}

// ListWorktrees implements WorktreeManager.ListWorktrees
func (r *CLIRepository) ListWorktrees(repoPath string) ([]string, error) {
	return listWorktrees(repoPath)
}

// RepairWorktrees implements WorktreeManager.RepairWorktrees
func (r *CLIRepository) RepairWorktrees(mainRepoPath string, worktreePaths []string) error {
	return repairWorktrees(mainRepoPath, worktreePaths)
}

// BuildWorktreePath implements WorktreeManager.BuildWorktreePath
func (r *CLIRepository) BuildWorktreePath(base, repoInfo, sessionName string) string {
	return buildWorktreePath(base, repoInfo, sessionName)
}

// RepoCloner methods

// GetOrCloneRepository implements RepoCloner.GetOrCloneRepository
func (r *CLIRepository) GetOrCloneRepository(source, worktreeBase string) (string, *domain.RepoSource, error) {
	localPath, rs, err := getOrCloneRepository(source, worktreeBase)
	if err != nil {
		return "", nil, err
	}
	return localPath, repoSourceToDomain(rs), nil
}

// BranchValidator methods

// ValidateBranchName implements BranchValidator.ValidateBranchName
func (r *CLIRepository) ValidateBranchName(name string) error {
	return validateBranchName(name)
}

// SanitizeBranchName implements BranchValidator.SanitizeBranchName
func (r *CLIRepository) SanitizeBranchName(name string) (string, error) {
	return sanitizeBranchName(name)
}

// RepoSourceParser methods

// IsGitURL implements RepoSourceParser.IsGitURL
func (r *CLIRepository) IsGitURL(source string) bool {
	return isGitURL(source)
}

// ParseRepoSource implements RepoSourceParser.ParseRepoSource
func (r *CLIRepository) ParseRepoSource(source string) (*domain.RepoSource, error) {
	rs, err := parseRepoSource(source)
	if err != nil {
		return nil, err
	}
	return repoSourceToDomain(rs), nil
}

// GitStatsProvider methods

// FetchGitStats implements GitStatsProvider.FetchGitStats
func (r *CLIRepository) FetchGitStats(ctx context.Context, worktreePath string) (*domain.GitStats, error) {
	return fetchGitStats(ctx, worktreePath)
}

// repoSourceToDomain converts local repoSource to domain.RepoSource
func repoSourceToDomain(rs *repoSource) *domain.RepoSource {
	if rs == nil {
		return nil
	}
	return &domain.RepoSource{
		Branch:   rs.branch,
		IsRemote: rs.isRemote,
		Owner:    rs.owner,
		Path:     rs.path,
		Repo:     rs.repo,
	}
}
