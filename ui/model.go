package ui

import (
	"fmt"
	"log"
	"os/exec"
	"rocha/state"
	"rocha/tmux"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Padding(1, 0)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 0)
)

type uiState int

const (
	stateList uiState = iota
	stateCreating
)

type Model struct {
	sessions  []*tmux.Session
	cursor    int
	state     uiState
	nameInput string
	width     int
	height    int
	err       error
}

func NewModel() Model {
	sessions, err := tmux.List()
	var errMsg error
	if err != nil {
		errMsg = fmt.Errorf("failed to load sessions: %w", err)
		sessions = []*tmux.Session{}
	}
	return Model{
		sessions: sessions,
		cursor:   0,
		state:    stateList,
		err:      errMsg,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateCreating:
		return m.updateCreating(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		m.err = msg
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}

		case "n":
			m.state = stateCreating
			m.nameInput = ""

		case "enter":
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				// Use tea.ExecProcess to suspend Bubble Tea and attach to tmux
				session := m.sessions[m.cursor]
				c := exec.Command("tmux", "attach-session", "-t", session.Name)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					if err != nil {
						return err
					}
					return detachedMsg{}
				})
			}

		case "x":
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				if err := session.Kill(); err != nil {
					m.err = err
				} else {
					// Remove session from state
					st, err := state.Load()
					if err != nil {
						log.Printf("Warning: failed to load state: %v", err)
					} else {
						if err := st.RemoveSession(session.Name); err != nil {
							log.Printf("Warning: failed to remove session from state: %v", err)
						}
					}

					// Remove from session list
					m.sessions = append(m.sessions[:m.cursor], m.sessions[m.cursor+1:]...)
					if m.cursor >= len(m.sessions) && m.cursor > 0 {
						m.cursor--
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case detachedMsg:
		// Returned from attached state
		m.state = stateList
		// Refresh session list after detaching
		sessions, err := tmux.List()
		if err != nil {
			m.err = fmt.Errorf("failed to refresh sessions: %w", err)
		} else {
			m.sessions = sessions
			if m.cursor >= len(m.sessions) {
				m.cursor = len(m.sessions) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		}
	}

	return m, nil
}

func (m Model) updateCreating(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateList
			m.nameInput = ""

		case "enter":
			if m.nameInput != "" {
				session, err := tmux.NewSession(m.nameInput)
				if err != nil {
					m.err = err
					m.state = stateList
				} else {
					m.sessions = append(m.sessions, session)
					m.cursor = len(m.sessions) - 1
					m.state = stateList
				}
				m.nameInput = ""
			}

		case "backspace":
			if len(m.nameInput) > 0 {
				m.nameInput = m.nameInput[:len(m.nameInput)-1]
			}

		default:
			if len(msg.String()) == 1 {
				m.nameInput += msg.String()
			}
		}
	}

	return m, nil
}

type detachedMsg struct{}

func (m Model) View() string {
	switch m.state {
	case stateList:
		return m.viewList()
	case stateCreating:
		return m.viewCreating()
	}
	return ""
}

func (m Model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rocha - Claude Code Session Manager"))
	b.WriteString("\n\n")

	if len(m.sessions) == 0 {
		b.WriteString(normalStyle.Render("No Claude Code sessions yet. Press 'n' to create one."))
	} else {
		for i, session := range m.sessions {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}

			line := fmt.Sprintf("%s %d. %s", cursor, i+1, session.Name)
			if i == m.cursor {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(normalStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", m.err)))
		m.err = nil // Clear error after showing
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("↑/k: up • ↓/j: down • n: new • enter: attach (Ctrl+B D or Ctrl+Q to detach) • x: kill • q: quit"))

	return b.String()
}

func (m Model) viewCreating() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Create New Session"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("Session name: "))
	b.WriteString(selectedStyle.Render(m.nameInput + "█"))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter: create • esc: cancel"))

	return b.String()
}
