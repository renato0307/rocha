package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// SQLiteRepository implements ports.SessionRepository using GORM
type SQLiteRepository struct {
	db *gorm.DB
}

// Verify interface compliance at compile time
var _ ports.SessionRepository = (*SQLiteRepository)(nil)

// gormLogger wraps the rocha logger for GORM
type gormLogger struct {
	level logger.LogLevel
}

func (l *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &gormLogger{level: level}
}

func (l *gormLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Info {
		logging.Logger.Info(fmt.Sprintf(msg, data...))
	}
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Warn {
		logging.Logger.Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Error {
		logging.Logger.Error(fmt.Sprintf(msg, data...))
	}
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level < logger.Info {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logging.Logger.Error("gorm query error",
			"error", err,
			"duration", elapsed,
			"sql", sql,
			"rows", rows,
		)
	} else if elapsed > 200*time.Millisecond {
		logging.Logger.Warn("slow query",
			"duration", elapsed,
			"sql", sql,
			"rows", rows,
		)
	} else {
		logging.Logger.Debug("gorm query",
			"duration", elapsed,
			"sql", sql,
			"rows", rows,
		)
	}
}

func newGormLogger() logger.Interface {
	if os.Getenv("ROCHA_DEBUG") == "1" {
		return (&gormLogger{}).LogMode(logger.Info)
	}
	return (&gormLogger{}).LogMode(logger.Silent)
}

// NewSQLiteRepository creates a new SQLiteRepository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
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
		PrepareStmt: false,
		NowFunc:     func() time.Time { return time.Now().UTC() },
		Logger:      newGormLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for concurrent access
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA foreign_keys=ON")

	// Auto-migrate Session table
	if err := db.AutoMigrate(&SessionModel{}); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("failed to migrate Session schema: %w", err)
		}
	}

	// Manually create extension tables
	migrator := db.Migrator()

	if !migrator.HasTable(&SessionFlagModel{}) {
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

	if !migrator.HasTable(&SessionStatusModel{}) {
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

	if !migrator.HasTable(&SessionCommentModel{}) {
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

	if !migrator.HasTable(&SessionArchiveModel{}) {
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

	if !migrator.HasTable(&SessionAgentCLIFlagsModel{}) {
		if err := db.Exec(`
			CREATE TABLE IF NOT EXISTS session_agent_cli_flags (
				session_name TEXT PRIMARY KEY,
				allow_dangerously_skip_permissions INTEGER NOT NULL DEFAULT 0,
				debug_claude INTEGER NOT NULL DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY (session_name) REFERENCES sessions(name) ON UPDATE CASCADE ON DELETE CASCADE
			)
		`).Error; err != nil {
			return nil, fmt.Errorf("failed to create session_agent_cli_flags table: %w", err)
		}
	}

	// Migrate existing session_agent_cli_flags table to add debug_claude column
	if migrator.HasTable(&SessionAgentCLIFlagsModel{}) {
		if !migrator.HasColumn(&SessionAgentCLIFlagsModel{}, "debug_claude") {
			if err := migrator.AddColumn(&SessionAgentCLIFlagsModel{}, "debug_claude"); err != nil {
				return nil, fmt.Errorf("failed to migrate debug_claude column: %w", err)
			}
		}
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(0)

	return &SQLiteRepository{db: db}, nil
}

// NewSQLiteRepositoryForPath creates a new SQLiteRepository for a specific ROCHA_HOME path
func NewSQLiteRepositoryForPath(rochaHomePath string) (*SQLiteRepository, error) {
	dbPath := rochaHomePath + "/state.db"
	return NewSQLiteRepository(dbPath)
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Get implements SessionReader.Get
func (r *SQLiteRepository) Get(ctx context.Context, name string) (*domain.Session, error) {
	var session SessionModel
	var nestedSession SessionModel
	var flag SessionFlagModel
	var status SessionStatusModel
	var comment SessionCommentModel
	var archive SessionArchiveModel
	var agentCLIFlags SessionAgentCLIFlagsModel
	var nestedAgentCLIFlags SessionAgentCLIFlagsModel

	err := withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("name = ?", name).First(&session).Error; err != nil {
				return err
			}

			// Load related data (ignore not found errors)
			tx.Where("session_name = ?", name).First(&flag)
			tx.Where("session_name = ?", name).First(&status)
			tx.Where("session_name = ?", name).First(&comment)
			tx.Where("session_name = ?", name).First(&archive)
			tx.Where("session_name = ?", name).First(&agentCLIFlags)

			// Load nested session
			err := tx.Where("parent_name = ?", name).First(&nestedSession).Error
			if err == nil && nestedSession.Name != "" {
				tx.Where("session_name = ?", nestedSession.Name).First(&nestedAgentCLIFlags)
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

	var statusPtr *string
	if status.Status != "" {
		statusPtr = &status.Status
	}

	result := sessionModelToDomain(session, flag.IsFlagged, statusPtr, comment.Comment, archive.IsArchived, agentCLIFlags.AllowDangerouslySkipPermissions, agentCLIFlags.DebugClaude)

	// Add nested session if found
	if nestedSession.Name != "" {
		nested := sessionModelToDomain(nestedSession, false, nil, "", false, nestedAgentCLIFlags.AllowDangerouslySkipPermissions, nestedAgentCLIFlags.DebugClaude)
		result.ShellSession = &nested
	}

	return &result, nil
}

// List implements SessionReader.List
func (r *SQLiteRepository) List(ctx context.Context, includeArchived bool) ([]domain.Session, error) {
	var sessions []SessionModel
	var allSessions []SessionModel
	var flags []SessionFlagModel
	var statuses []SessionStatusModel
	var comments []SessionCommentModel
	var archives []SessionArchiveModel
	var agentCLIFlags []SessionAgentCLIFlagsModel

	err := withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			query := tx.Where("parent_name IS NULL")
			if !includeArchived {
				query = query.Where("name NOT IN (SELECT session_name FROM session_archives WHERE is_archived = 1)")
			}
			if err := query.Find(&sessions).Error; err != nil {
				return err
			}

			tx.Find(&allSessions)
			tx.Find(&flags)
			tx.Find(&statuses)
			tx.Find(&comments)
			tx.Find(&archives)
			tx.Find(&agentCLIFlags)

			return nil
		})
	}, 3)

	if err != nil {
		return nil, err
	}

	// Build lookup maps
	nestedMap := make(map[string]SessionModel)
	for _, s := range allSessions {
		if s.ParentName != nil {
			nestedMap[*s.ParentName] = s
		}
	}

	flagMap := make(map[string]bool)
	for _, f := range flags {
		flagMap[f.SessionName] = f.IsFlagged
	}

	statusMap := make(map[string]*string)
	for _, s := range statuses {
		statusCopy := s.Status
		statusMap[s.SessionName] = &statusCopy
	}

	commentMap := make(map[string]string)
	for _, c := range comments {
		commentMap[c.SessionName] = c.Comment
	}

	archiveMap := make(map[string]bool)
	for _, a := range archives {
		archiveMap[a.SessionName] = a.IsArchived
	}

	cliMap := make(map[string]bool)
	debugMap := make(map[string]bool)
	for _, f := range agentCLIFlags {
		cliMap[f.SessionName] = f.AllowDangerouslySkipPermissions
		debugMap[f.SessionName] = f.DebugClaude
	}

	// Convert to domain
	result := make([]domain.Session, len(sessions))
	for i, sess := range sessions {
		result[i] = sessionModelToDomain(sess, flagMap[sess.Name], statusMap[sess.Name], commentMap[sess.Name], archiveMap[sess.Name], cliMap[sess.Name], debugMap[sess.Name])

		if nested, ok := nestedMap[sess.Name]; ok {
			nestedDomain := sessionModelToDomain(nested, false, nil, "", false, cliMap[nested.Name], debugMap[nested.Name])
			result[i].ShellSession = &nestedDomain
		}
	}

	return result, nil
}

// Add implements SessionWriter.Add
func (r *SQLiteRepository) Add(ctx context.Context, session domain.Session) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Find minimum position to add at top
			var minPosition int
			tx.Model(&SessionModel{}).Select("MIN(position)").Scan(&minPosition)

			model := domainToSessionModel(session)
			model.Position = minPosition - 1

			if err := tx.Create(&model).Error; err != nil {
				return fmt.Errorf("failed to create session: %w", err)
			}

			// Add nested session if exists
			if session.ShellSession != nil {
				nested := domainToSessionModel(*session.ShellSession)
				nested.ParentName = &session.Name
				if err := tx.Create(&nested).Error; err != nil {
					return fmt.Errorf("failed to create nested session: %w", err)
				}

				if session.ShellSession.AllowDangerouslySkipPermissions || session.ShellSession.DebugClaude {
					if err := tx.Create(&SessionAgentCLIFlagsModel{
						SessionName:                     session.ShellSession.Name,
						AllowDangerouslySkipPermissions: session.ShellSession.AllowDangerouslySkipPermissions,
						DebugClaude:                     session.ShellSession.DebugClaude,
					}).Error; err != nil {
						return fmt.Errorf("failed to create nested session agent CLI flags: %w", err)
					}
				}
			}

			// Save agent CLI flags if enabled
			if session.AllowDangerouslySkipPermissions || session.DebugClaude {
				if err := tx.Create(&SessionAgentCLIFlagsModel{
					SessionName:                     session.Name,
					AllowDangerouslySkipPermissions: session.AllowDangerouslySkipPermissions,
					DebugClaude:                     session.DebugClaude,
				}).Error; err != nil {
					return fmt.Errorf("failed to create session agent CLI flags: %w", err)
				}
			}

			return nil
		})
	}, 3)
}

// Delete implements SessionWriter.Delete
func (r *SQLiteRepository) Delete(ctx context.Context, name string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Where("name = ?", name).Delete(&SessionModel{})
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

// LinkShellSession implements SessionWriter.LinkShellSession
func (r *SQLiteRepository) LinkShellSession(ctx context.Context, parentName, shellSessionName string) error {
	return withRetry(func() error {
		result := r.db.WithContext(ctx).Model(&SessionModel{}).
			Where("name = ?", shellSessionName).
			Update("parent_name", parentName)
		if result.Error != nil {
			return fmt.Errorf("failed to link shell session: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("shell session %s not found", shellSessionName)
		}
		return nil
	}, 3)
}

// SwapPositions implements SessionWriter.SwapPositions
func (r *SQLiteRepository) SwapPositions(ctx context.Context, name1, name2 string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var session1, session2 SessionModel
			if err := tx.Where("name = ?", name1).First(&session1).Error; err != nil {
				return fmt.Errorf("failed to find session %s: %w", name1, err)
			}
			if err := tx.Where("name = ?", name2).First(&session2).Error; err != nil {
				return fmt.Errorf("failed to find session %s: %w", name2, err)
			}

			session1.Position, session2.Position = session2.Position, session1.Position

			if err := tx.Model(&SessionModel{}).Where("name = ?", name1).Update("position", session1.Position).Error; err != nil {
				return fmt.Errorf("failed to update position for %s: %w", name1, err)
			}
			if err := tx.Model(&SessionModel{}).Where("name = ?", name2).Update("position", session2.Position).Error; err != nil {
				return fmt.Errorf("failed to update position for %s: %w", name2, err)
			}

			return nil
		})
	}, 3)
}

// UpdateState implements SessionStateUpdater.UpdateState
func (r *SQLiteRepository) UpdateState(ctx context.Context, name string, state domain.SessionState, executionID string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updates := map[string]any{
				"state":        string(state),
				"execution_id": executionID,
				"last_updated": time.Now().UTC(),
			}
			result := tx.Model(&SessionModel{}).Where("name = ?", name).Updates(updates)
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

// UpdateClaudeDir implements SessionStateUpdater.UpdateClaudeDir
func (r *SQLiteRepository) UpdateClaudeDir(ctx context.Context, name, claudeDir string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updates := map[string]any{
				"claude_dir":   claudeDir,
				"last_updated": time.Now().UTC(),
			}
			result := tx.Model(&SessionModel{}).Where("name = ?", name).Updates(updates)
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

// UpdateRepoSource implements SessionStateUpdater.UpdateRepoSource
func (r *SQLiteRepository) UpdateRepoSource(ctx context.Context, name, repoSource string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updates := map[string]any{
				"repo_source":  repoSource,
				"last_updated": time.Now().UTC(),
			}
			result := tx.Model(&SessionModel{}).Where("name = ?", name).Updates(updates)
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

// UpdateSkipPermissions implements SessionStateUpdater.UpdateSkipPermissions
func (r *SQLiteRepository) UpdateSkipPermissions(ctx context.Context, name string, skip bool) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Update timestamp
			result := tx.Model(&SessionModel{}).Where("name = ?", name).Update("last_updated", time.Now().UTC())
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("session %s not found", name)
			}

			var flags SessionAgentCLIFlagsModel
			err := tx.Where("session_name = ?", name).First(&flags).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if skip {
					return tx.Create(&SessionAgentCLIFlagsModel{
						SessionName:                     name,
						AllowDangerouslySkipPermissions: true,
					}).Error
				}
				return nil
			}
			if err != nil {
				return err
			}

			flags.AllowDangerouslySkipPermissions = skip
			if !flags.AllowDangerouslySkipPermissions && !flags.DebugClaude {
				tx.Where("session_name = ?", name).Delete(&SessionAgentCLIFlagsModel{})
				return nil
			}
			return tx.Save(&flags).Error
		})
	}, 3)
}

// UpdateDebugClaude implements SessionStateUpdater.UpdateDebugClaude
func (r *SQLiteRepository) UpdateDebugClaude(ctx context.Context, name string, debug bool) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Update timestamp
			result := tx.Model(&SessionModel{}).Where("name = ?", name).Update("last_updated", time.Now().UTC())
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("session %s not found", name)
			}

			var flags SessionAgentCLIFlagsModel
			err := tx.Where("session_name = ?", name).First(&flags).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if debug {
					return tx.Create(&SessionAgentCLIFlagsModel{
						SessionName: name,
						DebugClaude: true,
					}).Error
				}
				return nil
			}
			if err != nil {
				return err
			}

			flags.DebugClaude = debug
			if !flags.AllowDangerouslySkipPermissions && !flags.DebugClaude {
				tx.Where("session_name = ?", name).Delete(&SessionAgentCLIFlagsModel{})
				return nil
			}
			return tx.Save(&flags).Error
		})
	}, 3)
}

// ToggleFlag implements SessionMetadataUpdater.ToggleFlag
func (r *SQLiteRepository) ToggleFlag(ctx context.Context, name string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var flag SessionFlagModel
			err := tx.Where("session_name = ?", name).First(&flag).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				now := time.Now().UTC()
				return tx.Create(&SessionFlagModel{
					IsFlagged:   true,
					SessionName: name,
					FlaggedAt:   &now,
				}).Error
			}
			if err != nil {
				return fmt.Errorf("failed to load flag: %w", err)
			}

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

// Rename implements SessionMetadataUpdater.Rename
func (r *SQLiteRepository) Rename(ctx context.Context, oldName, newName, newDisplayName string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Update session name and display name, preserving position
			result := tx.Model(&SessionModel{}).
				Where("name = ?", oldName).
				Updates(map[string]any{
					"name":         newName,
					"display_name": newDisplayName,
					"last_updated": time.Now().UTC(),
				})
			if result.Error != nil {
				return fmt.Errorf("failed to rename session: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("session %s not found", oldName)
			}

			// Update parent_name references for shell sessions
			tx.Model(&SessionModel{}).
				Where("parent_name = ?", oldName).
				Update("parent_name", newName)

			// Update related tables
			tx.Model(&SessionAgentCLIFlagsModel{}).
				Where("session_name = ?", oldName).
				Update("session_name", newName)

			tx.Model(&SessionArchiveModel{}).
				Where("session_name = ?", oldName).
				Update("session_name", newName)

			return nil
		})
	}, 3)
}

// ToggleArchive implements SessionMetadataUpdater.ToggleArchive
func (r *SQLiteRepository) ToggleArchive(ctx context.Context, name string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var archive SessionArchiveModel
			err := tx.Where("session_name = ?", name).First(&archive).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				now := time.Now().UTC()
				return tx.Create(&SessionArchiveModel{
					IsArchived:  true,
					SessionName: name,
					ArchivedAt:  &now,
				}).Error
			}
			if err != nil {
				return fmt.Errorf("failed to load archive: %w", err)
			}

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

// UpdateStatus implements SessionMetadataUpdater.UpdateStatus
func (r *SQLiteRepository) UpdateStatus(ctx context.Context, name string, status *string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Check session exists and is not nested
			var session SessionModel
			if err := tx.Where("name = ?", name).First(&session).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("session %s not found", name)
				}
				return err
			}
			if session.ParentName != nil {
				return fmt.Errorf("cannot set status on nested session %s", name)
			}

			if status == nil || *status == "" {
				tx.Where("session_name = ?", name).Delete(&SessionStatusModel{})
				return nil
			}

			return tx.Save(&SessionStatusModel{
				SessionName: name,
				Status:      *status,
			}).Error
		})
	}, 3)
}

// UpdateDisplayName implements SessionMetadataUpdater.UpdateDisplayName
func (r *SQLiteRepository) UpdateDisplayName(ctx context.Context, name, displayName string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Model(&SessionModel{}).
				Where("name = ?", name).
				Updates(map[string]any{
					"display_name": displayName,
					"last_updated": time.Now().UTC(),
				})
			if result.Error != nil {
				return fmt.Errorf("failed to update display name: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("session %s not found", name)
			}
			return nil
		})
	}, 3)
}

// UpdateComment implements SessionMetadataUpdater.UpdateComment
func (r *SQLiteRepository) UpdateComment(ctx context.Context, name, comment string) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if comment == "" {
				tx.Where("session_name = ?", name).Delete(&SessionCommentModel{})
				return nil
			}

			var existing SessionCommentModel
			err := tx.Where("session_name = ?", name).First(&existing).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tx.Create(&SessionCommentModel{
					Comment:     comment,
					SessionName: name,
				}).Error
			}
			if err != nil {
				return fmt.Errorf("failed to load comment: %w", err)
			}

			existing.Comment = comment
			return tx.Save(&existing).Error
		})
	}, 3)
}

// saveAgentCLIFlags saves or deletes agent CLI flags for a session
func saveAgentCLIFlags(tx *gorm.DB, sessionName string, allowSkip, debug bool) error {
	if allowSkip || debug {
		return tx.Save(&SessionAgentCLIFlagsModel{
			SessionName:                     sessionName,
			AllowDangerouslySkipPermissions: allowSkip,
			DebugClaude:                     debug,
		}).Error
	}
	tx.Where("session_name = ?", sessionName).Delete(&SessionAgentCLIFlagsModel{})
	return nil
}

// LoadState implements SessionStateLoader.LoadState
func (r *SQLiteRepository) LoadState(ctx context.Context, includeArchived bool) (*domain.SessionCollection, error) {
	var sessions []SessionModel
	var flags []SessionFlagModel
	var comments []SessionCommentModel
	var statuses []SessionStatusModel
	var archives []SessionArchiveModel
	var agentCLIFlags []SessionAgentCLIFlagsModel

	err := withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			query := tx.Where("parent_name IS NULL")
			if !includeArchived {
				query = query.Where("name NOT IN (SELECT session_name FROM session_archives WHERE is_archived = 1)")
			}
			if err := query.Order("position ASC, name ASC").Find(&sessions).Error; err != nil {
				return fmt.Errorf("failed to load sessions: %w", err)
			}

			tx.Find(&flags)
			tx.Find(&comments)
			tx.Find(&statuses)
			tx.Find(&archives)
			tx.Find(&agentCLIFlags)

			// Normalize positions if needed
			needsNormalization := false
			positionSet := make(map[int]bool)
			for i, sess := range sessions {
				if positionSet[sess.Position] || sess.Position != i {
					needsNormalization = true
					break
				}
				positionSet[sess.Position] = true
			}

			if needsNormalization {
				for i, sess := range sessions {
					if sess.Position != i {
						tx.Model(&SessionModel{}).Where("name = ?", sess.Name).Update("position", i)
						sessions[i].Position = i
					}
				}
			}

			return nil
		})
	}, 3)

	if err != nil {
		return nil, err
	}

	// Build lookup maps
	flagMap := make(map[string]bool)
	for _, f := range flags {
		flagMap[f.SessionName] = f.IsFlagged
	}

	statusMap := make(map[string]*string)
	for _, s := range statuses {
		statusCopy := s.Status
		statusMap[s.SessionName] = &statusCopy
	}

	commentMap := make(map[string]string)
	for _, c := range comments {
		commentMap[c.SessionName] = c.Comment
	}

	archiveMap := make(map[string]bool)
	for _, a := range archives {
		archiveMap[a.SessionName] = a.IsArchived
	}

	cliMap := make(map[string]bool)
	debugMap := make(map[string]bool)
	for _, f := range agentCLIFlags {
		cliMap[f.SessionName] = f.AllowDangerouslySkipPermissions
		debugMap[f.SessionName] = f.DebugClaude
	}

	// Build result
	collection := &domain.SessionCollection{
		OrderedNames: make([]string, len(sessions)),
		Sessions:     make(map[string]domain.Session),
	}

	for i, sess := range sessions {
		collection.OrderedNames[i] = sess.Name

		domainSess := sessionModelToDomain(sess, flagMap[sess.Name], statusMap[sess.Name], commentMap[sess.Name], archiveMap[sess.Name], cliMap[sess.Name], debugMap[sess.Name])

		// Load nested session
		var nestedSession SessionModel
		if err := r.db.Where("parent_name = ?", sess.Name).First(&nestedSession).Error; err == nil {
			nested := sessionModelToDomain(nestedSession, false, nil, "", false, cliMap[nestedSession.Name], debugMap[nestedSession.Name])
			domainSess.ShellSession = &nested
		}

		collection.Sessions[sess.Name] = domainSess
	}

	return collection, nil
}

// SaveState implements SessionStateLoader.SaveState
func (r *SQLiteRepository) SaveState(ctx context.Context, state *domain.SessionCollection) error {
	return withRetry(func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Get existing sessions
			var existingSessions []SessionModel
			if err := tx.Find(&existingSessions).Error; err != nil {
				return fmt.Errorf("failed to load existing sessions: %w", err)
			}

			existingNames := make(map[string]bool)
			existingPositions := make(map[string]int)
			for _, sess := range existingSessions {
				existingNames[sess.Name] = true
				existingPositions[sess.Name] = sess.Position
			}

			// Build position map
			positionMap := make(map[string]int)
			for i, name := range state.OrderedNames {
				positionMap[name] = i
			}

			// Save sessions
			for _, session := range state.Sessions {
				model := domainToSessionModel(session)

				if pos, exists := positionMap[model.Name]; exists {
					model.Position = pos
				} else if existingPos, exists := existingPositions[model.Name]; exists {
					model.Position = existingPos
				} else {
					model.Position = len(state.OrderedNames)
				}

				if err := tx.Save(&model).Error; err != nil {
					return fmt.Errorf("failed to save session %s: %w", session.Name, err)
				}
				delete(existingNames, session.Name)

				// Handle agent CLI flags
				if err := saveAgentCLIFlags(tx, session.Name, session.AllowDangerouslySkipPermissions, session.DebugClaude); err != nil {
					return fmt.Errorf("failed to save agent CLI flags for %s: %w", session.Name, err)
				}

				// Handle nested session
				if session.ShellSession != nil {
					nested := domainToSessionModel(*session.ShellSession)
					nested.ParentName = &session.Name

					var existingNested SessionModel
					if err := tx.Where("name = ?", nested.Name).First(&existingNested).Error; err == nil {
						nested.Position = existingNested.Position
					}

					if err := tx.Save(&nested).Error; err != nil {
						return fmt.Errorf("failed to save nested session for %s: %w", session.Name, err)
					}
					delete(existingNames, session.ShellSession.Name)

					if err := saveAgentCLIFlags(tx, session.ShellSession.Name, session.ShellSession.AllowDangerouslySkipPermissions, session.ShellSession.DebugClaude); err != nil {
						return fmt.Errorf("failed to save agent CLI flags for nested session %s: %w", session.ShellSession.Name, err)
					}
				}
			}

			// Delete removed sessions
			for name := range existingNames {
				if err := tx.Where("name = ?", name).Delete(&SessionModel{}).Error; err != nil {
					return fmt.Errorf("failed to delete session %s: %w", name, err)
				}
			}

			return nil
		})
	}, 3)
}

// withRetry retries operations on SQLITE_BUSY with exponential backoff
func withRetry(fn func() error, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && (sqliteErr.Code == sqlite3.ErrBusy || sqliteErr.Code == sqlite3.ErrLocked) {
			time.Sleep(time.Millisecond * time.Duration(50*(i+1)))
			continue
		}

		return err
	}
	return fmt.Errorf("operation failed after %d retries", maxRetries)
}
