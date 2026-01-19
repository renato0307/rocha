package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rocha/config"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/storage"
	"rocha/tmux"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// sessionCreatedMsg is sent when session creation completes
type sessionCreatedMsg struct {
	err error
}

// SessionFormResult contains the result of the session creation form
type SessionFormResult struct {
	AllowDangerouslySkipPermissions bool
	BranchName                      string
	Cancelled                       bool
	ClaudeDir                       string // User-provided CLAUDE_CONFIG_DIR override
	CreateWorktree                  bool
	Error                           error  // Error that occurred during session creation
	RepoSource                      string // User-provided repo path or URL
	SessionName                     string
}

// SessionForm is a Bubble Tea component for creating sessions
type SessionForm struct {
	Completed          bool // Exported so Model can check completion
	cancelled          bool
	creating           bool // True when session creation is in progress
	form               *huh.Form
	result             SessionFormResult
	sessionManager     tmux.SessionManager
	sessionState       *storage.SessionState
	spinner            spinner.Model
	store              *storage.Store
	tmuxStatusPosition string
	worktreePath       string
}

// NewSessionForm creates a new session creation form
func NewSessionForm(sessionManager tmux.SessionManager, store *storage.Store, worktreePath string, sessionState *storage.SessionState, tmuxStatusPosition string, allowDangerouslySkipPermissionsDefault bool) *SessionForm {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	sf := &SessionForm{
		result: SessionFormResult{
			AllowDangerouslySkipPermissions: allowDangerouslySkipPermissionsDefault,
			CreateWorktree:                  true, // Default to true
		},
		sessionManager:     sessionManager,
		sessionState:       sessionState,
		spinner:            s,
		store:              store,
		tmuxStatusPosition: tmuxStatusPosition,
		worktreePath:       worktreePath,
	}

	// Check if we're in a git repository
	cwd, _ := os.Getwd()
	isGit, repo := git.IsGitRepo(cwd)

	// Determine default ClaudeDir for display purposes
	var defaultClaudeDir string
	if isGit {
		repoInfo := git.GetRepoInfo(repo)
		defaultClaudeDir = config.ResolveClaudeDir(store, repoInfo, "")
	} else {
		defaultClaudeDir = config.DefaultClaudeDir()
	}
	// Leave sf.result.ClaudeDir empty - empty means "use default"
	logging.Logger.Debug("Default ClaudeDir for display", "claude_dir", defaultClaudeDir)

	logging.Logger.Debug("Creating session form", "is_git_repo", isGit, "cwd", cwd)

	// Build form fields - all in one group
	var sessionNameField *huh.Input
	if isGit {
		// For git repos, show suggested branch name in session name description
		sessionNameField = huh.NewInput().
			Title("Session name").
			Value(&sf.result.SessionName).
			DescriptionFunc(func() string {
				if sf.result.SessionName != "" {
					if sanitized, err := git.SanitizeBranchName(sf.result.SessionName); err == nil {
						return fmt.Sprintf("Suggested branch name: %s", sanitized)
					}
				}
				return ""
			}, &sf.result.SessionName).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("session name required")
				}
				return nil
			})
	} else {
		// For non-git repos, just show session name field
		sessionNameField = huh.NewInput().
			Title("Session name").
			Value(&sf.result.SessionName).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("session name required")
				}
				return nil
			})
	}

	fields := []huh.Field{
		sessionNameField,
		// Repository source field
		huh.NewInput().
			Title("Repository (optional)").
			DescriptionFunc(func() string {
				if sf.result.RepoSource == "" {
					return "Git URL or local path. Leave empty for current directory."
				}
				// Try to parse and show detected branch
				if repoSource, err := git.ParseRepoSource(sf.result.RepoSource); err == nil && repoSource.Branch != "" {
					return fmt.Sprintf("Detected branch: %s", repoSource.Branch)
				}
				return "Tip: Add #branch-name to specify a branch (e.g., https://github.com/owner/repo#main)"
			}, &sf.result.RepoSource).
			Placeholder("https://github.com/owner/repo#branch-name").
			Value(&sf.result.RepoSource).
			Validate(func(s string) error {
				if s == "" {
					return nil // Empty is OK, will use current directory
				}
				// Check if it's a URL or local path (strip branch fragment first)
				checkPath := s
				if idx := strings.Index(s, "#"); idx >= 0 {
					checkPath = s[:idx]
				}
				if git.IsGitURL(checkPath) {
					return nil // Valid URL format
				}
				// Check if it's an existing directory
				if strings.HasPrefix(checkPath, "~") {
					homeDir, err := os.UserHomeDir()
					if err != nil {
						return fmt.Errorf("failed to expand home directory")
					}
					checkPath = filepath.Join(homeDir, checkPath[1:])
				}
				if _, err := os.Stat(checkPath); err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("local path does not exist")
					}
					return fmt.Errorf("invalid path: %v", err)
				}
				return nil
			}),
	}

	// Only add worktree options if in a git repo OR if repo source is provided
	if isGit || sf.result.RepoSource != "" {
		fields = append(fields,
			huh.NewConfirm().
				Title("Create worktree?").
				Description("Creates an isolated git worktree for this session").
				Value(&sf.result.CreateWorktree).
				Affirmative("Yes").
				Negative("No"),
			huh.NewInput().
				Title("Override branch name").
				Description("Leave empty to use suggested name above. Must match git naming rules.").
				Value(&sf.result.BranchName).
				Validate(func(s string) error {
					if s == "" {
						return nil // Empty is OK, will use suggested name
					}
					// Validate user-provided name
					if err := git.ValidateBranchName(s); err != nil {
						// Show what it would become if sanitized
						sanitized, sanitizeErr := git.SanitizeBranchName(s)
						if sanitizeErr == nil {
							return fmt.Errorf("invalid branch name: %v (suggestion: %s)", err, sanitized)
						}
						return fmt.Errorf("invalid branch name: %v", err)
					}
					return nil
				}),
		)
	}

	// Add CLAUDE_CONFIG_DIR field
	fields = append(fields,
		huh.NewInput().
			Title("Claude directory (optional)").
			Description(fmt.Sprintf("Leave empty to use default: %s", defaultClaudeDir)).
			Placeholder(defaultClaudeDir).
			Value(&sf.result.ClaudeDir).
			Validate(func(s string) error {
				if s == "" {
					return nil // Empty is OK - means use default
				}
				// Basic path validation - just check format
				if !filepath.IsAbs(s) && !strings.HasPrefix(s, "~") {
					return fmt.Errorf("path must be absolute or start with ~")
				}
				return nil
			}),
	)

	// Add skip permissions field for all repos
	fields = append(fields,
		huh.NewConfirm().
			Title("Skip permission prompts? (DANGEROUS)").
			Description("Allows Claude to execute commands without asking. Use with caution!").
			Value(&sf.result.AllowDangerouslySkipPermissions).
			Affirmative("Yes").
			Negative("No"),
	)

	sf.form = huh.NewForm(huh.NewGroup(fields...))

	return sf
}

func (sf *SessionForm) Init() tea.Cmd {
	return sf.form.Init()
}

func (sf *SessionForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle session creation result
	if msg, ok := msg.(sessionCreatedMsg); ok {
		sf.creating = false
		sf.Completed = true
		if msg.err != nil {
			logging.Logger.Error("Failed to create session", "error", msg.err)
			sf.result.Error = msg.err
		}
		return sf, nil
	}

	// If already creating, only update spinner
	if sf.creating {
		var cmd tea.Cmd
		sf.spinner, cmd = sf.spinner.Update(msg)
		return sf, cmd
	}

	// Handle Escape or Ctrl+C to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			sf.cancelled = true
			sf.result.Cancelled = true
			return sf, tea.Quit
		}
	}

	// Forward message to form
	form, cmd := sf.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		sf.form = f
	}

	// Check if form completed - trigger async session creation
	if sf.form.State == huh.StateCompleted && !sf.creating {
		sf.creating = true
		// Return commands to create session and start spinner
		return sf, tea.Batch(sf.createSessionCmd(), sf.spinner.Tick)
	}

	return sf, cmd
}

func (sf *SessionForm) View() string {
	if sf.creating {
		return fmt.Sprintf("\n%s Creating session...\n", sf.spinner.View())
	}
	if sf.form != nil {
		return sf.form.View()
	}
	return ""
}

// Result returns the form result
func (sf *SessionForm) Result() SessionFormResult {
	return sf.result
}

// createSessionCmd returns a command that creates the session asynchronously
func (sf *SessionForm) createSessionCmd() tea.Cmd {
	return func() tea.Msg {
		err := sf.createSession()
		return sessionCreatedMsg{err: err}
	}
}

// createSession creates the tmux session with optional worktree
func (sf *SessionForm) createSession() error {
	sessionName := sf.result.SessionName
	branchName := sf.result.BranchName
	createWorktree := sf.result.CreateWorktree
	repoSource := sf.result.RepoSource

	logging.Logger.Info("Creating session",
		"name", sessionName,
		"create_worktree", createWorktree,
		"branch", branchName,
		"repo_source", repoSource)

	// Generate tmux-compatible name (remove colons, replace spaces/special chars with underscores)
	tmuxName := sanitizeTmuxName(sessionName)

	var claudeDir string
	var repoInfo string
	var repoPath string
	var worktreePath string

	// 1. Determine repository source
	var sourceBranch string // Branch from URL (if specified)
	if repoSource != "" {
		// User provided repo (URL or path)
		logging.Logger.Info("Using user-provided repository source", "source", repoSource)

		// Expand worktree base path
		worktreeBase := sf.worktreePath
		if strings.HasPrefix(worktreeBase, "~/") {
			home, _ := os.UserHomeDir()
			worktreeBase = filepath.Join(home, worktreeBase[2:])
		}

		// Get or clone repository
		localPath, src, err := git.GetOrCloneRepository(repoSource, worktreeBase)
		if err != nil {
			return fmt.Errorf("failed to get repository: %w", err)
		}
		repoPath = localPath
		if src.Owner != "" && src.Repo != "" {
			repoInfo = fmt.Sprintf("%s/%s", src.Owner, src.Repo)
		}
		// Remember branch from URL if specified
		sourceBranch = src.Branch
		if sourceBranch != "" {
			logging.Logger.Info("Branch specified in URL", "branch", sourceBranch)
		}
		logging.Logger.Info("Repository ready", "path", repoPath, "repo_info", repoInfo)
	} else {
		// Use current directory (existing behavior)
		logging.Logger.Info("Using current directory as repository source")
		cwd, _ := os.Getwd()
		isGit, repo := git.IsGitRepo(cwd)
		if isGit {
			repoPath = repo
			repoInfo = git.GetRepoInfo(repo)
			logging.Logger.Info("Extracted repo info from current directory", "repo_info", repoInfo)
		}
	}

	// 2. Resolve ClaudeDir
	claudeDir = config.ResolveClaudeDir(sf.store, repoInfo, sf.result.ClaudeDir)
	logging.Logger.Info("Resolved ClaudeDir", "path", claudeDir)

	// If the resolved ClaudeDir is the same as the default, treat it as "no customization"
	// This ensures we don't set CLAUDE_CONFIG_DIR env var unnecessarily
	defaultDir := config.DefaultClaudeDir()
	if claudeDir == defaultDir {
		logging.Logger.Info("ClaudeDir is default, not setting custom override", "default", defaultDir)
		claudeDir = "" // Empty means: use Claude's default, don't set env var
	}

	// 3. Create worktree if requested
	if createWorktree && repoPath != "" {
		// Auto-generate branch name if not provided (user left field empty)
		if branchName == "" {
			var err error
			branchName, err = git.SanitizeBranchName(sessionName)
			if err != nil {
				return fmt.Errorf("failed to generate branch name from session name: %w", err)
			}
			logging.Logger.Info("Auto-generated branch name from session name", "branch", branchName)
		}

		// Expand worktree base path
		worktreeBase := sf.worktreePath
		if strings.HasPrefix(worktreeBase, "~/") {
			home, _ := os.UserHomeDir()
			worktreeBase = filepath.Join(home, worktreeBase[2:])
		}

		// Build worktree path with repository organization
		worktreePath = git.BuildWorktreePath(worktreeBase, repoInfo, tmuxName)
		logging.Logger.Info("Creating worktree", "path", worktreePath, "branch", branchName)

		// Create the worktree
		if err := git.CreateWorktree(repoPath, worktreePath, branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else if createWorktree && repoPath == "" {
		logging.Logger.Warn("Cannot create worktree: not in a git repository")
	} else if !createWorktree && sourceBranch != "" {
		// Not creating worktree, but branch was specified in URL
		// Store the branch from URL for reference
		branchName = sourceBranch
		logging.Logger.Info("Using branch from URL (no worktree)", "branch", branchName)
	}

	// 4. Create tmux session with ClaudeDir and status position
	session, err := sf.sessionManager.Create(tmuxName, worktreePath, claudeDir, sf.tmuxStatusPosition)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// 5. Save session state with git metadata and new fields
	executionID := os.Getenv("ROCHA_EXECUTION_ID")

	sessionInfo := storage.SessionInfo{
		AllowDangerouslySkipPermissions: sf.result.AllowDangerouslySkipPermissions,
		BranchName:                      branchName,
		ClaudeDir:                       claudeDir,
		DisplayName:                     sessionName,
		ExecutionID:                     executionID,
		LastUpdated:                     time.Now().UTC(),
		Name:                            tmuxName,
		RepoInfo:                        repoInfo,
		RepoPath:                        repoPath,
		RepoSource:                      repoSource,
		State:                           state.StateWaitingUser,
		WorktreePath:                    worktreePath,
	}

	sf.sessionState.Sessions[tmuxName] = sessionInfo

	// Add to database (will be added with position 0 by default, appearing at top)
	if err := sf.store.AddSession(context.Background(), sessionInfo); err != nil {
		logging.Logger.Error("Failed to add session to database", "error", err)
		return err
	}

	logging.Logger.Info("Session created successfully",
		"name", session.Name,
		"claude_dir", claudeDir,
		"repo_source", repoSource)
	return nil
}
