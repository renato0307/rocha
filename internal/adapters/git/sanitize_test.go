package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBranchName_EmptyName(t *testing.T) {
	err := validateBranchName("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateBranchName_InvalidPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"starts with dot", ".hidden", "start with '.'"},
		{"starts with slash", "/path", "start with '/'"},
		{"starts with hyphen", "-feature", "start with '-'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestValidateBranchName_InvalidSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"ends with .lock", "branch.lock", ".lock"},
		{"ends with dot", "branch.", "end with '.'"},
		{"ends with slash", "branch/", "end with '/'"},
		{"ends with hyphen", "branch-", "end with '-'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestValidateBranchName_InvalidSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"double dot", "feature..branch", "'..'"},
		{"double slash", "feature//branch", "'//'"},
		{"at brace", "branch@{0}", "'@{'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestValidateBranchName_ControlCharacters(t *testing.T) {
	err := validateBranchName("feature\x00branch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "control characters")
}

func TestValidateBranchName_InvalidCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"space", "feature branch"},
		{"tilde", "feature~1"},
		{"caret", "feature^1"},
		{"colon", "feature:1"},
		{"question mark", "feature?"},
		{"asterisk", "feature*"},
		{"bracket", "feature[0]"},
		{"backslash", "feature\\path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid characters")
		})
	}
}

func TestValidateBranchName_AtSymbolAlone(t *testing.T) {
	err := validateBranchName("@")
	require.Error(t, err)
	// '@' is not in valid chars, so it fails with invalid characters error first
	assert.Contains(t, err.Error(), "invalid characters")
}

func TestValidateBranchName_ValidNames(t *testing.T) {
	tests := []string{
		"main",
		"feature/add-tests",
		"fix_bug_123",
		"release-1.0.0",
		"user/feature.name",
		"a",
		"UPPERCASE",
		"MixedCase123",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			err := validateBranchName(input)
			assert.NoError(t, err)
		})
	}
}

func TestSanitizeBranchName_EmptyInput(t *testing.T) {
	_, err := sanitizeBranchName("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestSanitizeBranchName_Lowercase(t *testing.T) {
	result, err := sanitizeBranchName("FEATURE-BRANCH")
	require.NoError(t, err)
	assert.Equal(t, "feature-branch", result)
}

func TestSanitizeBranchName_InvalidCharReplacement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"spaces", "feature branch name", "feature-branch-name"},
		{"special chars", "feature@#$branch", "feature-branch"},
		{"shell metachar", "feature&branch", "feature-branch"},
		{"parentheses", "feature(test)", "feature-test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeBranchName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeBranchName_ConsecutiveHyphenCollapse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"double hyphen", "feature--branch", "feature-branch"},
		{"triple hyphen", "feature---branch", "feature-branch"},
		{"multiple special chars", "feature@#$branch", "feature-branch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeBranchName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeBranchName_ControlCharRemoval(t *testing.T) {
	result, err := sanitizeBranchName("feature\x00\x01branch")
	require.NoError(t, err)
	assert.Equal(t, "featurebranch", result)
}

func TestSanitizeBranchName_TrimLeadingChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"leading dot", ".hidden-branch", "hidden-branch"},
		{"leading slash", "/path/branch", "path/branch"},
		{"leading hyphen", "-feature", "feature"},
		{"multiple leading", "./-feature", "feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeBranchName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeBranchName_TrimTrailingChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"trailing .lock", "branch.lock", "branch"},
		{"trailing dot", "branch.", "branch"},
		{"trailing slash", "branch/", "branch"},
		{"trailing hyphen", "branch-", "branch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeBranchName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeBranchName_DoubleDotReplacement(t *testing.T) {
	result, err := sanitizeBranchName("feature..branch")
	require.NoError(t, err)
	assert.Equal(t, "feature-branch", result)
}

func TestSanitizeBranchName_DoubleSlashCollapse(t *testing.T) {
	result, err := sanitizeBranchName("feature//branch")
	require.NoError(t, err)
	assert.Equal(t, "feature/branch", result)
}

func TestSanitizeBranchName_ResultEmpty(t *testing.T) {
	_, err := sanitizeBranchName("...")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestSanitizeBranchName_ResultIsAtSymbol(t *testing.T) {
	// "@" is replaced by "-" which then gets trimmed, resulting in empty
	_, err := sanitizeBranchName("@")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestSanitizeBranchName_ValidOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "feature", "feature"},
		{"with slash", "feature/test", "feature/test"},
		{"with numbers", "release-1.0.0", "release-1.0.0"},
		{"mixed case", "Feature-Branch", "feature-branch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeBranchName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
