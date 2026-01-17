package version

import "fmt"

// Tagline is the application's tagline used in help text and documentation
const Tagline = "I'm Rocha, and I manage coding agents"

// Build information injected at build time via ldflags
var (
	Version   = "dev"      // Semantic version or "dev"
	Commit    = "unknown"  // Git commit hash
	Date      = "unknown"  // Build date (RFC3339)
	GoVersion = "unknown"  // Go version used
)

// Info returns formatted version information
func Info() string {
	return fmt.Sprintf("rocha %s (commit: %s, built: %s, go: %s)",
		Version, Commit, Date, GoVersion)
}
