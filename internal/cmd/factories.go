package cmd

import (
	"context"
	"os"
	"path/filepath"

	adaptereditor "github.com/renato0307/rocha/internal/adapters/editor"
	adaptergit "github.com/renato0307/rocha/internal/adapters/git"
	adaptersound "github.com/renato0307/rocha/internal/adapters/sound"
	adapterstorage "github.com/renato0307/rocha/internal/adapters/storage"
	adaptertmux "github.com/renato0307/rocha/internal/adapters/tmux"
	"github.com/renato0307/rocha/internal/config"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/services"
)

// Container holds all dependencies for the application
type Container struct {
	// Services
	GitService          *services.GitService
	MigrationService    *services.MigrationService
	NotificationService *services.NotificationService
	SessionService      *services.SessionService
	SettingsService     *services.SettingsService
	ShellService        *services.ShellService

	// Internal - for cleanup only
	sessionRepo ports.SessionRepository
}

// NewContainer creates a new Container with all dependencies wired
func NewContainer(tmuxClient ports.TmuxClient) (*Container, error) {
	// Create adapters
	sessionRepo, err := adapterstorage.NewSQLiteRepository(config.GetDBPath())
	if err != nil {
		return nil, err
	}

	// Create default tmux client if not provided
	if tmuxClient == nil {
		tmuxClient = adaptertmux.NewClient()
	}

	editorOpener := adaptereditor.NewOpener()
	gitRepo := adaptergit.NewCLIRepository()
	soundPlayer := adaptersound.NewPlayer()

	// Create ClaudeDir resolver
	claudeDirResolver := NewClaudeDirResolverAdapter(sessionRepo)

	// Create repository factory for migration service
	repoFactory := func(rochaHomePath string) (ports.SessionRepository, error) {
		return adapterstorage.NewSQLiteRepositoryForPath(rochaHomePath)
	}

	// Create services
	gitService := services.NewGitService(gitRepo)
	migrationService := services.NewMigrationService(gitRepo, tmuxClient, repoFactory)
	notificationService := services.NewNotificationService(sessionRepo, sessionRepo, soundPlayer)
	sessionService := services.NewSessionService(sessionRepo, gitRepo, tmuxClient, claudeDirResolver)
	settingsService := services.NewSettingsService(sessionRepo)
	shellService := services.NewShellService(sessionRepo, sessionRepo, tmuxClient, editorOpener)

	return &Container{
		GitService:          gitService,
		MigrationService:    migrationService,
		NotificationService: notificationService,
		SessionService:      sessionService,
		SettingsService:     settingsService,
		ShellService:        shellService,
		sessionRepo:         sessionRepo,
	}, nil
}

// Close closes all resources held by the container
func (c *Container) Close() error {
	if c.sessionRepo != nil {
		return c.sessionRepo.Close()
	}
	return nil
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
