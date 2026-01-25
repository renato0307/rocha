package ports

import "rocha/domain"

// RepoInspector queries repository information
type RepoInspector interface {
	GetBranchName(path string) string
	GetMainRepoPath(path string) (string, error)
	GetRemoteURL(repoPath string) string
	GetRepoInfo(repoPath string) string
	IsGitRepo(path string) (bool, string)
}

// WorktreeManager handles worktree lifecycle
type WorktreeManager interface {
	BuildWorktreePath(base, repoInfo, sessionName string) string
	CreateWorktree(repoPath, worktreePath, branchName string) error
	ListWorktrees(repoPath string) ([]string, error)
	RemoveWorktree(repoPath, worktreePath string) error
	RepairWorktrees(mainRepoPath string, worktreePaths []string) error
}

// RepoCloner handles repository cloning
type RepoCloner interface {
	GetOrCloneRepository(source, worktreeBase string) (string, *domain.RepoSource, error)
}

// BranchValidator validates and sanitizes branch names
type BranchValidator interface {
	SanitizeBranchName(name string) (string, error)
	ValidateBranchName(name string) error
}

// GitRepository is the composite interface
type GitRepository interface {
	BranchValidator
	RepoCloner
	RepoInspector
	WorktreeManager
}
