package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"rocha/git"
	"rocha/logging"
	"rocha/state"
	"rocha/tmux"
	"strings"

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
	sessionManager tmux.SessionManager
	form           *huh.Form
	worktreePath   string
	sessionState   *state.SessionState
	result         SessionFormResult
	Completed      bool    // Exported so Model can check completion
	cancelled      bool
	creating       bool    // True when session creation is in progress
	spinner        spinner.Model
}

// NewSessionForm creates a new session creation form
func NewSessionForm(sessionManager tmux.SessionManager, worktreePath string, sessionState *state.SessionState) *SessionForm {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	sf := &SessionForm{
		sessionManager: sessionManager,
		worktreePath:   worktreePath,
		sessionState:   sessionState,
		result: SessionFormResult{
			CreateWorktree: true, // Default to true
		},
		spinner: s,
	}

	// Check if we're in a git repository
	cwd, _ := os.Getwd()
	isGit, _ := git.IsGitRepo(cwd)

	logging.Logger.Debug("Creating session form", "is_git_repo", isGit, "cwd", cwd)

	// Build form fields
	fields := []huh.Field{
		huh.NewInput().
			Title("Session name").
			Value(&sf.result.SessionName).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("session name required")
				}
				return nil
			}),
	}

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
				Title("Branch name").
				Description("Leave empty to auto-generate from session name").
				Value(&sf.result.BranchName).
				Placeholder("(auto-generated)"),
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

			// Auto-generate branch name if not provided
			if branchName == "" {
				branchName = strings.ReplaceAll(sessionName, " ", "-")
				logging.Logger.Info("Auto-generated branch name", "branch", branchName)
			}

			// Expand worktree base path
			worktreeBase := sf.worktreePath
			if strings.HasPrefix(worktreeBase, "~/") {
				home, _ := os.UserHomeDir()
				worktreeBase = filepath.Join(home, worktreeBase[2:])
			}

			// Use tmux name (no spaces) for worktree directory
			worktreePath = filepath.Join(worktreeBase, tmuxName)
			logging.Logger.Info("Creating worktree", "path", worktreePath, "branch", branchName)

			// Create the worktree
			if err := git.CreateWorktree(repoPath, worktreePath, branchName); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		}
	}

	// Create tmux session
	session, err := sf.sessionManager.Create(tmuxName, worktreePath)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Save session state with git metadata
	executionID := os.Getenv("ROCHA_EXECUTION_ID")
	if err := sf.sessionState.UpdateSessionWithGit(tmuxName, sessionName, state.StateWaitingUser, executionID, repoPath, repoInfo, branchName, worktreePath); err != nil {
		logging.Logger.Error("Failed to save session state", "error", err)
	}

	// Add new session to top of manual order
	if err := sf.sessionState.AddSessionToTop(tmuxName); err != nil {
		logging.Logger.Error("Failed to add session to top of order", "error", err)
	}

	logging.Logger.Info("Session created successfully", "name", session.Name)
	return nil
}
