package storage

import (
	"rocha/internal/domain"
)

// sessionModelToDomain converts a SessionModel (GORM) to domain.Session
func sessionModelToDomain(m SessionModel, isFlagged bool, status *string, comment string, isArchived bool, allowSkipPerms bool) domain.Session {
	return domain.Session{
		AllowDangerouslySkipPermissions: allowSkipPerms,
		BranchName:                      m.BranchName,
		ClaudeDir:                       m.ClaudeDir,
		Comment:                         comment,
		DisplayName:                     m.DisplayName,
		ExecutionID:                     m.ExecutionID,
		GitStats:                        nil, // Not persisted, populated at runtime
		IsArchived:                      isArchived,
		IsFlagged:                       isFlagged,
		LastUpdated:                     m.LastUpdated,
		Name:                            m.Name,
		RepoInfo:                        m.RepoInfo,
		RepoPath:                        m.RepoPath,
		RepoSource:                      m.RepoSource,
		ShellSession:                    nil, // Set separately if nested session exists
		State:                           domain.SessionState(m.State),
		Status:                          status,
		WorktreePath:                    m.WorktreePath,
	}
}

// domainToSessionModel converts a domain.Session to SessionModel (GORM)
func domainToSessionModel(s domain.Session) SessionModel {
	return SessionModel{
		BranchName:   s.BranchName,
		ClaudeDir:    s.ClaudeDir,
		DisplayName:  s.DisplayName,
		ExecutionID:  s.ExecutionID,
		LastUpdated:  s.LastUpdated,
		Name:         s.Name,
		RepoInfo:     s.RepoInfo,
		RepoPath:     s.RepoPath,
		RepoSource:   s.RepoSource,
		State:        string(s.State),
		WorktreePath: s.WorktreePath,
	}
}
