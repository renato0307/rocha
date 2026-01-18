package git

import (
	"fmt"
	"rocha/logging"
	"strings"
	"unicode"
)

// ValidateBranchName checks if a branch name is valid according to git rules.
// Returns nil if valid, error with helpful message if invalid.
// Use for user-provided branch names.
//
// Git branch naming rules:
// - Cannot start with '.' or end with '.lock', '.', '/', or '-'
// - Cannot contain '..' or '//' or '@{'
// - Cannot contain '~', '^', ':', '?', '*', '[', ']', '\', spaces, control chars
// - Cannot start or end with '/'
func ValidateBranchName(name string) error {
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

	// Check for invalid characters
	invalidChars := []rune{'~', '^', ':', '?', '*', '[', ']', '\\', ' ', '#', '@', '{', '}'}
	for _, r := range name {
		// Check for control characters
		if unicode.IsControl(r) {
			return fmt.Errorf("branch name cannot contain control characters")
		}
		// Check for specific invalid characters
		for _, invalid := range invalidChars {
			if r == invalid {
				return fmt.Errorf("branch name cannot contain '%c'", invalid)
			}
		}
	}

	// Special check: '@' alone is not valid
	if name == "@" {
		return fmt.Errorf("branch name cannot be '@'")
	}

	return nil
}

// SanitizeBranchName transforms a string into a valid git branch name.
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
func SanitizeBranchName(name string) (string, error) {
	logging.Logger.Debug("Sanitizing branch name", "input", name)

	if name == "" {
		return "", fmt.Errorf("cannot sanitize empty string")
	}

	// Step 1: Lowercase
	result := strings.ToLower(name)

	// Step 2: Replace invalid characters with '-'
	var builder strings.Builder
	invalidChars := map[rune]bool{
		' ': true, '~': true, '^': true, ':': true, '?': true, '*': true,
		'[': true, ']': true, '\\': true, '{': true, '}': true, '#': true, '@': true,
	}

	for _, r := range result {
		// Step 3: Remove control characters (skip them)
		if unicode.IsControl(r) {
			continue
		}
		// Replace invalid characters with '-'
		if invalidChars[r] {
			builder.WriteRune('-')
		} else {
			builder.WriteRune(r)
		}
	}

	result = builder.String()

	// Step 4: Replace '..' with '-'
	result = strings.ReplaceAll(result, "..", "-")

	// Step 5: Replace '//' with '/'
	result = strings.ReplaceAll(result, "//", "/")

	// Step 6: Remove leading '.', '/', '-'
	result = strings.TrimLeft(result, "./-")

	// Remove trailing '.lock', '.', '/', '-'
	result = strings.TrimSuffix(result, ".lock")
	result = strings.TrimRight(result, "./-")

	// Step 7: Collapse consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

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
