package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
	"github.com/tapass/tapass-tools/vault"
)

type detailState int

const (
	detailView detailState = iota
	detailEditAttr
	detailAddAttr
)

type EntryDetailModel struct {
	entry       *model.Node
	groupPath   string
	v           *vault.Vault
	state       detailState
	attrKeys    []string
	cursor      int
	editInput   textinput.Model
	editKey     string
	newKeyInput textinput.Model
	err         error
	width       int
	height      int
}

var (
	attrKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")).
			Width(16)

	attrValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))
)

func NewEntryDetailModel(entry *model.Node, groupPath string, v *vault.Vault) EntryDetailModel {
	editInput := textinput.New()
	editInput.CharLimit = 4096

	newKeyInput := textinput.New()
	newKeyInput.Placeholder = "attribute name"
	newKeyInput.CharLimit = 256

	m := EntryDetailModel{
		entry:       entry,
		groupPath:   groupPath,
		v:           v,
		state:       detailView,
		editInput:   editInput,
		newKeyInput: newKeyInput,
	}
	m.refreshAttrs()
	return m
}

func (m EntryDetailModel) Init() tea.Cmd {
	return nil
}

func (m EntryDetailModel) Update(msg tea.Msg) (EntryDetailModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		m.err = nil
		switch m.state {
		case detailView:
			switch msg.String() {
			case "esc":
				return m, func() tea.Msg { return BackToMainMsg{} }
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.attrKeys)-1 {
					m.cursor++
				}
			case "e":
				if m.cursor < len(m.attrKeys) {
					m.state = detailEditAttr
					m.editKey = m.attrKeys[m.cursor]
					m.editInput.SetValue(string(m.entry.Attrs[m.editKey]))
					m.editInput.Focus()
					return m, nil
				}
			case "a":
				m.state = detailAddAttr
				m.newKeyInput.SetValue("")
				m.newKeyInput.Focus()
				return m, nil
			case "d":
				if m.cursor < len(m.attrKeys) && m.v != nil {
					key := m.attrKeys[m.cursor]
					fullKey := m.entry.Path + "/" + key
					m.v.Delete(fullKey)
					delete(m.entry.Attrs, key)
					m.refreshAttrs()
				}
			}

		case detailEditAttr:
			switch msg.String() {
			case "enter":
				if m.v != nil {
					fullKey := m.entry.Path + "/" + m.editKey
					m.v.Set(fullKey, []byte(m.editInput.Value()))
					m.entry.Attrs[m.editKey] = []byte(m.editInput.Value())
					m.refreshAttrs()
				}
				m.state = detailView
				m.editInput.Blur()
				return m, nil
			case "esc":
				m.state = detailView
				m.editInput.Blur()
				return m, nil
			}
			m.editInput, cmd = m.editInput.Update(msg)
			return m, cmd

		case detailAddAttr:
			switch msg.String() {
			case "enter":
				key := m.newKeyInput.Value()
				if key == "" {
					m.err = fmt.Errorf("attribute name cannot be empty")
					return m, nil
				}
				m.state = detailEditAttr
				m.editKey = key
				m.editInput.SetValue("")
				m.editInput.Focus()
				return m, nil
			case "esc":
				m.state = detailView
				m.newKeyInput.Blur()
				return m, nil
			}
			m.newKeyInput, cmd = m.newKeyInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m EntryDetailModel) View() string {
	width := m.width
	if width < 1 {
		width = 40
	}
	height := m.height
	if height < 1 {
		height = 20
	}

	var b strings.Builder

	id := "Unknown"
	if m.entry != nil {
		id = m.entry.ID
	}
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Entry: %s", id)))
	b.WriteString("\n\n")

	switch m.state {
	case detailView:
		maxLines := height - 4
		if maxLines < 1 {
			maxLines = 1
		}
		for i, key := range m.attrKeys {
			if i >= maxLines {
				b.WriteString(menuStyle.Render(fmt.Sprintf("  ... %d more", len(m.attrKeys)-maxLines)))
				break
			}
			value := string(m.entry.Attrs[key])
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			maxValLen := width - 16 - len(prefix) - 2
			if maxValLen > 0 && len(value) > maxValLen {
				value = value[:maxValLen-1] + "…"
			}
			line := fmt.Sprintf("%s%s: %s", prefix, attrKeyStyle.Render(key), attrValueStyle.Render(value))
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[e] edit  [a] add attr  [d] delete attr  [esc] back"))

	case detailEditAttr:
		b.WriteString(fmt.Sprintf("Editing: %s\n", m.editKey))
		b.WriteString(inputStyle.Width(width - 4).Render(m.editInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[enter] save  [esc] cancel"))

	case detailAddAttr:
		b.WriteString("New attribute name:\n")
		b.WriteString(inputStyle.Width(width - 4).Render(m.newKeyInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[enter] next  [esc] cancel"))
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

func (m *EntryDetailModel) refreshAttrs() {
	m.attrKeys = make([]string, 0, len(m.entry.Attrs))
	for k := range m.entry.Attrs {
		m.attrKeys = append(m.attrKeys, k)
	}
	sort.Strings(m.attrKeys)
	if m.cursor >= len(m.attrKeys) {
		m.cursor = 0
	}
}

func (m *EntryDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
