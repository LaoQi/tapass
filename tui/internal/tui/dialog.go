package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
)

type NewEntryDialogModel struct {
	idInput       textinput.Model
	pathInput     textinput.Model
	currentPrefix string
	step          int
	err           error
	width         int
	height        int
}

func NewNewEntryDialog(currentPrefix string) NewEntryDialogModel {
	pathInput := textinput.New()
	pathInput.Placeholder = "group/subgroup"
	pathInput.CharLimit = 256
	pathInput.SetValue(currentPrefix)

	idInput := textinput.New()
	idInput.Placeholder = "entry key (e.g. github)"
	idInput.CharLimit = 256
	idInput.Focus()

	return NewEntryDialogModel{
		pathInput:     pathInput,
		idInput:       idInput,
		currentPrefix: currentPrefix,
		step:          0,
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
		switch m.step {
		case 0:
			switch msg.String() {
			case "enter":
				if m.idInput.Value() == "" {
					m.err = fmt.Errorf("key cannot be empty")
					return m, nil
				}
				groupPath := m.pathInput.Value()
				entryPathPrefix := model.EntryPath(groupPath, m.idInput.Value())
				return m, func() tea.Msg {
					return NewEntryCreatedMsg{
						EntryPathPrefix: entryPathPrefix,
						GroupPath:       groupPath,
					}
				}
			case "tab":
				m.step = 1
				m.idInput.Blur()
				m.pathInput.Focus()
				return m, nil
			case "esc":
				return m, func() tea.Msg { return BackToMainMsg{} }
			}
			m.idInput, cmd = m.idInput.Update(msg)
			return m, cmd

		case 1:
			switch msg.String() {
			case "enter":
				if m.idInput.Value() == "" {
					m.err = fmt.Errorf("key cannot be empty")
					return m, nil
				}
				groupPath := m.pathInput.Value()
				entryPathPrefix := model.EntryPath(groupPath, m.idInput.Value())
				return m, func() tea.Msg {
					return NewEntryCreatedMsg{
						EntryPathPrefix: entryPathPrefix,
						GroupPath:       groupPath,
					}
				}
			case "tab":
				m.step = 0
				m.pathInput.Blur()
				m.idInput.Focus()
				return m, nil
			case "esc":
				m.step = 0
				m.pathInput.Blur()
				m.idInput.Focus()
				return m, nil
			}
			m.pathInput, cmd = m.pathInput.Update(msg)
			return m, cmd
		}
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

	inputW := width - 4
	if inputW < 20 {
		inputW = 20
	}

	var b strings.Builder
	b.WriteString("New entry:\n\n")

	b.WriteString("Group path:\n")
	pathStyle := inputStyle.Width(inputW)
	if m.step == 1 {
		pathStyle = pathStyle.BorderForeground(lipgloss.Color("#7C3AED"))
	}
	b.WriteString(pathStyle.Render(m.pathInput.View()))
	b.WriteString("\n\n")

	b.WriteString("Entry key:\n")
	idStyle := inputStyle.Width(inputW)
	if m.step == 0 {
		idStyle = idStyle.BorderForeground(lipgloss.Color("#7C3AED"))
	}
	b.WriteString(idStyle.Render(m.idInput.View()))
	b.WriteString("\n\n")

	b.WriteString(menuStyle.Render("[Tab] switch field  [enter] create  [esc] cancel"))

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
