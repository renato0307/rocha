package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"rocha/config"
	"rocha/logging"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	wishlogging "github.com/charmbracelet/wish/logging"
)

// Server represents the SSH server for rocha
type Server struct {
	dbPath     string
	host       string
	port       string
	settings   *config.Settings
	wishServer *ssh.Server
}

// NewServer creates a new SSH server instance
func NewServer(host, port, dbPath string, settings *config.Settings) (*Server, error) {
	s := &Server{
		dbPath:   dbPath,
		host:     host,
		port:     port,
		settings: settings,
	}

	// Ensure SSH directory exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	sshDir := filepath.Join(homeDir, ".rocha", "ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Host key path
	hostKeyPath := filepath.Join(sshDir, "id_ed25519")

	// Create middleware chain
	// Note: Middleware executes in reverse order (last to first)
	wishServer, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%s", host, port)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			// Get key fingerprint for logging
			fingerprint := getKeyFingerprint(key)
			user := ctx.User()

			// Check if key is authorized
			homeDir, err := os.UserHomeDir()
			if err != nil {
				logging.Logger.Error("Failed to get home directory",
					"error", err,
					"user", user,
					"fingerprint", fingerprint)
				return false
			}

			authorizedKeysPath := filepath.Join(homeDir, ".ssh", "authorized_keys")
			authorized := isKeyAuthorized(key, authorizedKeysPath)

			if authorized {
				logging.Logger.Info("SSH key authenticated",
					"user", user,
					"fingerprint", fingerprint,
					"key_type", key.Type())
			} else {
				logging.Logger.Warn("Unauthorized SSH key",
					"user", user,
					"fingerprint", fingerprint,
					"key_type", key.Type())
			}

			return authorized
		}),
		wish.WithMiddleware(
			bubbletea.Middleware(s.teaHandler),
			activeterm.Middleware(), // Require PTY
			wishlogging.Middleware(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH server: %w", err)
	}

	s.wishServer = wishServer
	return s, nil
}

// Start starts the SSH server and blocks until shutdown
func (s *Server) Start() error {
	// Handle graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	logging.Logger.Info("Starting SSH server", "address", fmt.Sprintf("%s:%s", s.host, s.port))
	fmt.Printf("SSH server listening on %s:%s\n", s.host, s.port)

	// Start server in background
	go func() {
		if err := s.wishServer.ListenAndServe(); err != nil {
			logging.Logger.Error("SSH server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-done
	logging.Logger.Info("Shutting down SSH server")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.wishServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown SSH server: %w", err)
	}

	logging.Logger.Info("SSH server stopped")
	return nil
}
