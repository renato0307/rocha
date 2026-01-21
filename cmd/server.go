package cmd

import (
	"fmt"

	"rocha/logging"
	"rocha/server"
)

// ServerCmd starts the SSH server
type ServerCmd struct {
	Host string `help:"Host to bind to" default:"localhost"`
	Port string `help:"Port to listen on" default:"23234"`
}

// Run executes the server command
func (s *ServerCmd) Run(cli *CLI) error {
	logging.Logger.Info("Starting rocha SSH server",
		"host", s.Host,
		"port", s.Port,
		"db_path", cli.DBPath)

	// Expand database path
	dbPath := expandPath(cli.DBPath)

	// Create server
	srv, err := server.NewServer(s.Host, s.Port, dbPath, cli.settings)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server (blocks until shutdown)
	return srv.Start()
}
