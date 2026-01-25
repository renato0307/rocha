package storage

import (
	"context"

	"rocha/domain"
	"rocha/ports"
	existingstore "rocha/storage"
)

// SQLiteRepository implements ports.SessionRepository by wrapping the existing storage.Store
type SQLiteRepository struct {
	store *existingstore.Store
}

// Verify interface compliance at compile time
var _ ports.SessionRepository = (*SQLiteRepository)(nil)

// NewSQLiteRepository creates a new SQLiteRepository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	store, err := existingstore.NewStore(dbPath)
	if err != nil {
		return nil, err
	}
	return &SQLiteRepository{store: store}, nil
}

// NewSQLiteRepositoryForPath creates a new SQLiteRepository for a specific ROCHA_HOME path
// This is useful for operations that need to work with multiple databases (e.g., migrations)
func NewSQLiteRepositoryForPath(rochaHomePath string) (*SQLiteRepository, error) {
	dbPath := rochaHomePath + "/state.db"
	return NewSQLiteRepository(dbPath)
}

// Close closes the underlying store
func (r *SQLiteRepository) Close() error {
	return r.store.Close()
}

// Get implements SessionReader.Get
func (r *SQLiteRepository) Get(ctx context.Context, name string) (*domain.Session, error) {
	info, err := r.store.GetSession(ctx, name)
	if err != nil {
		return nil, err
	}
	session := sessionInfoToDomain(*info)
	return &session, nil
}

// List implements SessionReader.List
func (r *SQLiteRepository) List(ctx context.Context, includeArchived bool) ([]domain.Session, error) {
	infos, err := r.store.ListSessions(ctx, includeArchived)
	if err != nil {
		return nil, err
	}
	sessions := make([]domain.Session, len(infos))
	for i, info := range infos {
		sessions[i] = sessionInfoToDomain(info)
	}
	return sessions, nil
}

// Add implements SessionWriter.Add
func (r *SQLiteRepository) Add(ctx context.Context, session domain.Session) error {
	info := domainToSessionInfo(session)
	return r.store.AddSession(ctx, info)
}

// Delete implements SessionWriter.Delete
func (r *SQLiteRepository) Delete(ctx context.Context, name string) error {
	return r.store.DeleteSession(ctx, name)
}

// SwapPositions implements SessionWriter.SwapPositions
func (r *SQLiteRepository) SwapPositions(ctx context.Context, name1, name2 string) error {
	return r.store.SwapPositions(ctx, name1, name2)
}

// UpdateState implements SessionStateUpdater.UpdateState
func (r *SQLiteRepository) UpdateState(ctx context.Context, name string, state domain.SessionState, executionID string) error {
	return r.store.UpdateSession(ctx, name, string(state), executionID)
}

// UpdateClaudeDir implements SessionStateUpdater.UpdateClaudeDir
func (r *SQLiteRepository) UpdateClaudeDir(ctx context.Context, name, claudeDir string) error {
	return r.store.UpdateSessionClaudeDir(ctx, name, claudeDir)
}

// UpdateRepoSource implements SessionStateUpdater.UpdateRepoSource
func (r *SQLiteRepository) UpdateRepoSource(ctx context.Context, name, repoSource string) error {
	return r.store.UpdateSessionRepoSource(ctx, name, repoSource)
}

// UpdateSkipPermissions implements SessionStateUpdater.UpdateSkipPermissions
func (r *SQLiteRepository) UpdateSkipPermissions(ctx context.Context, name string, skip bool) error {
	return r.store.UpdateSessionSkipPermissions(ctx, name, skip)
}

// ToggleFlag implements SessionMetadataUpdater.ToggleFlag
func (r *SQLiteRepository) ToggleFlag(ctx context.Context, name string) error {
	return r.store.ToggleFlag(ctx, name)
}

// ToggleArchive implements SessionMetadataUpdater.ToggleArchive
func (r *SQLiteRepository) ToggleArchive(ctx context.Context, name string) error {
	return r.store.ToggleArchive(ctx, name)
}

// UpdateStatus implements SessionMetadataUpdater.UpdateStatus
func (r *SQLiteRepository) UpdateStatus(ctx context.Context, name string, status *string) error {
	return r.store.UpdateSessionStatus(ctx, name, status)
}

// UpdateComment implements SessionMetadataUpdater.UpdateComment
func (r *SQLiteRepository) UpdateComment(ctx context.Context, name, comment string) error {
	return r.store.UpdateComment(ctx, name, comment)
}

// LoadState implements SessionStateLoader.LoadState
func (r *SQLiteRepository) LoadState(ctx context.Context, includeArchived bool) (*domain.SessionCollection, error) {
	state, err := r.store.Load(ctx, includeArchived)
	if err != nil {
		return nil, err
	}
	return sessionStateToDomain(state), nil
}

// SaveState implements SessionStateLoader.SaveState
func (r *SQLiteRepository) SaveState(ctx context.Context, state *domain.SessionCollection) error {
	storageState := domainToSessionState(state)
	return r.store.Save(ctx, storageState)
}
