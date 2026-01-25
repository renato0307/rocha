package cmd

import (
	"context"
	"os"
	"path/filepath"

	adaptereditor "rocha/internal/adapters/editor"
	adaptergit "rocha/internal/adapters/git"
	adaptersound "rocha/internal/adapters/sound"
	adapterstorage "rocha/internal/adapters/storage"
	"rocha/internal/application"
	"rocha/internal/config"
	"rocha/internal/logging"
	"rocha/internal/ports"
)

// Container holds all dependencies for the application
type Container struct {
	// Adapters
	EditorOpener      ports.EditorOpener
	GitRepository     ports.GitRepository
	SessionRepository ports.SessionRepository
	SoundPlayer       ports.SoundPlayer
	TmuxClient        ports.TmuxClient

	// Services
	GitService          *application.GitService
	MigrationService    *application.MigrationService
	NotificationService *application.NotificationService
	SessionService      *application.SessionService
	SettingsService     *application.SettingsService
	ShellService        *application.ShellService
}

// NewContainer creates a new Container with all dependencies wired
func NewContainer(tmuxClient ports.TmuxClient) (*Container, error) {
	// Create adapters
	sessionRepo, err := adapterstorage.NewSQLiteRepository(config.GetDBPath())
	if err != nil {
		return nil, err
	}

	editorOpener := adaptereditor.NewOpener()
	gitRepo := adaptergit.NewCLIRepository()
	soundPlayer := adaptersound.NewPlayer()

	// Create ClaudeDir resolver
	claudeDirResolver := NewClaudeDirResolverAdapter(sessionRepo)

	// Create services
	gitService := application.NewGitService(gitRepo)
	migrationService := application.NewMigrationService(gitRepo, tmuxClient)
	notificationService := application.NewNotificationService(sessionRepo, sessionRepo)
	sessionService := application.NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver)
	settingsService := application.NewSettingsService(sessionRepo)
	shellService := application.NewShellService(sessionRepo, sessionRepo, tmuxClient)

	return &Container{
		EditorOpener:        editorOpener,
		GitRepository:       gitRepo,
		GitService:          gitService,
		MigrationService:    migrationService,
		NotificationService: notificationService,
		SessionRepository:   sessionRepo,
		SessionService:      sessionService,
		SettingsService:     settingsService,
		ShellService:        shellService,
		SoundPlayer:         soundPlayer,
		TmuxClient:          tmuxClient,
	}, nil
}

// Close closes all resources held by the container
func (c *Container) Close() error {
	if c.SessionRepository != nil {
		return c.SessionRepository.Close()
	}
	return nil
}

// NewSessionRepositoryForPath creates a session repository for a specific ROCHA_HOME path.
// This is used for migration operations that need to work with multiple databases.
func NewSessionRepositoryForPath(rochaHomePath string) (ports.SessionRepository, error) {
	return adapterstorage.NewSQLiteRepositoryForPath(rochaHomePath)
}

// ClaudeDirResolverAdapter implements application.ClaudeDirResolver
type ClaudeDirResolverAdapter struct {
	sessionReader ports.SessionReader
}

// NewClaudeDirResolverAdapter creates a new ClaudeDirResolverAdapter
func NewClaudeDirResolverAdapter(sessionReader ports.SessionReader) *ClaudeDirResolverAdapter {
	return &ClaudeDirResolverAdapter{
		sessionReader: sessionReader,
	}
}

// Resolve determines ClaudeDir with precedence:
// 1. User override (if provided and non-empty)
// 2. Existing sessions' ClaudeDir (if repo has sessions with ClaudeDir set)
// 3. Default ~/.claude
func (r *ClaudeDirResolverAdapter) Resolve(repoInfo, userOverride string) string {
	logging.Logger.Debug("Resolving ClaudeDir",
		"repo_info", repoInfo,
		"user_override", userOverride)

	// 1. User override takes precedence
	if userOverride != "" {
		path := config.ExpandPath(userOverride)
		logging.Logger.Info("Using user-provided ClaudeDir", "path", path)
		return path
	}

	// 2. Try to detect from existing sessions
	if repoInfo != "" {
		detected := r.detectClaudeDirForRepo(repoInfo)
		if detected != "" {
			path := config.ExpandPath(detected)
			logging.Logger.Info("Using detected ClaudeDir from existing sessions", "path", path)
			return path
		}
	}

	// 3. Fall back to default
	path := r.defaultClaudeDir()
	logging.Logger.Info("Using default ClaudeDir", "path", path)
	return path
}

// detectClaudeDirForRepo finds ClaudeDir from existing sessions of the same repository
func (r *ClaudeDirResolverAdapter) detectClaudeDirForRepo(repoInfo string) string {
	if repoInfo == "" {
		return ""
	}

	logging.Logger.Debug("Detecting ClaudeDir for repo", "repo_info", repoInfo)

	// List all sessions (including archived)
	sessions, err := r.sessionReader.List(context.Background(), true)
	if err != nil {
		logging.Logger.Warn("Failed to list sessions for ClaudeDir detection", "error", err)
		return ""
	}

	// Find first session with matching repoInfo and non-empty ClaudeDir
	for _, session := range sessions {
		if session.RepoInfo == repoInfo && session.ClaudeDir != "" {
			logging.Logger.Info("Found existing ClaudeDir for repo",
				"repo_info", repoInfo,
				"claude_dir", session.ClaudeDir,
				"from_session", session.Name)
			return session.ClaudeDir
		}
	}

	logging.Logger.Debug("No existing ClaudeDir found for repo", "repo_info", repoInfo)
	return ""
}

// defaultClaudeDir returns the default Claude directory
func (r *ClaudeDirResolverAdapter) defaultClaudeDir() string {
	// Check environment variable first
	if envDir := os.Getenv("CLAUDE_CONFIG_DIR"); envDir != "" {
		return config.ExpandPath(envDir)
	}

	// Fall back to ~/.claude
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.Logger.Warn("Failed to get home directory for ClaudeDir", "error", err)
		return "~/.claude"
	}
	return filepath.Join(homeDir, ".claude")
}
