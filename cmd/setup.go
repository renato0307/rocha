package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SetupCmd configures tmux automatically
type SetupCmd struct{}

const (
	tmuxConfig = `
# Rocha status bar configuration
set -g status-right "Claude: #(rocha status) | %H:%M"
set -g status-interval 1  # Update every 1 second
`
	pathMarker = "# Added by rocha setup"
)

// Run executes the setup command
func (s *SetupCmd) Run() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Get the directory containing the rocha binary
	rochaBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get rocha binary path: %w", err)
	}
	rochaDir := filepath.Dir(rochaBinary)

	// Setup PATH in shell rc file
	if err := s.setupPath(homeDir, rochaDir); err != nil {
		return err
	}

	// Setup tmux configuration
	if err := s.setupTmux(homeDir); err != nil {
		return err
	}

	fmt.Println("\n✓ Setup complete!")
	fmt.Println("Run 'source ~/.zshrc' or 'source ~/.bashrc' to reload your shell")
	fmt.Println("Then start rocha to see it in action: rocha")

	return nil
}

// setupPath adds rocha directory to PATH in shell rc file (idempotent)
func (s *SetupCmd) setupPath(homeDir, rochaDir string) error {
	// Detect which shell rc file to use
	var rcFiles []string
	if _, err := os.Stat(filepath.Join(homeDir, ".zshrc")); err == nil {
		rcFiles = append(rcFiles, filepath.Join(homeDir, ".zshrc"))
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".bashrc")); err == nil {
		rcFiles = append(rcFiles, filepath.Join(homeDir, ".bashrc"))
	}

	if len(rcFiles) == 0 {
		// No rc file found, create .bashrc
		rcFiles = append(rcFiles, filepath.Join(homeDir, ".bashrc"))
	}

	pathLine := fmt.Sprintf(`export PATH="%s:$PATH" %s`, rochaDir, pathMarker)

	for _, rcFile := range rcFiles {
		// Read existing config
		content, err := os.ReadFile(rcFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", rcFile, err)
		}

		contentStr := string(content)

		// Check if already configured (idempotent)
		if strings.Contains(contentStr, pathMarker) {
			fmt.Printf("✓ PATH already configured in %s\n", filepath.Base(rcFile))
			continue
		}

		// Append PATH configuration
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", rcFile, err)
		}
		defer f.Close()

		if _, err := f.WriteString("\n" + pathLine + "\n"); err != nil {
			return fmt.Errorf("failed to write to %s: %w", rcFile, err)
		}

		fmt.Printf("✓ Added rocha to PATH in %s\n", filepath.Base(rcFile))
	}

	return nil
}

// setupTmux configures tmux status bar (idempotent)
func (s *SetupCmd) setupTmux(homeDir string) error {
	tmuxConfPath := filepath.Join(homeDir, ".tmux.conf")

	// Read existing config
	existingConfig, err := os.ReadFile(tmuxConfPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .tmux.conf: %w", err)
	}

	existingConfigStr := string(existingConfig)

	// Check if config already exists (idempotent)
	if strings.Contains(existingConfigStr, "rocha status") {
		fmt.Println("✓ Tmux configuration already exists in ~/.tmux.conf")
		return nil
	}

	// Append config to file
	f, err := os.OpenFile(tmuxConfPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .tmux.conf: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(tmuxConfig); err != nil {
		return fmt.Errorf("failed to write to .tmux.conf: %w", err)
	}

	fmt.Println("✓ Added rocha configuration to ~/.tmux.conf")

	// Reload tmux configuration if tmux is running
	cmd := exec.Command("tmux", "source-file", tmuxConfPath)
	if err := cmd.Run(); err != nil {
		// It's OK if this fails (tmux might not be running)
		fmt.Println("  Note: tmux is not currently running. Configuration will be loaded when you start tmux.")
	} else {
		fmt.Println("✓ Reloaded tmux configuration")
	}

	return nil
}
