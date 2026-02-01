package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGitSetup holds paths for a complete git test environment.
// It creates a bare repo (simulating remote/origin) and a clone with origin configured.
type TestGitSetup struct {
	BareRepoPath string // Acts as "origin" remote
	ClonePath    string // Working repo with origin configured
	tb           testing.TB
}

// NewTestGitSetup creates a complete git environment with origin.
//  1. Creates a bare repo (simulates remote/origin)
//  2. Clones it to create a working repo with origin remote
//  3. Creates initial commit so branches work
//
// Setup structure:
//
//	tb.TempDir()/
//	├── bare/           <- git init --bare (acts as origin)
//	└── clone/          <- git clone bare/ clone/ (has origin remote)
//	    └── worktrees/  <- git worktree add ...
func NewTestGitSetup(tb testing.TB) *TestGitSetup {
	tb.Helper()

	baseDir := tb.TempDir()
	bareRepoPath := filepath.Join(baseDir, "bare")
	clonePath := filepath.Join(baseDir, "clone")

	// Create bare repository (acts as origin)
	runGitCommand(tb, baseDir, "init", "--bare", bareRepoPath)

	// Clone bare repo to create working repo with origin
	runGitCommand(tb, baseDir, "clone", bareRepoPath, clonePath)

	// Configure git user for commits
	runGitCommand(tb, clonePath, "config", "user.email", "test@example.com")
	runGitCommand(tb, clonePath, "config", "user.name", "Test User")

	// Create initial commit so branches can be created
	dummyFile := filepath.Join(clonePath, "README.md")
	if err := os.WriteFile(dummyFile, []byte("# Test Repo\n"), 0644); err != nil {
		tb.Fatalf("Failed to create dummy file: %v", err)
	}
	runGitCommand(tb, clonePath, "add", "README.md")
	runGitCommand(tb, clonePath, "commit", "-m", "Initial commit")

	// Ensure branch is named "main" (git might default to "master")
	runGitCommand(tb, clonePath, "branch", "-M", "main")

	// Push initial commit to origin
	runGitCommand(tb, clonePath, "push", "-u", "origin", "main")

	return &TestGitSetup{
		BareRepoPath: bareRepoPath,
		ClonePath:    clonePath,
		tb:           tb,
	}
}

// CreateBranch creates a branch in the working repo.
func (g *TestGitSetup) CreateBranch(name string) {
	g.tb.Helper()
	runGitCommand(g.tb, g.ClonePath, "branch", name)
}

// PushBranch pushes a branch to origin (bare repo).
func (g *TestGitSetup) PushBranch(name string) {
	g.tb.Helper()
	runGitCommand(g.tb, g.ClonePath, "push", "-u", "origin", name)
}

// CreateWorktree creates a git worktree at the given path for the specified branch.
// Returns the full path to the created worktree.
func (g *TestGitSetup) CreateWorktree(path, branch string) string {
	g.tb.Helper()

	// Ensure branch exists
	g.CreateBranch(branch)

	// Create worktree
	runGitCommand(g.tb, g.ClonePath, "worktree", "add", path, branch)

	return path
}

// CreateRemoteBranch creates a branch that exists only on origin.
// It creates a local branch, pushes it to origin, then deletes the local branch.
func (g *TestGitSetup) CreateRemoteBranch(name string) {
	g.tb.Helper()

	// Create and checkout branch
	runGitCommand(g.tb, g.ClonePath, "checkout", "-b", name)

	// Create a commit on this branch
	dummyFile := filepath.Join(g.ClonePath, name+".txt")
	if err := os.WriteFile(dummyFile, []byte("content for "+name+"\n"), 0644); err != nil {
		g.tb.Fatalf("Failed to create file for branch %s: %v", name, err)
	}
	runGitCommand(g.tb, g.ClonePath, "add", name+".txt")
	runGitCommand(g.tb, g.ClonePath, "commit", "-m", "Add file for "+name)

	// Push to origin
	runGitCommand(g.tb, g.ClonePath, "push", "-u", "origin", name)

	// Switch back to main and delete local branch
	runGitCommand(g.tb, g.ClonePath, "checkout", "main")
	runGitCommand(g.tb, g.ClonePath, "branch", "-D", name)
}

// GetClonePath returns the path to the clone directory.
func (g *TestGitSetup) GetClonePath() string {
	return g.ClonePath
}

// GetBareRepoPath returns the path to the bare repository (origin).
func (g *TestGitSetup) GetBareRepoPath() string {
	return g.BareRepoPath
}

// RunGitCommand executes a git command in the specified directory (exported for tests).
func RunGitCommand(tb testing.TB, dir string, args ...string) {
	runGitCommand(tb, dir, args...)
}

// runGitCommand executes a git command in the specified directory.
func runGitCommand(tb testing.TB, dir string, args ...string) {
	tb.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("git %v failed in %s: %v\nOutput: %s", args, dir, err, output)
	}
}
