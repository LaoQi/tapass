package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/store"
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
	store         store.Store
	pathInput     textinput.Model
	passwordInput textinput.Model
	confirmInput  textinput.Model
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

func (m WelcomeModel) Update(msg tea.Msg) (WelcomeModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
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
			}
			m.pathInput, cmd = m.pathInput.Update(msg)
			return m, cmd

		case WelcomeOpenPassword:
			switch msg.String() {
			case "enter":
				v, err := m.store.Open(m.pathInput.Value(), m.passwordInput.Value())
				if err != nil {
					m.err = err
					m.passwordInput.SetValue("")
					return m, nil
				}
				return m, func() tea.Msg {
					return OpenVaultMsg{Vault: v, Path: m.pathInput.Value()}
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
				if err := m.store.Create(m.pathInput.Value(), m.passwordInput.Value()); err != nil {
					m.err = err
					return m, nil
				}
				v, err := m.store.Open(m.pathInput.Value(), m.passwordInput.Value())
				if err != nil {
					m.err = err
					return m, nil
				}
				return m, func() tea.Msg {
					return CreateVaultMsg{Vault: v, Path: m.pathInput.Value()}
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

func (m WelcomeModel) View() string {
	width := m.width
	if width < 1 {
		width = 40
	}
	height := m.height
	if height < 1 {
		height = 20
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("tapass-tui"))
	b.WriteString("\n\n")

	inputW := width - 4
	if inputW < 20 {
		inputW = 20
	}

	switch m.state {
	case WelcomeSelect:
		b.WriteString(menuStyle.Render("  [o] Open existing vault\n"))
		b.WriteString(menuStyle.Render("  [n] Create new vault\n"))
		b.WriteString(menuStyle.Render("  [q] Quit\n"))

	case WelcomeOpenPath:
		b.WriteString("Enter vault path:\n")
		b.WriteString(inputStyle.Width(inputW).Render(m.pathInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[esc] back"))

	case WelcomeOpenPassword:
		b.WriteString(fmt.Sprintf("Opening: %s\n", m.pathInput.Value()))
		b.WriteString("Enter master password:\n")
		b.WriteString(inputStyle.Width(inputW).Render(m.passwordInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[esc] back"))

	case WelcomeNewPath:
		b.WriteString("Enter path for new vault:\n")
		b.WriteString(inputStyle.Width(inputW).Render(m.pathInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[esc] back"))

	case WelcomeNewPassword:
		b.WriteString(fmt.Sprintf("Creating: %s\n", m.pathInput.Value()))
		b.WriteString("Enter master password:\n")
		b.WriteString(inputStyle.Width(inputW).Render(m.passwordInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[esc] back"))

	case WelcomeNewPasswordConfirm:
		b.WriteString("Confirm master password:\n")
		b.WriteString(inputStyle.Width(inputW).Render(m.confirmInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[esc] back"))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(b.String())
}

func (m *WelcomeModel) SetStore(s store.Store) {
	m.store = s
}

func (m *WelcomeModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
