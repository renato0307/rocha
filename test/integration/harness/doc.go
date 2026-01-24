// Package harness provides utilities for integration testing the rocha CLI.
// It handles binary compilation, environment isolation, and command execution.
//
// Environment variables managed:
//   - ROCHA_HOME: Isolated per test (temp directory)
//   - ROCHA_DEBUG: Disabled to reduce noise
//   - ROCHA_EDITOR: Set to prevent interactive prompts
package harness
