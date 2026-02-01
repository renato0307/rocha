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
