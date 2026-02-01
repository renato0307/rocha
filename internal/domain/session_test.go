package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeSessionName_KeepsAlphanumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"Session123", "Session123"},
		{"test-name", "test-name"},
		{"with.dot", "with.dot"},
		{"under_score", "under_score"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSessionName_SpaceReplacement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single space", "hello world", "hello_world"},
		{"multiple spaces", "hello  world", "hello_world"},
		{"tabs", "hello\tworld", "hello_world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSessionName_ParenthesesReplacement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"open paren", "test(name", "test_name"},
		{"close paren", "test)name", "test_name"},
		{"both parens", "test(name)", "test_name"},
		{"nested parens", "test((name))", "test_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSessionName_SlashReplacement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single slash", "feature/branch", "feature_branch"},
		{"multiple slashes", "path/to/file", "path_to_file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSessionName_ConsecutiveUnderscoreCollapse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"space then paren", "test (name)", "test_name"},
		{"multiple spaces and parens", "test  (name)  here", "test_name_here"},
		{"slash space combo", "path/ name", "path_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSessionName_SpecialCharsRemoved(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"brackets", "test[name]", "testname"},
		{"braces", "test{name}", "testname"},
		{"colon", "test:name", "testname"},
		{"semicolon", "test;name", "testname"},
		{"comma", "test,name", "testname"},
		{"exclamation", "test!name", "testname"},
		{"at sign", "test@name", "testname"},
		{"hash", "test#name", "testname"},
		{"dollar", "test$name", "testname"},
		{"percent", "test%name", "testname"},
		{"caret", "test^name", "testname"},
		{"ampersand", "test&name", "testname"},
		{"asterisk", "test*name", "testname"},
		{"plus", "test+name", "testname"},
		{"equals", "test=name", "testname"},
		{"pipe", "test|name", "testname"},
		{"backslash", "test\\name", "testname"},
		{"single quote", "test'name", "testname"},
		{"double quote", "test\"name", "testname"},
		{"less than", "test<name", "testname"},
		{"greater than", "test>name", "testname"},
		{"question mark", "test?name", "testname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSessionName_TrailingUnderscoreTrimmed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"trailing space", "name ", "name"},
		{"trailing paren", "name)", "name"},
		{"trailing slash", "name/", "name"},
		{"trailing underscore", "name_", "name"},
		{"multiple trailing", "name / ", "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSessionName_LeadingSpecialIgnored(t *testing.T) {
	// Leading spaces/parens/slashes are not converted to underscore at the start
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"leading space", " name", "name"},
		{"leading paren", "(name", "name"},
		{"leading slash", "/name", "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSessionName_EmptyResult(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only special chars", "!@#$%^&*"},
		{"only spaces", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Empty(t, result)
		})
	}
}

func TestSanitizeSessionName_ComplexExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"feature branch", "feature/add-tests (WIP)", "feature_add-tests_WIP"},
		{"github issue", "Fix bug #123", "Fix_bug_123"},
		{"mixed case", "MySession-Test_123.final", "MySession-Test_123.final"},
		{"unicode kept", "session\u00e9", "session\u00e9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSessionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
