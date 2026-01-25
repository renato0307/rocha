package git

import (
	"fmt"
	"regexp"
	"github.com/renato0307/rocha/internal/logging"
	"strings"
	"unicode"
)

// validBranchNameChars matches valid characters for git branch names
// Allows: alphanumeric, hyphens, underscores, dots, slashes
var validBranchNameChars = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

// invalidBranchNameChars matches characters that should be replaced with hyphens
// Includes:
// - Git-prohibited: space, ~, ^, :, ?, *, [, \
// - Shell metacharacters: &, |, ;, <, >, $, `, ', "
// - Other problematic: #, @, {, }, (, )
var invalidBranchNameChars = regexp.MustCompile(`[\s~^:?*\[\]\\{}#@()&|;<>$` + "`" + `'"]+`)

// consecutiveHyphens matches two or more consecutive hyphens
var consecutiveHyphens = regexp.MustCompile(`-{2,}`)

// validateBranchName checks if a branch name is valid according to git rules.
// Returns nil if valid, error with helpful message if invalid.
// Use for user-provided branch names.
//
// Note: We enforce stricter rules than git's git-check-ref-format because rocha
// executes git commands via shell. While git allows shell metacharacters like
// &, |, ;, etc., they require quoting in shell commands and can cause issues.
//
// Git branch naming rules enforced:
// - Cannot start with '.' or end with '.lock', '.', '/', or '-'
// - Cannot contain '..' or '//' or '@{'
// - Cannot contain git-prohibited chars: ~, ^, :, ?, *, [, \, space, control chars
// - Cannot contain shell metacharacters: &, |, ;, <, >, $, `, ', "
// - Cannot start or end with '/'
func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check for invalid starting characters
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("branch name cannot start with '.'")
	}
	if strings.HasPrefix(name, "/") {
		return fmt.Errorf("branch name cannot start with '/'")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("branch name cannot start with '-'")
	}

	// Check for invalid ending
	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("branch name cannot end with '.lock'")
	}
	if strings.HasSuffix(name, ".") {
		return fmt.Errorf("branch name cannot end with '.'")
	}
	if strings.HasSuffix(name, "/") {
		return fmt.Errorf("branch name cannot end with '/'")
	}
	if strings.HasSuffix(name, "-") {
		return fmt.Errorf("branch name cannot end with '-'")
	}

	// Check for invalid sequences
	if strings.Contains(name, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}
	if strings.Contains(name, "//") {
		return fmt.Errorf("branch name cannot contain '//'")
	}
	if strings.Contains(name, "@{") {
		return fmt.Errorf("branch name cannot contain '@{'")
	}

	// Check for control characters
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("branch name cannot contain control characters")
		}
	}

	// Check for valid characters using regex
	if !validBranchNameChars.MatchString(name) {
		return fmt.Errorf("branch name contains invalid characters (only alphanumeric, '.', '_', '-', '/' allowed)")
	}

	// Special check: '@' alone is not valid
	if name == "@" {
		return fmt.Errorf("branch name cannot be '@'")
	}

	return nil
}

// sanitizeBranchName transforms a string into a valid git branch name.
// Returns error if result would be empty after sanitization.
// Use for auto-generated branch names.
//
// Sanitization process:
// 1. Lowercase the input
// 2. Replace invalid characters with '-'
// 3. Remove control characters
// 4. Replace '..' with '-'
// 5. Replace '//' with '/'
// 6. Remove leading '.', '/', '-' and trailing '.lock', '.', '/', '-'
// 7. Collapse consecutive hyphens
// 8. Return error if result is empty or '@'
func sanitizeBranchName(name string) (string, error) {
	logging.Logger.Debug("Sanitizing branch name", "input", name)

	if name == "" {
		return "", fmt.Errorf("cannot sanitize empty string")
	}

	// Step 1: Lowercase
	result := strings.ToLower(name)

	// Step 2: Remove control characters
	var builder strings.Builder
	for _, r := range result {
		if !unicode.IsControl(r) {
			builder.WriteRune(r)
		}
	}
	result = builder.String()

	// Step 3: Replace invalid characters with '-' using regex
	result = invalidBranchNameChars.ReplaceAllString(result, "-")

	// Step 4: Replace '..' with '-'
	result = strings.ReplaceAll(result, "..", "-")

	// Step 5: Replace '//' with '/'
	result = strings.ReplaceAll(result, "//", "/")

	// Step 6: Remove leading '.', '/', '-'
	result = strings.TrimLeft(result, "./-")

	// Remove trailing '.lock', '.', '/', '-'
	result = strings.TrimSuffix(result, ".lock")
	result = strings.TrimRight(result, "./-")

	// Step 7: Collapse consecutive hyphens using regex
	result = consecutiveHyphens.ReplaceAllString(result, "-")

	// Step 8: Check if result is valid
	if result == "" {
		return "", fmt.Errorf("sanitization resulted in empty branch name")
	}
	if result == "@" {
		return "", fmt.Errorf("sanitization resulted in invalid branch name '@'")
	}

	logging.Logger.Info("Branch name sanitized", "input", name, "output", result)
	return result, nil
}
