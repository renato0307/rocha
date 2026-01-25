package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"rocha/internal/config"
	"rocha/internal/domain"
	"rocha/internal/logging"
	"rocha/internal/services"
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
	gitService         *services.GitService
	result             SessionFormResult
	sessionService     *services.SessionService
	sessionState       *domain.SessionCollection
	spinner            spinner.Model
	tmuxStatusPosition string
}

// NewSessionForm creates a new session creation form
func NewSessionForm(
	gitService *services.GitService,
	sessionService *services.SessionService,
	sessionState *domain.SessionCollection,
	tmuxStatusPosition string,
	allowDangerouslySkipPermissionsDefault bool,
	defaultRepoSource string,
) *SessionForm {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	sf := &SessionForm{
		gitService: gitService,
		result: SessionFormResult{
			AllowDangerouslySkipPermissions: allowDangerouslySkipPermissionsDefault,
			RepoSource:                      defaultRepoSource,
		},
		sessionService:     sessionService,
		sessionState:       sessionState,
		spinner:            s,
		tmuxStatusPosition: tmuxStatusPosition,
	}

	logging.Logger.Debug("Creating session form with default values",
		"allow_dangerously_skip_permissions_default", allowDangerouslySkipPermissionsDefault,
		"default_repo_source", defaultRepoSource)

	// Check if we're in a git repository
	cwd, _ := os.Getwd()
	isGit, repo := sf.gitService.IsGitRepo(cwd)

	// Determine default ClaudeDir for display purposes
	var defaultClaudeDir string
	if isGit {
		repoInfo := sf.gitService.GetRepoInfo(repo)
		defaultClaudeDir = sf.sessionService.ResolveClaudeDir(repoInfo, "")
	} else {
		defaultClaudeDir = config.DefaultClaudeDir()
	}
	logging.Logger.Debug("Default ClaudeDir for display", "claude_dir", defaultClaudeDir)

	logging.Logger.Debug("Creating session form", "is_git_repo", isGit, "cwd", cwd)

	// Build form fields
	sessionNameField := huh.NewInput().
		Title("Session name").
		Value(&sf.result.SessionName).
		DescriptionFunc(func() string {
			if sf.result.SessionName != "" {
				if sanitized, err := sf.gitService.SanitizeBranchName(sf.result.SessionName); err == nil {
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

	fields := []huh.Field{
		sessionNameField,
		huh.NewInput().
			Title("Repository (optional)").
			DescriptionFunc(func() string {
				if sf.result.RepoSource == "" {
					return "Git remote URL. Leave empty for current directory."
				}
				if repoSource, err := sf.gitService.ParseRepoSource(sf.result.RepoSource); err == nil && repoSource.Branch != "" {
					return fmt.Sprintf("Detected branch: %s", repoSource.Branch)
				}
				return "Tip: Add #branch-name to specify a remote branch (e.g., https://github.com/owner/repo#main)"
			}, &sf.result.RepoSource).
			Placeholder("https://github.com/owner/repo#branch-name").
			Value(&sf.result.RepoSource).
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				checkPath := s
				if idx := strings.Index(s, "#"); idx >= 0 {
					checkPath = s[:idx]
				}
				if sf.gitService.IsGitURL(checkPath) {
					return nil
				}
				return fmt.Errorf("must be a git URL (e.g., https://github.com/owner/repo or git@github.com:owner/repo)")
			}),
	}

	fields = append(fields,
		huh.NewInput().
			Title("Override branch name (optional)").
			Description("Leave empty to use suggested name above. Must match git naming rules.").
			Value(&sf.result.BranchName).
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				if err := sf.gitService.ValidateBranchName(s); err != nil {
					sanitized, sanitizeErr := sf.gitService.SanitizeBranchName(s)
					if sanitizeErr == nil {
						return fmt.Errorf("invalid branch name: %v (suggestion: %s)", err, sanitized)
					}
					return fmt.Errorf("invalid branch name: %v", err)
				}
				return nil
			}),
	)

	fields = append(fields,
		huh.NewInput().
			Title("Claude directory (optional)").
			Description(fmt.Sprintf("Leave empty to use default: %s", defaultClaudeDir)).
			Placeholder(defaultClaudeDir).
			Value(&sf.result.ClaudeDir).
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				if !filepath.IsAbs(s) && !strings.HasPrefix(s, "~") {
					return fmt.Errorf("path must be absolute or start with ~")
				}
				return nil
			}),
	)

	logging.Logger.Debug("Creating skip permissions field",
		"current_value", sf.result.AllowDangerouslySkipPermissions)
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
	if msg, ok := msg.(sessionCreatedMsg); ok {
		sf.creating = false
		sf.Completed = true
		if msg.err != nil {
			logging.Logger.Error("Failed to create session", "error", msg.err)
			sf.result.Error = msg.err
		}
		return sf, nil
	}

	if sf.creating {
		var cmd tea.Cmd
		sf.spinner, cmd = sf.spinner.Update(msg)
		return sf, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
			sf.Completed = true
			sf.cancelled = true
			sf.result.Cancelled = true
			return sf, nil
		}
	}

	form, cmd := sf.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		sf.form = f
	}

	if sf.form.State == huh.StateCompleted && !sf.creating {
		sf.creating = true
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
	params := services.CreateSessionParams{
		AllowDangerouslySkipPermissions: sf.result.AllowDangerouslySkipPermissions,
		BranchNameOverride:              sf.result.BranchName,
		ClaudeDirOverride:               sf.result.ClaudeDir,
		RepoSource:                      sf.result.RepoSource,
		SessionName:                     sf.result.SessionName,
		TmuxStatusPosition:              sf.tmuxStatusPosition,
	}

	result, err := sf.sessionService.CreateSession(context.Background(), params)
	if err != nil {
		return err
	}

	// Update sessionState with the new session (for UI refresh)
	if result.Session != nil {
		sf.sessionState.Sessions[result.Session.Name] = *result.Session
	}

	logging.Logger.Info("Session created",
		"name", result.Session.Name,
		"worktree_path", result.WorktreePath)
	return nil
}
