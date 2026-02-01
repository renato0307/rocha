package ports

import (
	"context"

	"github.com/renato0307/rocha/internal/domain"
)

// SessionReader reads session data
type SessionReader interface {
	Get(ctx context.Context, name string) (*domain.Session, error)
	List(ctx context.Context, includeArchived bool) ([]domain.Session, error)
}

// SessionWriter creates, deletes, and reorders sessions
type SessionWriter interface {
	Add(ctx context.Context, session domain.Session) error
	Delete(ctx context.Context, name string) error
	LinkShellSession(ctx context.Context, parentName, shellSessionName string) error
	SwapPositions(ctx context.Context, name1, name2 string) error
}

// SessionStateUpdater updates session state
type SessionStateUpdater interface {
	UpdateClaudeDir(ctx context.Context, name, claudeDir string) error
	UpdateExecutionID(ctx context.Context, name, executionID string) error
	UpdateRepoSource(ctx context.Context, name, repoSource string) error
	UpdateSkipPermissions(ctx context.Context, name string, skip bool) error
	UpdateState(ctx context.Context, name string, state domain.SessionState, executionID string) error
}

// SessionMetadataUpdater updates session metadata
type SessionMetadataUpdater interface {
	Rename(ctx context.Context, oldName, newName, newDisplayName string) error
	ToggleArchive(ctx context.Context, name string) error
	ToggleFlag(ctx context.Context, name string) error
	UpdateComment(ctx context.Context, name, comment string) error
	UpdateDisplayName(ctx context.Context, name, displayName string) error
	UpdatePRInfo(ctx context.Context, name string, prInfo *domain.PRInfo) error
	UpdateStatus(ctx context.Context, name string, status *string) error
}

// SessionStateLoader loads full session state for UI
type SessionStateLoader interface {
	LoadState(ctx context.Context, includeArchived bool) (*domain.SessionCollection, error)
	SaveState(ctx context.Context, state *domain.SessionCollection) error
}

// SessionRepository is the composite interface
type SessionRepository interface {
	SessionReader
	SessionWriter
	SessionStateUpdater
	SessionMetadataUpdater
	SessionStateLoader
	Close() error
}
