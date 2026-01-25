package ports

import (
	"context"

	"rocha/domain"
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
	SwapPositions(ctx context.Context, name1, name2 string) error
}

// SessionStateUpdater updates session state
type SessionStateUpdater interface {
	UpdateClaudeDir(ctx context.Context, name, claudeDir string) error
	UpdateRepoSource(ctx context.Context, name, repoSource string) error
	UpdateSkipPermissions(ctx context.Context, name string, skip bool) error
	UpdateState(ctx context.Context, name string, state domain.SessionState, executionID string) error
}

// SessionMetadataUpdater updates session metadata
type SessionMetadataUpdater interface {
	ToggleArchive(ctx context.Context, name string) error
	ToggleFlag(ctx context.Context, name string) error
	UpdateComment(ctx context.Context, name, comment string) error
	UpdateStatus(ctx context.Context, name string, status *string) error
}

// SessionRepository is the composite interface
type SessionRepository interface {
	SessionReader
	SessionWriter
	SessionStateUpdater
	SessionMetadataUpdater
	Close() error
}
