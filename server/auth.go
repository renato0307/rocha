package server

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rocha/logging"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	gossh "golang.org/x/crypto/ssh"
)

// authMiddleware implements public key authentication
func (s *Server) authMiddleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			pubKey := sess.PublicKey()
			if pubKey == nil {
				logging.Logger.Warn("No public key provided",
					"user", sess.User(),
					"remote_addr", sess.RemoteAddr().String())
				sess.Exit(1)
				return
			}

			// Get key fingerprint for logging
			fingerprint := getKeyFingerprint(pubKey)

			// Check if key is authorized
			homeDir, err := os.UserHomeDir()
			if err != nil {
				logging.Logger.Error("Failed to get home directory",
					"error", err,
					"user", sess.User(),
					"fingerprint", fingerprint)
				sess.Exit(1)
				return
			}

			authorizedKeysPath := filepath.Join(homeDir, ".ssh", "authorized_keys")
			if !isKeyAuthorized(pubKey, authorizedKeysPath) {
				logging.Logger.Warn("Unauthorized key",
					"user", sess.User(),
					"remote_addr", sess.RemoteAddr().String(),
					"fingerprint", fingerprint,
					"key_type", pubKey.Type())
				sess.Exit(1)
				return
			}

			logging.Logger.Info("SSH user authenticated",
				"user", sess.User(),
				"remote_addr", sess.RemoteAddr().String(),
				"fingerprint", fingerprint,
				"key_type", pubKey.Type())

			next(sess)
		}
	}
}

// isKeyAuthorized checks if the client's public key is in authorized_keys
func isKeyAuthorized(clientKey ssh.PublicKey, authorizedKeysPath string) bool {
	file, err := os.Open(authorizedKeysPath)
	if err != nil {
		logging.Logger.Warn("Failed to open authorized_keys", "error", err, "path", authorizedKeysPath)
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the authorized key
		authorizedKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(line))
		if err != nil {
			logging.Logger.Debug("Failed to parse authorized key line", "error", err)
			continue
		}

		// Compare keys
		if bytes.Equal(clientKey.Marshal(), authorizedKey.Marshal()) {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		logging.Logger.Error("Error reading authorized_keys", "error", err)
		return false
	}

	return false
}

// getKeyFingerprint returns the MD5 fingerprint of an SSH public key
// in the format "MD5:xx:xx:xx:..." for security audit trail
func getKeyFingerprint(key ssh.PublicKey) string {
	hash := md5.Sum(key.Marshal())
	fingerprint := make([]string, len(hash))
	for i, b := range hash {
		fingerprint[i] = fmt.Sprintf("%02x", b)
	}
	return "MD5:" + strings.Join(fingerprint, ":")
}

// getKeyComment extracts the comment from an authorized_keys line (for logging)
func getKeyComment(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
