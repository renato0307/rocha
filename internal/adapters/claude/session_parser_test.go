package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTodayUsage_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	assert.Empty(t, usage)
}

func TestGetTodayUsage_NonExistentDirectory(t *testing.T) {
	parser := NewSessionParserWithDir("/non/existent/path")

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	assert.Empty(t, usage)
}

func TestGetTodayUsage_ParsesTokenUsage(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Create JSONL file with today's timestamp
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	content := `{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":5}}}
`
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(content), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 1)
	assert.Equal(t, 100, usage[0].InputTokens)
	assert.Equal(t, 50, usage[0].OutputTokens)
	assert.Equal(t, 10, usage[0].CacheCreation)
	assert.Equal(t, 5, usage[0].CacheRead)
}

func TestGetTodayUsage_FiltersOldEntries(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Create entries from today and yesterday
	now := time.Now()
	todayTimestamp := now.Format(time.RFC3339)
	yesterdayTimestamp := now.AddDate(0, 0, -1).Format(time.RFC3339)

	content := `{"type":"assistant","timestamp":"` + yesterdayTimestamp + `","message":{"usage":{"input_tokens":100,"output_tokens":50}}}
{"type":"assistant","timestamp":"` + todayTimestamp + `","message":{"usage":{"input_tokens":200,"output_tokens":100}}}
`
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(content), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 1)
	assert.Equal(t, 200, usage[0].InputTokens)
}

func TestGetTodayUsage_SkipsNonAssistantMessages(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	content := `{"type":"user","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":100}}}
{"type":"system","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":200}}}
{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":300,"output_tokens":150}}}
`
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(content), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 1)
	assert.Equal(t, 300, usage[0].InputTokens)
}

func TestGetTodayUsage_SkipsEntriesWithoutUsage(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	content := `{"type":"assistant","timestamp":"` + timestamp + `","message":{}}
{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":100,"output_tokens":50}}}
{"type":"assistant","timestamp":"` + timestamp + `"}
`
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(content), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 1)
	assert.Equal(t, 100, usage[0].InputTokens)
}

func TestGetTodayUsage_HandlesInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	content := `not valid json
{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":100,"output_tokens":50}}}
{invalid
`
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(content), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 1)
	assert.Equal(t, 100, usage[0].InputTokens)
}

func TestGetTodayUsage_HandlesInvalidTimestamp(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	content := `{"type":"assistant","timestamp":"invalid-timestamp","message":{"usage":{"input_tokens":100}}}
{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":200,"output_tokens":100}}}
`
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(content), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 1)
	assert.Equal(t, 200, usage[0].InputTokens)
}

func TestGetTodayUsage_ProcessesMultipleProjects(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()
	timestamp := now.Format(time.RFC3339)

	// Create two project directories
	project1 := filepath.Join(tempDir, "project1")
	project2 := filepath.Join(tempDir, "project2")
	require.NoError(t, os.MkdirAll(project1, 0755))
	require.NoError(t, os.MkdirAll(project2, 0755))

	content1 := `{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":100,"output_tokens":50}}}
`
	content2 := `{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":200,"output_tokens":100}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(project1, "session.jsonl"), []byte(content1), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(project2, "session.jsonl"), []byte(content2), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 2)
	// Check total tokens
	totalInput := 0
	for _, u := range usage {
		totalInput += u.InputTokens
	}
	assert.Equal(t, 300, totalInput)
}

func TestGetTodayUsage_ProcessesMultipleJSONLFiles(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	now := time.Now()
	timestamp := now.Format(time.RFC3339)

	content1 := `{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":100,"output_tokens":50}}}
`
	content2 := `{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":200,"output_tokens":100}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session1.jsonl"), []byte(content1), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session2.jsonl"), []byte(content2), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 2)
}

func TestGetTodayUsage_SkipsNonDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file (not a directory) in the projects dir
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "not-a-dir.txt"), []byte("hello"), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	assert.Empty(t, usage)
}

func TestGetTodayUsage_SkipsEmptyLines(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	content := `

{"type":"assistant","timestamp":"` + timestamp + `","message":{"usage":{"input_tokens":100,"output_tokens":50}}}

`
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(content), 0644))

	parser := NewSessionParserWithDir(tempDir)

	usage, err := parser.GetTodayUsage()

	require.NoError(t, err)
	require.Len(t, usage, 1)
	assert.Equal(t, 100, usage[0].InputTokens)
}

func TestNewSessionParser_DefaultDirectory(t *testing.T) {
	parser := NewSessionParser()

	// Just verify it doesn't panic and creates a parser
	assert.NotNil(t, parser)
}
