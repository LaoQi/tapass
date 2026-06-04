package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
)

type EntryListModel struct {
	group   *model.Node
	entries []*model.Node
	cursor  int
	width   int
	height  int
}

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#6B7280")).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#374151"))

	rowSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))
)

func NewEntryListModel(group *model.Node) EntryListModel {
	m := EntryListModel{
		group: group,
	}
	if group != nil {
		m.entries = group.DirectEntries()
	}
	return m
}

func (m EntryListModel) Init() tea.Cmd {
	return nil
}

func (m EntryListModel) Update(msg tea.Msg) (EntryListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				groupPath := ""
				if m.group != nil {
					groupPath = m.group.Path
				}
				return m, func() tea.Msg {
					return SelectEntryMsg{Node: entry, GroupPath: groupPath}
				}
			}
		}
	}
	return m, nil
}

func (m EntryListModel) View() string {
	width := m.width
	if width < 1 {
		width = 40
	}
	height := m.height
	if height < 1 {
		height = 20
	}

	if m.group == nil {
		return lipgloss.NewStyle().Width(width).Height(height).Render("Select a group from the sidebar")
	}

	idWidth := width / 3
	attrWidth := width - idWidth
	if idWidth < 8 {
		idWidth = 8
	}
	if attrWidth < 8 {
		attrWidth = 8
	}

	var b strings.Builder

	header := fmt.Sprintf("%-*s%-*s", idWidth, "ID", attrWidth, "Attributes")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	if len(m.entries) == 0 {
		b.WriteString("\n  No entries in this group")
		return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
	}

	displayHeight := height - 2
	if displayHeight < 1 {
		displayHeight = 1
	}

	for i, entry := range m.entries {
		if i >= displayHeight {
			break
		}

		attrNames := make([]string, 0, len(entry.Attrs))
		for k := range entry.Attrs {
			attrNames = append(attrNames, k)
		}
		attrStr := strings.Join(attrNames, ", ")

		id := entry.ID
		if len(id) > idWidth-1 {
			id = id[:idWidth-2] + "…"
		}
		if len(attrStr) > attrWidth-1 {
			attrStr = attrStr[:attrWidth-2] + "…"
		}

		line := fmt.Sprintf("%-*s%-*s", idWidth, id, attrWidth, attrStr)

		if i == m.cursor {
			b.WriteString(rowSelectedStyle.Render(line))
		} else {
			b.WriteString(rowStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

func (m EntryListModel) SelectedEntry() *model.Node {
	if m.cursor < len(m.entries) {
		return m.entries[m.cursor]
	}
	return nil
}

func (m *EntryListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
