package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/tapass/tapass-tui/internal/model"
)

type dbConfigState int

const (
	dbConfigMenu dbConfigState = iota
	dbConfigChangeOldPassword
	dbConfigChangeNewPassword
	dbConfigChangeNewPasswordConfirm
)

type DBConfigModel struct {
	db             *model.DB
	dbPath         string
	state          dbConfigState
	oldPassword    textinput.Model
	newPassword    textinput.Model
	confirmPassword textinput.Model
	err            error
	success        bool
	width          int
	height         int
}

func NewDBConfigModel(db *model.DB, dbPath string) DBConfigModel {
	oldPassword := textinput.New()
	oldPassword.EchoMode = textinput.EchoPassword
	oldPassword.Placeholder = "current password"
	oldPassword.CharLimit = 256

	newPassword := textinput.New()
	newPassword.EchoMode = textinput.EchoPassword
	newPassword.Placeholder = "new password"
	newPassword.CharLimit = 256

	confirmPassword := textinput.New()
	confirmPassword.EchoMode = textinput.EchoPassword
	confirmPassword.Placeholder = "confirm new password"
	confirmPassword.CharLimit = 256

	return DBConfigModel{
		db:             db,
		dbPath:         dbPath,
		state:          dbConfigMenu,
		oldPassword:    oldPassword,
		newPassword:    newPassword,
		confirmPassword: confirmPassword,
	}
}

func (m DBConfigModel) Init() tea.Cmd {
	return nil
}

func (m DBConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case resizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		m.err = nil
		m.success = false
		switch m.state {
		case dbConfigMenu:
			switch msg.String() {
			case "p":
				m.state = dbConfigChangeOldPassword
				m.oldPassword.Focus()
				return m, nil
			case "esc":
				return m, func() tea.Msg { return BackToMainMsg{} }
			}

		case dbConfigChangeOldPassword:
			switch msg.String() {
			case "enter":
				m.state = dbConfigChangeNewPassword
				m.oldPassword.Blur()
				m.newPassword.Focus()
				return m, nil
			case "esc":
				m.state = dbConfigMenu
				m.oldPassword.Blur()
				return m, nil
			}
			m.oldPassword, cmd = m.oldPassword.Update(msg)
			return m, cmd

		case dbConfigChangeNewPassword:
			switch msg.String() {
			case "enter":
				m.state = dbConfigChangeNewPasswordConfirm
				m.newPassword.Blur()
				m.confirmPassword.Focus()
				return m, nil
			case "esc":
				m.state = dbConfigChangeOldPassword
				m.newPassword.Blur()
				m.oldPassword.Focus()
				return m, nil
			}
			m.newPassword, cmd = m.newPassword.Update(msg)
			return m, cmd

		case dbConfigChangeNewPasswordConfirm:
			switch msg.String() {
			case "enter":
				if m.newPassword.Value() != m.confirmPassword.Value() {
					m.err = fmt.Errorf("passwords do not match")
					m.confirmPassword.SetValue("")
					return m, nil
				}
				cmds, err := m.db.ChangePassword(m.oldPassword.Value(), m.newPassword.Value())
				if err != nil {
					m.err = err
					m.oldPassword.SetValue("")
					m.newPassword.SetValue("")
					m.confirmPassword.SetValue("")
					m.state = dbConfigMenu
					return m, nil
				}
				m.success = true
				m.state = dbConfigMenu
				m.oldPassword.SetValue("")
				m.newPassword.SetValue("")
				m.confirmPassword.SetValue("")
				return m, tea.Batch(append([]tea.Cmd{func() tea.Msg { return PasswordChangedMsg{} }}, cmds...)...)
			case "esc":
				m.state = dbConfigChangeNewPassword
				m.confirmPassword.Blur()
				m.newPassword.Focus()
				return m, nil
			}
			m.confirmPassword, cmd = m.confirmPassword.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m DBConfigModel) View() tea.View {
	width := m.width
	if width < 1 {
		width = 40
	}
	height := m.height
	if height < 1 {
		height = 20
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render("Database Settings"))
	b.WriteString("\n\n")

	inputW := width - 4
	if inputW < 20 {
		inputW = 20
	}

	hdr := m.db.Config()

	switch m.state {
	case dbConfigMenu:
		argon2 := hdr.Argon2
		b.WriteString(fmt.Sprintf("Path: %s\n", m.dbPath))
		b.WriteString(fmt.Sprintf("Argon2id: time=%d memory=%d parallelism=%d\n",
			argon2.TimeCost, argon2.MemoryCost, argon2.Parallelism))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[p] change password  [esc] back\n"))

		if m.success {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("Password changed successfully!"))
		}

	case dbConfigChangeOldPassword:
		b.WriteString("Enter current master password:\n")
		b.WriteString(inputStyle.Width(inputW).Render(m.oldPassword.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[enter] next  [esc] cancel"))

	case dbConfigChangeNewPassword:
		b.WriteString("Enter new master password:\n")
		b.WriteString(inputStyle.Width(inputW).Render(m.newPassword.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[enter] next  [esc] back"))

	case dbConfigChangeNewPasswordConfirm:
		b.WriteString("Confirm new master password:\n")
		b.WriteString(inputStyle.Width(inputW).Render(m.confirmPassword.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[enter] confirm  [esc] back"))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return tea.NewView(lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(b.String()))
}


