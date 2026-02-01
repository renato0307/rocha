package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/ports"
	portsmocks "github.com/renato0307/rocha/internal/ports/mocks"
	servicesmocks "github.com/renato0307/rocha/internal/services/mocks"
)

func TestCreateSession_ReusesExistingWorktree(t *testing.T) {
	existingWorktreePath := "/path/to/existing/worktree"

	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	// Setup expectations
	gitRepo.EXPECT().GetOrCloneRepository(mock.Anything, mock.Anything).
		Return("/path/to/repo", &domain.RepoSource{Owner: "test", Repo: "repo"}, nil)
	gitRepo.EXPECT().GetWorktreeForBranch("/path/to/repo", "feature-branch").
		Return(existingWorktreePath, nil)

	claudeDirResolver.EXPECT().Resolve("test/repo", mock.Anything).Return("/tmp/claude")

	tmuxClient.EXPECT().CreateSession(mock.Anything, existingWorktreePath, mock.Anything, mock.Anything, mock.Anything).
		Return(&ports.TmuxSession{Name: "test-session"}, nil)

	sessionRepo.EXPECT().Add(mock.Anything, mock.Anything).Return(nil)

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	result, err := service.CreateSession(context.Background(), CreateSessionParams{
		SessionName:        "test-session",
		BranchNameOverride: "feature-branch",
		RepoSource:         "https://github.com/test/repo",
	})

	require.NoError(t, err)
	assert.Equal(t, existingWorktreePath, result.WorktreePath, "should use existing worktree path")
}

func TestCreateSession_CreatesNewWorktreeWhenNoneExists(t *testing.T) {
	newWorktreePath := "/path/to/new/worktree"

	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	// Setup expectations
	gitRepo.EXPECT().GetOrCloneRepository(mock.Anything, mock.Anything).
		Return("/path/to/repo", &domain.RepoSource{Owner: "test", Repo: "repo"}, nil)
	gitRepo.EXPECT().GetWorktreeForBranch("/path/to/repo", "feature-branch").
		Return("", nil) // No existing worktree
	gitRepo.EXPECT().BuildWorktreePath(mock.Anything, "test/repo", mock.Anything).
		Return(newWorktreePath)
	gitRepo.EXPECT().CreateWorktree("/path/to/repo", newWorktreePath, "feature-branch").
		Return(nil)

	claudeDirResolver.EXPECT().Resolve("test/repo", mock.Anything).Return("/tmp/claude")

	tmuxClient.EXPECT().CreateSession(mock.Anything, newWorktreePath, mock.Anything, mock.Anything, mock.Anything).
		Return(&ports.TmuxSession{Name: "test-session"}, nil)

	sessionRepo.EXPECT().Add(mock.Anything, mock.Anything).Return(nil)

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	result, err := service.CreateSession(context.Background(), CreateSessionParams{
		SessionName:        "test-session",
		BranchNameOverride: "feature-branch",
		RepoSource:         "https://github.com/test/repo",
	})

	require.NoError(t, err)
	assert.Equal(t, newWorktreePath, result.WorktreePath, "should use newly created worktree path")
}

func TestCreateSession_ContinuesOnWorktreeLookupError(t *testing.T) {
	newWorktreePath := "/path/to/new/worktree"

	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	// Setup expectations - GetWorktreeForBranch returns error
	gitRepo.EXPECT().GetOrCloneRepository(mock.Anything, mock.Anything).
		Return("/path/to/repo", &domain.RepoSource{Owner: "test", Repo: "repo"}, nil)
	gitRepo.EXPECT().GetWorktreeForBranch("/path/to/repo", "feature-branch").
		Return("", errors.New("lookup failed"))
	gitRepo.EXPECT().BuildWorktreePath(mock.Anything, "test/repo", mock.Anything).
		Return(newWorktreePath)
	gitRepo.EXPECT().CreateWorktree("/path/to/repo", newWorktreePath, "feature-branch").
		Return(nil)

	claudeDirResolver.EXPECT().Resolve("test/repo", mock.Anything).Return("/tmp/claude")

	tmuxClient.EXPECT().CreateSession(mock.Anything, newWorktreePath, mock.Anything, mock.Anything, mock.Anything).
		Return(&ports.TmuxSession{Name: "test-session"}, nil)

	sessionRepo.EXPECT().Add(mock.Anything, mock.Anything).Return(nil)

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	result, err := service.CreateSession(context.Background(), CreateSessionParams{
		SessionName:        "test-session",
		BranchNameOverride: "feature-branch",
		RepoSource:         "https://github.com/test/repo",
	})

	require.NoError(t, err, "should continue even when worktree lookup fails")
	assert.Equal(t, newWorktreePath, result.WorktreePath)
}

func TestDeleteSession_HappyPath(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	session := &domain.Session{
		Name:         "test-session",
		WorktreePath: "/path/to/worktree",
		RepoPath:     "/path/to/repo",
	}

	sessionRepo.EXPECT().Get(mock.Anything, "test-session").Return(session, nil)
	tmuxClient.EXPECT().KillSession("test-session").Return(nil)
	sessionRepo.EXPECT().Delete(mock.Anything, "test-session").Return(nil)
	gitRepo.EXPECT().RemoveWorktree("/path/to/repo", "/path/to/worktree").Return(nil)

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.DeleteSession(context.Background(), "test-session", DeleteSessionOptions{
		KillTmux:       true,
		RemoveWorktree: true,
	})

	require.NoError(t, err)
}

func TestDeleteSession_WithShellSession(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	shellSession := &domain.Session{Name: "test-session-shell"}
	session := &domain.Session{
		Name:         "test-session",
		ShellSession: shellSession,
		WorktreePath: "/path/to/worktree",
		RepoPath:     "/path/to/repo",
	}

	sessionRepo.EXPECT().Get(mock.Anything, "test-session").Return(session, nil)
	tmuxClient.EXPECT().KillSession("test-session-shell").Return(nil)
	tmuxClient.EXPECT().KillSession("test-session").Return(nil)
	sessionRepo.EXPECT().Delete(mock.Anything, "test-session").Return(nil)
	gitRepo.EXPECT().RemoveWorktree("/path/to/repo", "/path/to/worktree").Return(nil)

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.DeleteSession(context.Background(), "test-session", DeleteSessionOptions{
		KillTmux:       true,
		RemoveWorktree: true,
	})

	require.NoError(t, err)
}

func TestDeleteSession_NoWorktree(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	session := &domain.Session{
		Name:         "test-session",
		WorktreePath: "", // No worktree
		RepoPath:     "",
	}

	sessionRepo.EXPECT().Get(mock.Anything, "test-session").Return(session, nil)
	sessionRepo.EXPECT().Delete(mock.Anything, "test-session").Return(nil)
	// RemoveWorktree should NOT be called since paths are empty

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.DeleteSession(context.Background(), "test-session", DeleteSessionOptions{
		KillTmux:       false,
		RemoveWorktree: true, // Requested but should be skipped
	})

	require.NoError(t, err)
}

func TestDeleteSession_GetSessionError(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	sessionRepo.EXPECT().Get(mock.Anything, "test-session").Return(nil, errors.New("not found"))

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.DeleteSession(context.Background(), "test-session", DeleteSessionOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get session")
}

func TestDeleteSession_TmuxKillFailureContinues(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	session := &domain.Session{
		Name:         "test-session",
		WorktreePath: "/path/to/worktree",
		RepoPath:     "/path/to/repo",
	}

	sessionRepo.EXPECT().Get(mock.Anything, "test-session").Return(session, nil)
	// Tmux kill fails but deletion should continue
	tmuxClient.EXPECT().KillSession("test-session").Return(errors.New("tmux error"))
	sessionRepo.EXPECT().Delete(mock.Anything, "test-session").Return(nil)
	gitRepo.EXPECT().RemoveWorktree("/path/to/repo", "/path/to/worktree").Return(nil)

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.DeleteSession(context.Background(), "test-session", DeleteSessionOptions{
		KillTmux:       true,
		RemoveWorktree: true,
	})

	require.NoError(t, err)
}

func TestDeleteSession_DatabaseDeleteError(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	session := &domain.Session{
		Name:         "test-session",
		WorktreePath: "/path/to/worktree",
		RepoPath:     "/path/to/repo",
	}

	sessionRepo.EXPECT().Get(mock.Anything, "test-session").Return(session, nil)
	sessionRepo.EXPECT().Delete(mock.Anything, "test-session").Return(errors.New("db error"))

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.DeleteSession(context.Background(), "test-session", DeleteSessionOptions{
		KillTmux:       false,
		RemoveWorktree: true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete session")
}

func TestRenameSession_HappyPath(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	tmuxClient.EXPECT().RenameSession("old-session", "new-session").Return(nil)
	sessionRepo.EXPECT().Rename(mock.Anything, "old-session", "new-session", "New Session").Return(nil)

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.RenameSession(context.Background(), "old-session", "new-session", "New Session")

	require.NoError(t, err)
}

func TestRenameSession_TmuxRenameError(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	tmuxClient.EXPECT().RenameSession("old-session", "new-session").Return(errors.New("tmux error"))

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.RenameSession(context.Background(), "old-session", "new-session", "New Session")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to rename tmux session")
}

func TestRenameSession_DatabaseRenameErrorWithRollback(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	// Tmux rename succeeds
	tmuxClient.EXPECT().RenameSession("old-session", "new-session").Return(nil)
	// Database rename fails
	sessionRepo.EXPECT().Rename(mock.Anything, "old-session", "new-session", "New Session").Return(errors.New("db error"))
	// Rollback tmux rename
	tmuxClient.EXPECT().RenameSession("new-session", "old-session").Return(nil)

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.RenameSession(context.Background(), "old-session", "new-session", "New Session")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to rename in database")
}

func TestRenameSession_DatabaseRenameErrorRollbackFails(t *testing.T) {
	gitRepo := portsmocks.NewMockGitRepository(t)
	tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
	sessionRepo := portsmocks.NewMockSessionRepository(t)
	claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
	processInspector := portsmocks.NewMockProcessInspector(t)

	// Tmux rename succeeds
	tmuxClient.EXPECT().RenameSession("old-session", "new-session").Return(nil)
	// Database rename fails
	sessionRepo.EXPECT().Rename(mock.Anything, "old-session", "new-session", "New Session").Return(errors.New("db error"))
	// Rollback fails too (but error is logged, not returned)
	tmuxClient.EXPECT().RenameSession("new-session", "old-session").Return(errors.New("rollback failed"))

	service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

	err := service.RenameSession(context.Background(), "old-session", "new-session", "New Session")

	// Should still return the database error, not the rollback error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to rename in database")
}

func TestUpdatePRInfo(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		prInfo      *domain.PRInfo
		repoErr     error
		wantErr     bool
		errContains string
	}{
		{
			name:        "happy path with PR info",
			sessionName: "test-session",
			prInfo: &domain.PRInfo{
				Number: 123,
				State:  "OPEN",
				URL:    "https://github.com/test/repo/pull/123",
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name:        "nil PR info does not panic",
			sessionName: "test-session",
			prInfo:      nil,
			repoErr:     nil,
			wantErr:     false,
		},
		{
			name:        "repository error propagates",
			sessionName: "test-session",
			prInfo: &domain.PRInfo{
				Number: 123,
				State:  "OPEN",
			},
			repoErr:     errors.New("db error"),
			wantErr:     true,
			errContains: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitRepo := portsmocks.NewMockGitRepository(t)
			tmuxClient := portsmocks.NewMockTmuxSessionLifecycle(t)
			sessionRepo := portsmocks.NewMockSessionRepository(t)
			claudeDirResolver := servicesmocks.NewMockClaudeDirResolver(t)
			processInspector := portsmocks.NewMockProcessInspector(t)

			sessionRepo.EXPECT().UpdatePRInfo(mock.Anything, tt.sessionName, tt.prInfo).Return(tt.repoErr)

			service := NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver, processInspector)

			err := service.UpdatePRInfo(context.Background(), tt.sessionName, tt.prInfo)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
