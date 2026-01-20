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

	// Auto-migrate Session table first
	if err := db.AutoMigrate(&Session{}); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("failed to migrate Session schema: %w", err)
		}
	}

	// Manually create session_flags table (AutoMigrate has issues with foreign keys in SQLite)
	migrator := db.Migrator()
	if !migrator.HasTable(&SessionFlag{}) {
		if err := db.Exec(`
			CREATE TABLE IF NOT EXISTS session_flags (
				session_name TEXT PRIMARY KEY,
				is_flagged INTEGER NOT NULL DEFAULT 0,
				flagged_at DATETIME,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY (session_name) REFERENCES sessions(name) ON UPDATE CASCADE ON DELETE CASCADE
			)
		`).Error; err != nil {
			return nil, fmt.Errorf("failed to create session_flags table: %w", err)
		}
	}

	// Manually create session_statuses table (AutoMigrate has issues with foreign keys in SQLite)
	if !migrator.HasTable(&SessionStatus{}) {
		if err := db.Exec(`
			CREATE TABLE IF NOT EXISTS session_statuses (
				session_name TEXT PRIMARY KEY,
				status TEXT NOT NULL,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY (session_name) REFERENCES sessions(name) ON UPDATE CASCADE ON DELETE CASCADE
			)
		`).Error; err != nil {
			return nil, fmt.Errorf("failed to create session_statuses table: %w", err)
		}
	}

	// Manually create session_comments table (AutoMigrate has issues with foreign keys in SQLite)
	if !migrator.HasTable(&SessionComment{}) {
		if err := db.Exec(`
			CREATE TABLE IF NOT EXISTS session_comments (
				session_name TEXT PRIMARY KEY,
				comment TEXT NOT NULL DEFAULT '',
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY (session_name) REFERENCES sessions(name) ON UPDATE CASCADE ON DELETE CASCADE
			)
		`).Error; err != nil {
			return nil, fmt.Errorf("failed to create session_comments table: %w", err)
		}
	}

	// Manually create session_archives table
	if !migrator.HasTable(&SessionArchive{}) {
		if err := db.Exec(`
			CREATE TABLE IF NOT EXISTS session_archives (
				session_name TEXT PRIMARY KEY,
				is_archived INTEGER NOT NULL DEFAULT 0,
				archived_at DATETIME,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY (session_name) REFERENCES sessions(name) ON UPDATE CASCADE ON DELETE CASCADE
			)
		`).Error; err != nil {
			return nil, fmt.Errorf("failed to create session_archives table: %w", err)
		}
	}

	// Manually create session_agent_cli_flags table
	if !migrator.HasTable(&SessionAgentCLIFlags{}) {
		if err := db.Exec(`
			CREATE TABLE IF NOT EXISTS session_agent_cli_flags (
				session_name TEXT PRIMARY KEY,
				allow_dangerously_skip_permissions INTEGER NOT NULL DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY (session_name) REFERENCES sessions(name) ON UPDATE CASCADE ON DELETE CASCADE
			)
		`).Error; err != nil {
			return nil, fmt.Errorf("failed to create session_agent_cli_flags table: %w", err)
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
func (s *Store) Load(ctx context.Context, showArchived bool) (*SessionState, error) {
	var state SessionState

	err := withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Load top-level sessions only (parent_name IS NULL), ordered by position
			// Use secondary sort by name for consistent ordering when positions are equal
			query := tx.Where("parent_name IS NULL")

			// Filter archived sessions unless showArchived is true
			if !showArchived {
				query = query.Where("name NOT IN (SELECT session_name FROM session_archives WHERE is_archived = 1)")
			}

			var sessions []Session
			if err := query.Order("position ASC, name ASC").Find(&sessions).Error; err != nil {
				return fmt.Errorf("failed to load sessions: %w", err)
			}

			// Load all flags at once
			var flags []SessionFlag
			if err := tx.Find(&flags).Error; err != nil {
				return fmt.Errorf("failed to load flags: %w", err)
			}

			// Load all comments at once
			var comments []SessionComment
			if err := tx.Find(&comments).Error; err != nil {
				return fmt.Errorf("failed to load comments: %w", err)
			}

			// Load all session statuses for efficient lookup
			// Note: table might not exist yet on first run with new schema
			var statuses []SessionStatus
			if err := tx.Find(&statuses).Error; err != nil {
				if !strings.Contains(err.Error(), "no such table") {
					return fmt.Errorf("failed to load session statuses: %w", err)
				}
				// Table doesn't exist yet, skip status loading
				logging.Logger.Debug("session_statuses table doesn't exist yet, skipping status loading")
			}

			// Load all archives at once
			var archives []SessionArchive
			if err := tx.Find(&archives).Error; err != nil {
				return fmt.Errorf("failed to load archives: %w", err)
			}

			// Load all agent CLI flags at once
			var agentCLIFlags []SessionAgentCLIFlags
			if err := tx.Find(&agentCLIFlags).Error; err != nil {
				return fmt.Errorf("failed to load agent CLI flags: %w", err)
			}

			// Build flag lookup map
			flagMap := make(map[string]bool)
			for _, flag := range flags {
				if flag.IsFlagged {
					flagMap[flag.SessionName] = true
				}
			}

			// Build status lookup map
			statusMap := make(map[string]*string)
			for _, s := range statuses {
				statusCopy := s.Status
				statusMap[s.SessionName] = &statusCopy
			}

			// Build comment lookup map
			commentMap := make(map[string]string)
			for _, comment := range comments {
				if comment.Comment != "" {
					commentMap[comment.SessionName] = comment.Comment
				}
			}

			// Build archive lookup map
			archiveMap := make(map[string]bool)
			for _, archive := range archives {
				if archive.IsArchived {
					archiveMap[archive.SessionName] = true
				}
			}

			// Build agent CLI flags lookup map
			agentCLIFlagsMap := make(map[string]bool)
			for _, flags := range agentCLIFlags {
				agentCLIFlagsMap[flags.SessionName] = flags.AllowDangerouslySkipPermissions
			}

			// Build ordered names list and convert to map
			state.Sessions = make(map[string]SessionInfo)
			state.OrderedSessionNames = make([]string, len(sessions))

			// Check if we need to normalize positions (detect duplicates or non-sequential)
			needsNormalization := false
			positionSet := make(map[int]bool)
			for i, sess := range sessions {
				if positionSet[sess.Position] {
					needsNormalization = true // duplicate position found
					break
				}
				positionSet[sess.Position] = true
				if sess.Position != i {
					needsNormalization = true // non-sequential position
					break
				}
			}

			// Normalize positions if needed
			if needsNormalization {
				logging.Logger.Debug("Normalizing session positions", "count", len(sessions))
				for i, sess := range sessions {
					if sess.Position != i {
						if err := tx.Model(&Session{}).Where("name = ?", sess.Name).Update("position", i).Error; err != nil {
							logging.Logger.Warn("Failed to normalize position", "session", sess.Name, "error", err)
						} else {
							logging.Logger.Debug("Normalized position", "session", sess.Name, "old_pos", sess.Position, "new_pos", i)
							sessions[i].Position = i // Update in-memory too
						}
					}
				}
			}

			for i, sess := range sessions {
				// Get flag state, status, comment, archive, and agent CLI flags for top-level session
				isFlagged := flagMap[sess.Name]
				comment := commentMap[sess.Name]
				isArchived := archiveMap[sess.Name]
				allowSkipPerms := agentCLIFlagsMap[sess.Name]

				// Load nested session if exists (parent_name = this session's name)
				var nestedSession Session
				err := tx.Where("parent_name = ?", sess.Name).First(&nestedSession).Error
				if err == nil {
					// Nested session found (nested sessions are never flagged, don't have status, no comment, not archived, and no agent CLI flags)
					nestedAllowSkipPerms := agentCLIFlagsMap[nestedSession.Name]
					nestedInfo := convertToSessionInfo(nestedSession, false, nil, "", false, nestedAllowSkipPerms)
					sessInfo := convertToSessionInfo(sess, isFlagged, statusMap, comment, isArchived, allowSkipPerms)
					sessInfo.ShellSession = &nestedInfo
					state.Sessions[sess.Name] = sessInfo
				} else if errors.Is(err, gorm.ErrRecordNotFound) {
					// No nested session
					state.Sessions[sess.Name] = convertToSessionInfo(sess, isFlagged, statusMap, comment, isArchived, allowSkipPerms)
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
			existingPositions := make(map[string]int)
			for _, sess := range existingSessions {
				existingNames[sess.Name] = true
				existingPositions[sess.Name] = sess.Position
			}

			// Build position map from OrderedSessionNames
			// This ensures positions match the intended order
			positionMap := make(map[string]int)
			for i, name := range state.OrderedSessionNames {
				positionMap[name] = i
			}

			// Save sessions
			for _, sessInfo := range state.Sessions {
				session := convertFromSessionInfo(sessInfo)

				// Assign position based on OrderedSessionNames
				if pos, exists := positionMap[session.Name]; exists {
					// Use position from OrderedSessionNames
					session.Position = pos
				} else if existingPos, exists := existingPositions[session.Name]; exists {
					// Fallback: preserve existing position if not in OrderedSessionNames
					session.Position = existingPos
				} else {
					// New session not in OrderedSessionNames: add at end
					session.Position = len(state.OrderedSessionNames)
				}

				if err := tx.Save(&session).Error; err != nil {
					return fmt.Errorf("failed to save session %s: %w", sessInfo.Name, err)
				}
				delete(existingNames, sessInfo.Name)

				// Save or delete agent CLI flags based on the field value
				if sessInfo.AllowDangerouslySkipPermissions {
					flags := SessionAgentCLIFlags{
						SessionName:                     sessInfo.Name,
						AllowDangerouslySkipPermissions: true,
					}
					if err := tx.Save(&flags).Error; err != nil {
						return fmt.Errorf("failed to save session agent CLI flags: %w", err)
					}
				} else {
					// Delete agent CLI flags record if false (cleanup)
					tx.Where("session_name = ?", sessInfo.Name).Delete(&SessionAgentCLIFlags{})
				}

				// Save nested session if exists (as regular Session with ParentName set)
				if sessInfo.ShellSession != nil {
					nested := convertFromSessionInfo(*sessInfo.ShellSession)
					nested.ParentName = &sessInfo.Name // Set parent relationship

					// Preserve position field from existing nested session
					var existingNested Session
					err := tx.Where("name = ?", nested.Name).First(&existingNested).Error
					if err == nil {
						nested.Position = existingNested.Position
					}

					if err := tx.Save(&nested).Error; err != nil {
						return fmt.Errorf("failed to save nested session for %s: %w", sessInfo.Name, err)
					}
					delete(existingNames, sessInfo.ShellSession.Name)

					// Save or delete agent CLI flags for nested session
					if sessInfo.ShellSession.AllowDangerouslySkipPermissions {
						nestedFlags := SessionAgentCLIFlags{
							SessionName:                     sessInfo.ShellSession.Name,
							AllowDangerouslySkipPermissions: true,
						}
						if err := tx.Save(&nestedFlags).Error; err != nil {
							return fmt.Errorf("failed to save nested session agent CLI flags: %w", err)
						}
					} else {
						// Delete agent CLI flags record if false (cleanup)
						tx.Where("session_name = ?", sessInfo.ShellSession.Name).Delete(&SessionAgentCLIFlags{})
					}
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

// UpdateSessionRepoSource updates the repo_source field for a session
func (s *Store) UpdateSessionRepoSource(ctx context.Context, name, repoSource string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updates := map[string]interface{}{
				"repo_source":  repoSource,
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

// UpdateExecutionID updates only the execution ID without changing last_updated timestamp
func (s *Store) UpdateExecutionID(ctx context.Context, name, executionID string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updates := map[string]interface{}{
				"execution_id": executionID,
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

// UpdateSessionStatus updates the status of a single session atomically
// Note: Only top-level sessions can have status, nested sessions cannot
func (s *Store) UpdateSessionStatus(ctx context.Context, name string, status *string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// First check if session exists and is not nested
			var session Session
			if err := tx.Where("name = ?", name).First(&session).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("session %s not found", name)
				}
				return err
			}

			// Reject status updates for nested sessions
			if session.ParentName != nil {
				return fmt.Errorf("cannot set status on nested session %s", name)
			}

			if status == nil || *status == "" {
				// Delete the status entry if status is nil or empty
				if err := tx.Where("session_name = ?", name).Delete(&SessionStatus{}).Error; err != nil {
					// Ignore "no such table" errors on first run
					if !strings.Contains(err.Error(), "no such table") {
						return fmt.Errorf("failed to delete session status: %w", err)
					}
				}
			} else {
				// Upsert the status
				sessionStatus := SessionStatus{
					SessionName: name,
					Status:      *status,
				}
				// Use Save which will insert or update
				if err := tx.Save(&sessionStatus).Error; err != nil {
					return fmt.Errorf("failed to save session status: %w", err)
				}
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

			// Debug: Log before swap
			logging.Logger.Debug("SwapPositions - before swap",
				"session1", name1, "pos1_before", session1.Position,
				"session2", name2, "pos2_before", session2.Position)

			// Swap positions
			session1.Position, session2.Position = session2.Position, session1.Position

			// Debug: Log after swap (in memory)
			logging.Logger.Debug("SwapPositions - after swap (in memory)",
				"session1", name1, "pos1_after", session1.Position,
				"session2", name2, "pos2_after", session2.Position)

			// Update both sessions
			if err := tx.Model(&Session{}).Where("name = ?", name1).Update("position", session1.Position).Error; err != nil {
				return fmt.Errorf("failed to update position for %s: %w", name1, err)
			}
			if err := tx.Model(&Session{}).Where("name = ?", name2).Update("position", session2.Position).Error; err != nil {
				return fmt.Errorf("failed to update position for %s: %w", name2, err)
			}

			// Debug: Verify the update in the database
			var check1, check2 Session
			if err := tx.Where("name = ?", name1).First(&check1).Error; err != nil {
				logging.Logger.Warn("Failed to verify update", "session", name1, "error", err)
			} else {
				logging.Logger.Debug("SwapPositions - verified in DB",
					"session1", name1, "pos1_in_db", check1.Position)
			}
			if err := tx.Where("name = ?", name2).First(&check2).Error; err != nil {
				logging.Logger.Warn("Failed to verify update", "session", name2, "error", err)
			} else {
				logging.Logger.Debug("SwapPositions - verified in DB",
					"session2", name2, "pos2_in_db", check2.Position)
			}

			return nil
		})
	}, 3)
}

// ToggleFlag toggles the flag state for a session
func (s *Store) ToggleFlag(ctx context.Context, sessionName string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var flag SessionFlag
			err := tx.Where("session_name = ?", sessionName).First(&flag).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				// No flag record exists, create as flagged
				now := time.Now().UTC()
				flag = SessionFlag{
					IsFlagged:   true,
					SessionName: sessionName,
					FlaggedAt:   &now,
				}
				return tx.Create(&flag).Error
			}
			if err != nil {
				return fmt.Errorf("failed to load flag: %w", err)
			}

			// Toggle existing flag
			flag.IsFlagged = !flag.IsFlagged
			if flag.IsFlagged {
				now := time.Now().UTC()
				flag.FlaggedAt = &now
			} else {
				flag.FlaggedAt = nil
			}

			return tx.Save(&flag).Error
		})
	}, 3)
}

// UpdateComment updates the comment for a session (empty string deletes the comment)
func (s *Store) UpdateComment(ctx context.Context, sessionName, comment string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if comment == "" {
				// Empty comment - delete the record if it exists
				result := tx.Where("session_name = ?", sessionName).Delete(&SessionComment{})
				// Ignore error if record doesn't exist
				if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
					return fmt.Errorf("failed to delete comment: %w", result.Error)
				}
				return nil
			}

			// Non-empty comment - upsert (create or update)
			var existingComment SessionComment
			err := tx.Where("session_name = ?", sessionName).First(&existingComment).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				// No comment record exists, create new one
				newComment := SessionComment{
					Comment:     comment,
					SessionName: sessionName,
				}
				return tx.Create(&newComment).Error
			}
			if err != nil {
				return fmt.Errorf("failed to load comment: %w", err)
			}

			// Update existing comment
			existingComment.Comment = comment
			return tx.Save(&existingComment).Error
		})
	}, 3)
}

// ToggleArchive toggles the archive state for a session
func (s *Store) ToggleArchive(ctx context.Context, sessionName string) error {
	return withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var archive SessionArchive
			err := tx.Where("session_name = ?", sessionName).First(&archive).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				// No archive record exists, create as archived
				now := time.Now().UTC()
				archive = SessionArchive{
					IsArchived:  true,
					SessionName: sessionName,
					ArchivedAt:  &now,
				}
				return tx.Create(&archive).Error
			}
			if err != nil {
				return fmt.Errorf("failed to load archive: %w", err)
			}

			// Toggle existing archive
			archive.IsArchived = !archive.IsArchived
			if archive.IsArchived {
				now := time.Now().UTC()
				archive.ArchivedAt = &now
			} else {
				archive.ArchivedAt = nil
			}

			return tx.Save(&archive).Error
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

				// Save agent CLI flags for nested session if enabled
				if sessInfo.ShellSession.AllowDangerouslySkipPermissions {
					nestedFlags := SessionAgentCLIFlags{
						SessionName:                     sessInfo.ShellSession.Name,
						AllowDangerouslySkipPermissions: true,
					}
					if err := tx.Create(&nestedFlags).Error; err != nil {
						return fmt.Errorf("failed to create nested session agent CLI flags: %w", err)
					}
				}
			}

			// Save agent CLI flags if enabled
			if sessInfo.AllowDangerouslySkipPermissions {
				flags := SessionAgentCLIFlags{
					SessionName:                     sessInfo.Name,
					AllowDangerouslySkipPermissions: true,
				}
				if err := tx.Create(&flags).Error; err != nil {
					return fmt.Errorf("failed to create session agent CLI flags: %w", err)
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
	var flag SessionFlag
	var status SessionStatus
	var comment SessionComment
	var archive SessionArchive
	var agentCLIFlags SessionAgentCLIFlags
	var nestedAgentCLIFlags SessionAgentCLIFlags
	statusMap := make(map[string]*string)

	err := withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Get the session
			if err := tx.Where("name = ?", name).First(&session).Error; err != nil {
				return err
			}

			// Try to get flag
			err := tx.Where("session_name = ?", name).First(&flag).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			// Try to get status
			err = tx.Where("session_name = ?", name).First(&status).Error
			if err == nil {
				statusCopy := status.Status
				statusMap[name] = &statusCopy
			} else if !errors.Is(err, gorm.ErrRecordNotFound) && !strings.Contains(err.Error(), "no such table") {
				return err
			}

			// Try to get comment
			err = tx.Where("session_name = ?", name).First(&comment).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			// Try to get archive
			err = tx.Where("session_name = ?", name).First(&archive).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			// Try to get agent CLI flags
			err = tx.Where("session_name = ?", name).First(&agentCLIFlags).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			// Try to get nested session
			err = tx.Where("parent_name = ?", name).First(&nestedSession).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			// If nested session exists, try to get its agent CLI flags
			if nestedSession.Name != "" {
				err = tx.Where("session_name = ?", nestedSession.Name).First(&nestedAgentCLIFlags).Error
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
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

	isFlagged := flag.IsFlagged
	commentText := comment.Comment
	isArchived := archive.IsArchived
	allowSkipPerms := agentCLIFlags.AllowDangerouslySkipPermissions
	info := convertToSessionInfo(session, isFlagged, statusMap, commentText, isArchived, allowSkipPerms)

	// Add nested session if found (nested sessions are never flagged, don't have status, no comment, and not archived)
	if nestedSession.Name != "" {
		nestedAllowSkipPerms := nestedAgentCLIFlags.AllowDangerouslySkipPermissions
		nestedInfo := convertToSessionInfo(nestedSession, false, nil, "", false, nestedAllowSkipPerms)
		info.ShellSession = &nestedInfo
	}

	return &info, nil
}

// ListSessions returns all top-level sessions with their nested sessions
func (s *Store) ListSessions(ctx context.Context, showArchived bool) ([]SessionInfo, error) {
	var sessions []Session
	var allSessions []Session
	var flags []SessionFlag
	var statuses []SessionStatus
	var comments []SessionComment
	var archives []SessionArchive
	var agentCLIFlags []SessionAgentCLIFlags

	err := withRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Get top-level sessions
			query := tx.Where("parent_name IS NULL")

			// Filter archived sessions unless showArchived is true
			if !showArchived {
				query = query.Where("name NOT IN (SELECT session_name FROM session_archives WHERE is_archived = 1)")
			}

			if err := query.Find(&sessions).Error; err != nil {
				return err
			}

			// Get all sessions (including nested) for efficient lookup
			if err := tx.Find(&allSessions).Error; err != nil {
				return err
			}

			// Get all flags
			if err := tx.Find(&flags).Error; err != nil {
				return err
			}

			// Get all statuses
			if err := tx.Find(&statuses).Error; err != nil {
				if !strings.Contains(err.Error(), "no such table") {
					return err
				}
			}

			// Get all comments
			if err := tx.Find(&comments).Error; err != nil {
				return err
			}

			// Get all archives
			if err := tx.Find(&archives).Error; err != nil {
				return err
			}

			// Get all agent CLI flags
			if err := tx.Find(&agentCLIFlags).Error; err != nil {
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

	// Build flag lookup map
	flagMap := make(map[string]bool)
	for _, flag := range flags {
		if flag.IsFlagged {
			flagMap[flag.SessionName] = true
		}
	}

	// Build status lookup map
	statusMap := make(map[string]*string)
	for _, s := range statuses {
		statusCopy := s.Status
		statusMap[s.SessionName] = &statusCopy
	}

	// Build comment lookup map
	commentMap := make(map[string]string)
	for _, comment := range comments {
		if comment.Comment != "" {
			commentMap[comment.SessionName] = comment.Comment
		}
	}

	// Build archive lookup map
	archiveMap := make(map[string]bool)
	for _, archive := range archives {
		if archive.IsArchived {
			archiveMap[archive.SessionName] = true
		}
	}

	// Build agent CLI flags lookup map
	agentCLIFlagsMap := make(map[string]bool)
	for _, flags := range agentCLIFlags {
		agentCLIFlagsMap[flags.SessionName] = flags.AllowDangerouslySkipPermissions
	}

	// Convert to SessionInfo with nested sessions
	result := make([]SessionInfo, len(sessions))
	for i, sess := range sessions {
		isFlagged := flagMap[sess.Name]
		comment := commentMap[sess.Name]
		isArchived := archiveMap[sess.Name]
		allowSkipPerms := agentCLIFlagsMap[sess.Name]
		info := convertToSessionInfo(sess, isFlagged, statusMap, comment, isArchived, allowSkipPerms)

		// Add nested session if exists (nested sessions are never flagged, don't have status, no comment, and not archived)
		if nested, ok := nestedMap[sess.Name]; ok {
			nestedAllowSkipPerms := agentCLIFlagsMap[nested.Name]
			nestedInfo := convertToSessionInfo(nested, false, nil, "", false, nestedAllowSkipPerms)
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
func convertToSessionInfo(s Session, isFlagged bool, statusMap map[string]*string, comment string, isArchived bool, allowSkipPerms bool) SessionInfo {
	var status *string
	if statusMap != nil {
		if s, ok := statusMap[s.Name]; ok {
			status = s
		}
	}

	return SessionInfo{
		AllowDangerouslySkipPermissions: allowSkipPerms,
		BranchName:                      s.BranchName,
		ClaudeDir:                       s.ClaudeDir,
		Comment:                         comment,
		DisplayName:                     s.DisplayName,
		ExecutionID:                     s.ExecutionID,
		GitStats:                        nil, // Not persisted
		IsArchived:                      isArchived,
		IsFlagged:                       isFlagged,
		LastUpdated:                     s.LastUpdated,
		Name:                            s.Name,
		RepoInfo:                        s.RepoInfo,
		RepoPath:                        s.RepoPath,
		RepoSource:                      s.RepoSource,
		ShellSession:                    nil,
		State:                           s.State,
		Status:                          status,
		WorktreePath:                    s.WorktreePath,
	}
}

func convertFromSessionInfo(info SessionInfo) Session {
	return Session{
		BranchName:   info.BranchName,
		ClaudeDir:    info.ClaudeDir,
		DisplayName:  info.DisplayName,
		ExecutionID:  info.ExecutionID,
		LastUpdated:  info.LastUpdated,
		Name:         info.Name,
		RepoInfo:     info.RepoInfo,
		RepoPath:     info.RepoPath,
		RepoSource:   info.RepoSource,
		State:        info.State,
		WorktreePath: info.WorktreePath,
	}
}
