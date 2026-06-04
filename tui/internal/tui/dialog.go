package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
	"github.com/tapass/tapass-tools/vault"
)

type NewEntryDialogModel struct {
	idInput   textinput.Model
	groupPath string
	v         *vault.Vault
	err       error
	width     int
	height    int
}

func NewNewEntryDialog(groupPath string, v *vault.Vault) NewEntryDialogModel {
	idInput := textinput.New()
	idInput.Placeholder = "entry key (e.g. github)"
	idInput.CharLimit = 256
	idInput.Focus()

	return NewEntryDialogModel{
		idInput:   idInput,
		groupPath: groupPath,
		v:         v,
	}
}

func (m NewEntryDialogModel) Init() tea.Cmd {
	return nil
}

func (m NewEntryDialogModel) Update(msg tea.Msg) (NewEntryDialogModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		m.err = nil
		switch msg.String() {
		case "enter":
			if m.idInput.Value() == "" {
				m.err = fmt.Errorf("key cannot be empty")
				return m, nil
			}
			entryPathPrefix := model.EntryPath(m.groupPath, m.idInput.Value())
			return m, func() tea.Msg {
				return NewEntryCreatedMsg{
					EntryPathPrefix: entryPathPrefix,
					GroupPath:        m.groupPath,
				}
			}
		case "esc":
			return m, func() tea.Msg { return BackToMainMsg{} }
		}
		m.idInput, cmd = m.idInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m NewEntryDialogModel) View() string {
	width := m.width
	if width < 1 {
		width = 40
	}
	height := m.height
	if height < 1 {
		height = 10
	}

	var b strings.Builder
	b.WriteString("New entry key identifier:\n")
	b.WriteString("This will be used as the path ID. You can add attributes in the detail view.\n\n")
	b.WriteString(inputStyle.Width(width - 4).Render(m.idInput.View()))
	b.WriteString("\n[enter] create & edit  [esc] cancel")

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(b.String())
}

func (m *NewEntryDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
