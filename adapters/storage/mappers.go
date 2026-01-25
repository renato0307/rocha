package storage

import (
	"rocha/domain"
	existingstore "rocha/storage"
)

// sessionInfoToDomain converts a storage.SessionInfo to domain.Session
func sessionInfoToDomain(info existingstore.SessionInfo) domain.Session {
	var shellSession *domain.Session
	if info.ShellSession != nil {
		converted := sessionInfoToDomain(*info.ShellSession)
		shellSession = &converted
	}

	return domain.Session{
		AllowDangerouslySkipPermissions: info.AllowDangerouslySkipPermissions,
		BranchName:                      info.BranchName,
		ClaudeDir:                       info.ClaudeDir,
		Comment:                         info.Comment,
		DisplayName:                     info.DisplayName,
		ExecutionID:                     info.ExecutionID,
		GitStats:                        info.GitStats,
		IsArchived:                      info.IsArchived,
		IsFlagged:                       info.IsFlagged,
		LastUpdated:                     info.LastUpdated,
		Name:                            info.Name,
		RepoInfo:                        info.RepoInfo,
		RepoPath:                        info.RepoPath,
		RepoSource:                      info.RepoSource,
		ShellSession:                    shellSession,
		State:                           domain.SessionState(info.State),
		Status:                          info.Status,
		WorktreePath:                    info.WorktreePath,
	}
}

// domainToSessionInfo converts a domain.Session to storage.SessionInfo
func domainToSessionInfo(session domain.Session) existingstore.SessionInfo {
	var shellSession *existingstore.SessionInfo
	if session.ShellSession != nil {
		converted := domainToSessionInfo(*session.ShellSession)
		shellSession = &converted
	}

	return existingstore.SessionInfo{
		AllowDangerouslySkipPermissions: session.AllowDangerouslySkipPermissions,
		BranchName:                      session.BranchName,
		ClaudeDir:                       session.ClaudeDir,
		Comment:                         session.Comment,
		DisplayName:                     session.DisplayName,
		ExecutionID:                     session.ExecutionID,
		GitStats:                        session.GitStats,
		IsArchived:                      session.IsArchived,
		IsFlagged:                       session.IsFlagged,
		LastUpdated:                     session.LastUpdated,
		Name:                            session.Name,
		RepoInfo:                        session.RepoInfo,
		RepoPath:                        session.RepoPath,
		RepoSource:                      session.RepoSource,
		ShellSession:                    shellSession,
		State:                           string(session.State),
		Status:                          session.Status,
		WorktreePath:                    session.WorktreePath,
	}
}
