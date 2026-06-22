package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/LaoQi/tapass/tui/internal/model"
)

type WelcomeState int

const (
	WelcomeSelect WelcomeState = iota
	WelcomeOpenPath
	WelcomeOpenPassword
	WelcomeNewPath
	WelcomeNewPassword
	WelcomeNewPasswordConfirm
)

type WelcomeModel struct {
	state         WelcomeState
	pathInput     textinput.Model
	passwordInput textinput.Model
	confirmInput  textinput.Model
	initialPath   string
	err           error
	width         int
	height        int
}

func NewWelcomeModel() WelcomeModel {
	pathInput := textinput.New()
	pathInput.Placeholder = "path/to/vault.tap"
	pathInput.CharLimit = 256

	passwordInput := textinput.New()
	passwordInput.Placeholder = "master password"
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.CharLimit = 256

	confirmInput := textinput.New()
	confirmInput.Placeholder = "confirm password"
	confirmInput.EchoMode = textinput.EchoPassword
	confirmInput.CharLimit = 256

	return WelcomeModel{
		state:         WelcomeSelect,
		pathInput:     pathInput,
		passwordInput: passwordInput,
		confirmInput:  confirmInput,
	}
}

func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case initialPathMsg:
		if msg.Path != "" {
			m.initialPath = msg.Path
			m.pathInput.SetValue(msg.Path)
			m.state = WelcomeOpenPassword
			m.passwordInput.Focus()
		}
		return m, nil
	case resizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		m.err = nil
		switch m.state {
		case WelcomeSelect:
			switch msg.String() {
			case "o":
				m.state = WelcomeOpenPath
				m.pathInput.Focus()
				return m, nil
			case "n":
				m.state = WelcomeNewPath
				m.pathInput.Focus()
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			}

		case WelcomeOpenPath:
			switch msg.String() {
			case "enter":
				m.state = WelcomeOpenPassword
				m.pathInput.Blur()
				m.passwordInput.Focus()
				return m, nil
			case "esc":
				m.state = WelcomeSelect
				m.pathInput.Blur()
				return m, nil
			case "tab":
				m.pathInput.SetValue(completePath(m.pathInput.Value()))
				m.pathInput.CursorEnd()
				return m, nil
			}
			m.pathInput, cmd = m.pathInput.Update(msg)
			return m, cmd

		case WelcomeOpenPassword:
			switch msg.String() {
			case "enter":
				db, err := model.OpenDB(m.pathInput.Value(), m.passwordInput.Value())
				if err != nil {
					m.err = err
					m.passwordInput.SetValue("")
					return m, nil
				}
				return m, func() tea.Msg {
					return OpenVaultMsg{DB: db, Path: m.pathInput.Value()}
				}
			case "esc":
				m.state = WelcomeOpenPath
				m.passwordInput.Blur()
				m.pathInput.Focus()
				return m, nil
			}
			m.passwordInput, cmd = m.passwordInput.Update(msg)
			return m, cmd

		case WelcomeNewPath:
			switch msg.String() {
			case "enter":
				m.state = WelcomeNewPassword
				m.pathInput.Blur()
				m.passwordInput.Focus()
				return m, nil
			case "esc":
				m.state = WelcomeSelect
				m.pathInput.Blur()
				return m, nil
			case "tab":
				m.pathInput.SetValue(completePath(m.pathInput.Value()))
				m.pathInput.CursorEnd()
				return m, nil
			}
			m.pathInput, cmd = m.pathInput.Update(msg)
			return m, cmd

		case WelcomeNewPassword:
			switch msg.String() {
			case "enter":
				m.state = WelcomeNewPasswordConfirm
				m.passwordInput.Blur()
				m.confirmInput.Focus()
				return m, nil
			case "esc":
				m.state = WelcomeNewPath
				m.passwordInput.Blur()
				m.pathInput.Focus()
				return m, nil
			}
			m.passwordInput, cmd = m.passwordInput.Update(msg)
			return m, cmd

		case WelcomeNewPasswordConfirm:
			switch msg.String() {
			case "enter":
				if m.passwordInput.Value() != m.confirmInput.Value() {
					m.err = fmt.Errorf("passwords do not match")
					m.confirmInput.SetValue("")
					return m, nil
				}
				db, err := model.CreateDB(m.pathInput.Value(), m.passwordInput.Value())
				if err != nil {
					m.err = err
					return m, nil
				}
				return m, func() tea.Msg {
					return CreateVaultMsg{DB: db, Path: m.pathInput.Value()}
				}
			case "esc":
				m.state = WelcomeNewPassword
				m.confirmInput.Blur()
				m.passwordInput.Focus()
				return m, nil
			}
			m.confirmInput, cmd = m.confirmInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

var tapassASCII = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#7C3AED")).
	Bold(true).
	Render(
		`
████████╗ █████╗ ██████╗  █████╗ ███████╗███████╗
╚══██╔══╝██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔════╝
   ██║   ███████║██████╔╝███████║███████╗███████╗
   ██║   ██╔══██║██╔═══╝ ██╔══██║╚════██║╚════██║
   ██║   ██║  ██║██║     ██║  ██║███████║███████║
   ╚═╝   ╚═╝  ╚═╝╚═╝     ╚═╝  ╚═╝╚══════╝╚══════╝
`)

func (m WelcomeModel) View() tea.View {
	width := m.width
	if width < 1 {
		width = 40
	}
	height := m.height
	if height < 1 {
		height = 20
	}

	inputW := clampInt(width/2, 24, 60)

	var content strings.Builder

	content.WriteString(tapassASCII)
	content.WriteString("\n\n")

	switch m.state {
	case WelcomeSelect:
		content.WriteString(menuStyle.Render("  [o] Open existing vault\n  [n] Create new vault\n  [q] Quit"))

	case WelcomeOpenPath:
		content.WriteString("Enter vault path:\n\n")
		content.WriteString(inputStyle.Width(inputW).Render(m.pathInput.View()))

	case WelcomeOpenPassword:
		content.WriteString(fmt.Sprintf("Opening: %s\n\n", m.pathInput.Value()))
		content.WriteString("Enter master password:\n\n")
		content.WriteString(inputStyle.Width(inputW).Render(m.passwordInput.View()))

	case WelcomeNewPath:
		content.WriteString("Enter path for new vault:\n\n")
		content.WriteString(inputStyle.Width(inputW).Render(m.pathInput.View()))

	case WelcomeNewPassword:
		content.WriteString(fmt.Sprintf("Creating: %s\n\n", m.pathInput.Value()))
		content.WriteString("Enter master password:\n\n")
		content.WriteString(inputStyle.Width(inputW).Render(m.passwordInput.View()))

	case WelcomeNewPasswordConfirm:
		content.WriteString("Confirm master password:\n\n")
		content.WriteString(inputStyle.Width(inputW).Render(m.confirmInput.View()))
	}

	if m.err != nil {
		content.WriteString("\n\n")
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	titleView := lipgloss.NewStyle().
		Width(width).
		Padding(1, 0).
		Render(titleStyle.Render(""))

	contentStr := content.String()
	contentLines := strings.Count(contentStr, "\n") + 1
	contentMaxW := 0
	for _, line := range strings.Split(contentStr, "\n") {
		if lipgloss.Width(line) > contentMaxW {
			contentMaxW = lipgloss.Width(line)
		}
	}

	availH := height - 4
	padTop := (availH - contentLines) / 2
	if padTop < 0 {
		padTop = 0
	}
	padLeft := (width - contentMaxW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	centerView := lipgloss.NewStyle().
		Width(width).
		Height(availH).
		Padding(padTop, 0, 0, padLeft).
		Render(contentStr)

	hint := "[esc] back"
	if m.state == WelcomeSelect {
		hint = "[o] open  [n] new  [q] quit"
	}
	statusView := statusBarStyle.Width(width).Render(hint)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Top, titleView, centerView, statusView))
}
