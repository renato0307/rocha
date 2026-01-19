package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/storage"
	"rocha/tmux"
	"strings"
	"time"

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
	SessionName    string
	BranchName     string
	CreateWorktree bool
	Cancelled      bool
	Error          error // Error that occurred during session creation
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
func NewSessionForm(sessionManager tmux.SessionManager, store *storage.Store, worktreePath string, sessionState *storage.SessionState, tmuxStatusPosition string) *SessionForm {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	sf := &SessionForm{
		result: SessionFormResult{
			CreateWorktree: true, // Default to true
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
	isGit, _ := git.IsGitRepo(cwd)

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

	fields := []huh.Field{sessionNameField}

	// Only add worktree options if in a git repo
	if isGit {
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

	logging.Logger.Info("Creating session", "name", sessionName, "create_worktree", createWorktree, "branch", branchName)

	// Generate tmux-compatible name (remove colons, replace spaces/special chars with underscores)
	tmuxName := sanitizeTmuxName(sessionName)

	var worktreePath string
	var repoPath string
	var repoInfo string

	// Handle worktree creation if requested AND we're in a git repo
	if createWorktree {
		cwd, _ := os.Getwd()
		isGit, repo := git.IsGitRepo(cwd)

		if !isGit {
			logging.Logger.Warn("Not in git repo, cannot create worktree")
		} else {
			repoPath = repo
			repoInfo = git.GetRepoInfo(repo)
			logging.Logger.Info("Extracted repo info", "repo_info", repoInfo)

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
		}
	}

	// Create tmux session
	session, err := sf.sessionManager.Create(tmuxName, worktreePath, sf.tmuxStatusPosition)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Save session state with git metadata
	executionID := os.Getenv("ROCHA_EXECUTION_ID")

	// Create session info
	sessionInfo := storage.SessionInfo{
		Name:         tmuxName,
		DisplayName:  sessionName,
		State:        state.StateWaitingUser,
		ExecutionID:  executionID,
		LastUpdated:  time.Now().UTC(),
		RepoPath:     repoPath,
		RepoInfo:     repoInfo,
		BranchName:   branchName,
		WorktreePath: worktreePath,
	}

	sf.sessionState.Sessions[tmuxName] = sessionInfo

	// Add to database (will be added with position 0 by default, appearing at top)
	if err := sf.store.AddSession(context.Background(), sessionInfo); err != nil {
		logging.Logger.Error("Failed to add session to database", "error", err)
		return err
	}

	logging.Logger.Info("Session created successfully", "name", session.Name)
	return nil
}
