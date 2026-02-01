package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a git repo with initial commit for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	runGit("init")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")

	// Create initial commit
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# Test"), 0644))
	runGit("add", "README.md")
	runGit("commit", "-m", "Initial commit")

	return dir
}

func TestGetWorktreeForBranch_BranchDoesNotExist(t *testing.T) {
	repoPath := setupTestRepo(t)

	result, err := getWorktreeForBranch(repoPath, "nonexistent-branch")

	assert.NoError(t, err)
	assert.Empty(t, result, "should return empty for non-existent branch")
}

func TestGetWorktreeForBranch_BranchExistsNoWorktree(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create branch but no worktree
	cmd := exec.Command("git", "branch", "feature-branch")
	cmd.Dir = repoPath
	require.NoError(t, cmd.Run())

	result, err := getWorktreeForBranch(repoPath, "feature-branch")

	assert.NoError(t, err)
	assert.Empty(t, result, "should return empty when branch exists but no worktree")
}

func TestGetWorktreeForBranch_WorktreeExists(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create branch and worktree
	worktreePath := filepath.Join(t.TempDir(), "my-worktree")
	cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature-branch")
	cmd.Dir = repoPath
	require.NoError(t, cmd.Run())

	result, err := getWorktreeForBranch(repoPath, "feature-branch")

	assert.NoError(t, err)
	assert.Equal(t, worktreePath, result, "should return existing worktree path")
}

func TestGetWorktreeForBranch_SkipsMainDirectory(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create a worktree at a path ending with /.main
	mainWorktreePath := filepath.Join(t.TempDir(), "repo", ".main")
	require.NoError(t, os.MkdirAll(filepath.Dir(mainWorktreePath), 0755))

	cmd := exec.Command("git", "worktree", "add", mainWorktreePath, "-b", "feature-branch")
	cmd.Dir = repoPath
	require.NoError(t, cmd.Run())

	result, err := getWorktreeForBranch(repoPath, "feature-branch")

	assert.NoError(t, err)
	assert.Empty(t, result, "should skip worktree at .main path")
}
