// Package integration_test provides end-to-end tests for rocha CLI commands.
// Tests compile the binary once via TestMain and run each test with an
// isolated ROCHA_HOME to ensure test independence.
package integration_test

import (
	"log"
	"os"
	"testing"

	"github.com/renato0307/rocha/test/integration/harness"
)

func TestMain(m *testing.M) {
	// Build binary once before all tests
	_, err := harness.BuildBinary()
	if err != nil {
		log.Fatalf("Failed to build binary: %v", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	harness.CleanupBinary()

	os.Exit(code)
}
