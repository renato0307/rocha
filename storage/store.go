package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rocha/logging"

	"github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// gormLogger wraps the rocha logger for GORM
type gormLogger struct {
	level logger.LogLevel
}

// LogMode sets the log level
func (l *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &gormLogger{level: level}
}

// Info logs info messages
func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info {
		logging.Logger.Info(fmt.Sprintf(msg, data...))
	}
}

// Warn logs warn messages
func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn {
		logging.Logger.Warn(fmt.Sprintf(msg, data...))
	}
}

// Error logs error messages
func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error {
		logging.Logger.Error(fmt.Sprintf(msg, data...))
	}
}

// Trace logs SQL queries - only in debug mode
func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	// Only log traces in Info level (debug mode)
	if l.level < logger.Info {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// Log errors (except ErrRecordNotFound which is expected)
		logging.Logger.Error("gorm query error",
			"error", err,
			"duration", elapsed,
			"sql", sql,
			"rows", rows,
		)
	} else if elapsed > 200*time.Millisecond {
		// Log slow queries
		logging.Logger.Warn("slow query",
			"duration", elapsed,
			"sql", sql,
			"rows", rows,
		)
	} else {
		// Log all queries in debug mode
		logging.Logger.Debug("gorm query",
			"duration", elapsed,
			"sql", sql,
			"rows", rows,
		)
	}
}

// newGormLogger creates a GORM logger that respects rocha's debug settings
func newGormLogger() logger.Interface {
	// Check if debug mode is enabled via environment variable
	// (set by cmd/root.go when --debug flag is used)
	if os.Getenv("ROCHA_DEBUG") == "1" {
		// Debug mode: log all queries to the debug file
		return (&gormLogger{}).LogMode(logger.Info)
	}

	// Normal mode: silent (no logs)
	return (&gormLogger{}).LogMode(logger.Silent)
}

// Store provides thread-safe ACID access to session state
type Store struct {
	db *gorm.DB
}

// NewStore creates a new storage instance with WAL mode enabled
func NewStore(dbPath string) (*Store, error) {
	// Expand home directory if present
	if len(dbPath) > 0 && dbPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, dbPath[1:])
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database with WAL mode
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		PrepareStmt: false, // Disable to avoid transaction conflicts
		NowFunc:     func() time.Time { return time.Now().UTC() },
		Logger:      newGormLogger(), // Use custom logger that respects --debug flag
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for concurrent access
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000") // 5 second timeout
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA foreign_keys=ON")

	// Auto-migrate schema first
	if err := db.AutoMigrate(
		&Session{},
	); err != nil {
		// Ignore "index already exists" errors (GORM sometimes reports this incorrectly)
		if !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("failed to migrate schema: %w", err)
		}
	}

	// Configure connection pool after migration
	// SQLite with WAL mode can handle multiple readers + 1 writer
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)  // Allow multiple readers
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(0)

	return &Store{db: db}, nil
}

// Load reads all sessions with ACID guarantees
func (s *Store) Load(ctx context.Context) (*SessionState, error) {
	var state SessionState

	err := withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Load top-level sessions only (parent_name IS NULL), ordered by position
			var sessions []Session
			if err := tx.Where("parent_name IS NULL").Order("position").Find(&sessions).Error; err != nil {
				return fmt.Errorf("failed to load sessions: %w", err)
			}

			// Build ordered names list and convert to map
			state.Sessions = make(map[string]SessionInfo)
			state.OrderedSessionNames = make([]string, len(sessions))

			for i, sess := range sessions {
				// Load nested session if exists (parent_name = this session's name)
				var nestedSession Session
				err := tx.Where("parent_name = ?", sess.Name).First(&nestedSession).Error
				if err == nil {
					// Nested session found
					nestedInfo := convertToSessionInfo(nestedSession)
					sessInfo := convertToSessionInfo(sess)
					sessInfo.ShellSession = &nestedInfo
					state.Sessions[sess.Name] = sessInfo
				} else if errors.Is(err, gorm.ErrRecordNotFound) {
					// No nested session
					state.Sessions[sess.Name] = convertToSessionInfo(sess)
				} else {
					return fmt.Errorf("failed to load nested session for %s: %w", sess.Name, err)
				}

				state.OrderedSessionNames[i] = sess.Name
			}

			return nil
		})
	}, 3)

	return &state, err
}

// Save writes session state atomically with ACID guarantees
func (s *Store) Save(ctx context.Context, state *SessionState) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Get existing session names to detect deletions
			var existingSessions []Session
			if err := tx.Find(&existingSessions).Error; err != nil {
				return fmt.Errorf("failed to load existing sessions: %w", err)
			}

			existingNames := make(map[string]bool)
			for _, sess := range existingSessions {
				existingNames[sess.Name] = true
			}

			// Save sessions
			for _, sessInfo := range state.Sessions {
				session := convertFromSessionInfo(sessInfo)
				if err := tx.Save(&session).Error; err != nil {
					return fmt.Errorf("failed to save session %s: %w", sessInfo.Name, err)
				}
				delete(existingNames, sessInfo.Name)

				// Save nested session if exists (as regular Session with ParentName set)
				if sessInfo.ShellSession != nil {
					nested := convertFromSessionInfo(*sessInfo.ShellSession)
					nested.ParentName = &sessInfo.Name // Set parent relationship
					if err := tx.Save(&nested).Error; err != nil {
						return fmt.Errorf("failed to save nested session for %s: %w", sessInfo.Name, err)
					}
					delete(existingNames, sessInfo.ShellSession.Name)
				}
			}

			// Delete sessions that are no longer in state
			for name := range existingNames {
				if err := tx.Where("name = ?", name).Delete(&Session{}).Error; err != nil {
					return fmt.Errorf("failed to delete session %s: %w", name, err)
				}
			}

			return nil
		})
	}, 3)
}

// UpdateSession updates a single session atomically
// This REPLACES the old event queue pattern - hooks call this directly!
func (s *Store) UpdateSession(ctx context.Context, name, state, executionID string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updates := map[string]interface{}{
				"state":        state,
				"execution_id": executionID,
				"last_updated": time.Now().UTC(),
			}
			result := tx.Model(&Session{}).Where("name = ?", name).Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("session %s not found", name)
			}
			return nil
		})
	}, 3)
}

// SwapPositions swaps the position of two sessions
func (s *Store) SwapPositions(ctx context.Context, name1, name2 string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Get both sessions
			var session1, session2 Session
			if err := tx.Where("name = ?", name1).First(&session1).Error; err != nil {
				return fmt.Errorf("failed to find session %s: %w", name1, err)
			}
			if err := tx.Where("name = ?", name2).First(&session2).Error; err != nil {
				return fmt.Errorf("failed to find session %s: %w", name2, err)
			}

			// Swap positions
			session1.Position, session2.Position = session2.Position, session1.Position

			// Update both sessions
			if err := tx.Model(&Session{}).Where("name = ?", name1).Update("position", session1.Position).Error; err != nil {
				return fmt.Errorf("failed to update position for %s: %w", name1, err)
			}
			if err := tx.Model(&Session{}).Where("name = ?", name2).Update("position", session2.Position).Error; err != nil {
				return fmt.Errorf("failed to update position for %s: %w", name2, err)
			}

			return nil
		})
	}, 3)
}

// AddSession adds a new session to the database at the top (position = min - 1)
func (s *Store) AddSession(ctx context.Context, sessInfo SessionInfo) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Find minimum position to add at top
			var minPosition int
			if err := tx.Model(&Session{}).Select("MIN(position)").Scan(&minPosition).Error; err != nil {
				// If no sessions exist, start at 0
				minPosition = 1
			}

			session := convertFromSessionInfo(sessInfo)
			session.Position = minPosition - 1 // Add at top
			if err := tx.Create(&session).Error; err != nil {
				return fmt.Errorf("failed to create session: %w", err)
			}

			// Add nested session if exists (as regular Session with ParentName set)
			if sessInfo.ShellSession != nil {
				nested := convertFromSessionInfo(*sessInfo.ShellSession)
				nested.ParentName = &sessInfo.Name // Set parent relationship
				if err := tx.Create(&nested).Error; err != nil {
					return fmt.Errorf("failed to create nested session: %w", err)
				}
			}

			return nil
		})
	}, 3)
}

// DeleteSession removes a session from the database
func (s *Store) DeleteSession(ctx context.Context, name string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Where("name = ?", name).Delete(&Session{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("session %s not found", name)
			}
			return nil
		})
	}, 3)
}

// GetSession retrieves a single session by name
func (s *Store) GetSession(ctx context.Context, name string) (*SessionInfo, error) {
	var session Session
	var nestedSession Session

	err := withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Get the session
			if err := tx.Where("name = ?", name).First(&session).Error; err != nil {
				return err
			}

			// Try to get nested session
			err := tx.Where("parent_name = ?", name).First(&nestedSession).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			return nil
		})
	}, 3)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("session %s not found", name)
		}
		return nil, err
	}

	info := convertToSessionInfo(session)

	// Add nested session if found
	if nestedSession.Name != "" {
		nestedInfo := convertToSessionInfo(nestedSession)
		info.ShellSession = &nestedInfo
	}

	return &info, nil
}

// ListSessions returns all top-level sessions with their nested sessions
func (s *Store) ListSessions(ctx context.Context) ([]SessionInfo, error) {
	var sessions []Session
	var allSessions []Session

	err := withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Get top-level sessions
			if err := tx.Where("parent_name IS NULL").Find(&sessions).Error; err != nil {
				return err
			}

			// Get all sessions (including nested) for efficient lookup
			if err := tx.Find(&allSessions).Error; err != nil {
				return err
			}

			return nil
		})
	}, 3)

	if err != nil {
		return nil, err
	}

	// Build map of nested sessions by parent name
	nestedMap := make(map[string]Session)
	for _, s := range allSessions {
		if s.ParentName != nil {
			nestedMap[*s.ParentName] = s
		}
	}

	// Convert to SessionInfo with nested sessions
	result := make([]SessionInfo, len(sessions))
	for i, sess := range sessions {
		info := convertToSessionInfo(sess)

		// Add nested session if exists
		if nested, ok := nestedMap[sess.Name]; ok {
			nestedInfo := convertToSessionInfo(nested)
			info.ShellSession = &nestedInfo
		}

		result[i] = info
	}

	return result, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// withRetry retries operations on SQLITE_BUSY with exponential backoff
func withRetry(fn func() error, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		// Check if it's a busy error
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && (sqliteErr.Code == sqlite3.ErrBusy || sqliteErr.Code == sqlite3.ErrLocked) {
			time.Sleep(time.Millisecond * time.Duration(50*(i+1)))
			continue
		}

		return err
	}
	return fmt.Errorf("operation failed after %d retries", maxRetries)
}

// Helper conversion functions
func convertToSessionInfo(s Session) SessionInfo {
	return SessionInfo{
		Name:         s.Name,
		DisplayName:  s.DisplayName,
		State:        s.State,
		ExecutionID:  s.ExecutionID,
		LastUpdated:  s.LastUpdated,
		RepoPath:     s.RepoPath,
		RepoInfo:     s.RepoInfo,
		BranchName:   s.BranchName,
		WorktreePath: s.WorktreePath,
		GitStats:     nil, // Not persisted
	}
}

func convertFromSessionInfo(info SessionInfo) Session {
	return Session{
		Name:         info.Name,
		DisplayName:  info.DisplayName,
		State:        info.State,
		ExecutionID:  info.ExecutionID,
		LastUpdated:  info.LastUpdated,
		RepoPath:     info.RepoPath,
		RepoInfo:     info.RepoInfo,
		BranchName:   info.BranchName,
		WorktreePath: info.WorktreePath,
	}
}
