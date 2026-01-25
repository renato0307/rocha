package application

import "rocha/domain"

// CreateSessionParams contains parameters for creating a new session
type CreateSessionParams struct {
	AllowDangerouslySkipPermissions bool
	BranchNameOverride              string
	ClaudeDirOverride               string
	RepoSource                      string
	SessionName                     string
	TmuxStatusPosition              string
}

// CreateSessionResult contains the result of session creation
type CreateSessionResult struct {
	Session      *domain.Session
	WorktreePath string
}

// ClaudeDirResolver resolves the Claude configuration directory
type ClaudeDirResolver interface {
	Resolve(repoInfo, userOverride string) string
}
