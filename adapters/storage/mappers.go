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

	// GitStats is not persisted - it's populated at runtime by the UI
	// Type assertion to convert from interface{} to *domain.GitStats
	var gitStats *domain.GitStats
	if info.GitStats != nil {
		if stats, ok := info.GitStats.(*domain.GitStats); ok {
			gitStats = stats
		}
	}

	return domain.Session{
		AllowDangerouslySkipPermissions: info.AllowDangerouslySkipPermissions,
		BranchName:                      info.BranchName,
		ClaudeDir:                       info.ClaudeDir,
		Comment:                         info.Comment,
		DisplayName:                     info.DisplayName,
		ExecutionID:                     info.ExecutionID,
		GitStats:                        gitStats,
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

// sessionStateToDomain converts storage.SessionState to domain.SessionCollection
func sessionStateToDomain(state *existingstore.SessionState) *domain.SessionCollection {
	if state == nil {
		return &domain.SessionCollection{
			OrderedNames: []string{},
			Sessions:     make(map[string]domain.Session),
		}
	}

	sessions := make(map[string]domain.Session)
	for name, info := range state.Sessions {
		sessions[name] = sessionInfoToDomain(info)
	}

	return &domain.SessionCollection{
		OrderedNames: state.OrderedSessionNames,
		Sessions:     sessions,
	}
}

// domainToSessionState converts domain.SessionCollection to storage.SessionState
func domainToSessionState(collection *domain.SessionCollection) *existingstore.SessionState {
	if collection == nil {
		return &existingstore.SessionState{
			OrderedSessionNames: []string{},
			Sessions:            make(map[string]existingstore.SessionInfo),
		}
	}

	sessions := make(map[string]existingstore.SessionInfo)
	for name, session := range collection.Sessions {
		sessions[name] = domainToSessionInfo(session)
	}

	return &existingstore.SessionState{
		OrderedSessionNames: collection.OrderedNames,
		Sessions:            sessions,
	}
}
