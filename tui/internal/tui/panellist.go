package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
)

type PanelListModel struct {
	items   []model.ListItem
	cursor  int
	width   int
	height  int
	title   string
	focused bool
}

var (
	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#6B7280")).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#374151"))

	panelTitleFocusStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF8C00")).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("#FF8C00"))

	panelGroupStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA"))

	panelEntryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	panelAttrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399"))

	panelSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)
)

func NewPanelListModel(title string, items []model.ListItem) PanelListModel {
	return PanelListModel{
		title:  title,
		items:  items,
		cursor: 0,
	}
}

func (m PanelListModel) Init() {}

func (m PanelListModel) View() string {
	width := m.width
	if width < 1 {
		width = 20
	}
	height := m.height
	if height < 1 {
		height = 20
	}

	var b strings.Builder

	titleStyle := panelTitleStyle
	if m.focused {
		titleStyle = panelTitleFocusStyle
	}

	titleLine := fmt.Sprintf(" %s", m.title)
	if len(titleLine) > width-2 {
		titleLine = titleLine[:width-3] + "…"
	}
	b.WriteString(titleStyle.Width(width - 2).Render(titleLine))
	b.WriteString("\n")

	if len(m.items) == 0 {
		b.WriteString("\n  (empty)")
		content := b.String()
		return m.wrapBorder(content, width, height)
	}

	displayHeight := height - 4
	if displayHeight < 1 {
		displayHeight = 1
	}

	start := 0
	if m.cursor >= displayHeight {
		start = m.cursor - displayHeight + 1
	}
	end := start + displayHeight
	if end > len(m.items) {
		end = len(m.items)
	}

	for i := start; i < end; i++ {
		item := m.items[i]

		icon := "📁 "
		if item.IsAttr {
			icon = "🏷️ "
		} else if item.IsEntry {
			icon = "📝 "
		}

		label := item.Name
		maxLen := width - len(icon) - 3
		if maxLen > 0 && len(label) > maxLen {
			label = label[:maxLen-1] + "…"
		}

		line := icon + label

		if i == m.cursor {
			b.WriteString(panelSelectedStyle.Render(line))
		} else if item.IsAttr {
			b.WriteString(panelAttrStyle.Render(line))
		} else if item.IsEntry {
			b.WriteString(panelEntryStyle.Render(line))
		} else {
			b.WriteString(panelGroupStyle.Render(line))
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return m.wrapBorder(b.String(), width, height)
}

func (m PanelListModel) wrapBorder(content string, width, height int) string {
	style := blurBorderStyle
	if m.focused {
		style = focusBorderStyle
	}
	return style.Width(width - 2).Height(height - 2).Render(content)
}

func (m PanelListModel) SelectedItem() model.ListItem {
	if m.cursor < len(m.items) {
		return m.items[m.cursor]
	}
	return model.ListItem{}
}

func (m *PanelListModel) SetItems(items []model.ListItem) {
	m.items = items
	if m.cursor >= len(m.items) {
		m.cursor = 0
	}
}

func (m *PanelListModel) SetTitle(title string) {
	m.title = title
}

func (m *PanelListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *PanelListModel) SetFocused(f bool) {
	m.focused = f
}

func (m PanelListModel) ItemCount() int {
	return len(m.items)
}

func (m *PanelListModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *PanelListModel) MoveDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
	}
}
