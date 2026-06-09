package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
	"github.com/tapass/tapass-tui/internal/model"
)

const iconDisplayWidth = 3

type PanelListModel struct {
	db      *model.DB
	prefix  string
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

	panelAttrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399"))

	panelSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)
)

func truncateString(s string, maxWidth int) string {
	var b strings.Builder
	cur := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if cur+rw > maxWidth {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	return b.String()
}

func NewPanelListModel(db *model.DB, prefix string) PanelListModel {
	m := PanelListModel{
		db:     db,
		prefix: prefix,
	}
	m.deriveTitle()
	m.doQuery(true)
	return m
}

func (m *PanelListModel) deriveTitle() {
	if m.prefix == "" {
		m.title = "/"
	} else {
		m.title = m.prefix
	}
}

func (m *PanelListModel) doQuery(resetCursor bool) {
	if m.db == nil {
		m.items = nil
		m.cursor = 0
		return
	}

	var savedPath string
	if !resetCursor && m.cursor < len(m.items) {
		savedPath = m.items[m.cursor].FullPath
	}

	m.items = m.queryItems()

	if resetCursor || savedPath == "" {
		m.cursor = 0
		if m.cursor >= len(m.items) {
			m.cursor = 0
		}
		return
	}

	m.cursor = 0
	for i, item := range m.items {
		if item.FullPath == savedPath {
			m.cursor = i
			break
		}
	}
}

func (m *PanelListModel) queryItems() []model.ListItem {
	keys := m.db.QueryKeys(m.prefix)
	depthMap := make(map[string]int)
	orderMap := make(map[string]int)
	idx := 0

	for _, key := range keys {
		rest := strings.TrimPrefix(key, m.prefix+"/")
		name := rest
		depth := 0
		if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
			name = rest[:slashIdx]
			subRest := rest[slashIdx+1:]
			depth = strings.Count(subRest, "/") + 1
		}
		fullPath := m.prefix + "/" + name
		if _, exists := depthMap[fullPath]; !exists {
			depthMap[fullPath] = depth
			orderMap[fullPath] = idx
			idx++
		} else if depth > depthMap[fullPath] {
			depthMap[fullPath] = depth
		}
	}

	items := make([]model.ListItem, 0, len(depthMap))
	for fullPath, depth := range depthMap {
		name := fullPath[strings.LastIndex(fullPath, "/")+1:]
		items = append(items, model.ListItem{
			Name:     name,
			FullPath: fullPath,
			Depth:    depth,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if (items[i].Depth > 0) != (items[j].Depth > 0) {
			return items[i].Depth > 0
		}
		return items[i].Name < items[j].Name
	})

	return items
}

func (m *PanelListModel) SetDB(db *model.DB) {
	m.db = db
	m.doQuery(true)
}

func (m *PanelListModel) SetPrefix(prefix string) {
	m.prefix = prefix
	m.deriveTitle()
	m.doQuery(true)
}

func (m *PanelListModel) Prefix() string {
	return m.prefix
}

func (m *PanelListModel) Refresh() {
	m.doQuery(false)
}

func (m *PanelListModel) HandleEvent(evt model.Event) {
	switch evt.Type {
	case model.EventAttrSet, model.EventAttrDeleted:
		if m.prefix == "" {
			return
		}
		if strings.HasPrefix(evt.Key, m.prefix+"/") {
			m.doQuery(false)
		}
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
	contentWidth := width - 2
	maxTitleWidth := contentWidth - 1
	if runewidth.StringWidth(titleLine) > contentWidth {
		titleLine = truncateString(titleLine, maxTitleWidth) + "…"
	}
	b.WriteString(titleStyle.Width(contentWidth - 2).Render(titleLine))
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

		icon := "📂 "
		if item.Depth == 0 {
			icon = "🔖 "
		}

		label := item.Name
		maxLen := width - 5 - iconDisplayWidth
		if maxLen > 0 && runewidth.StringWidth(label) > maxLen {
			label = truncateString(label, maxLen-1) + "…"
		}

		line := icon + label

		if i == m.cursor {
			b.WriteString(panelSelectedStyle.Render(line))
		} else if item.Depth > 0 {
			b.WriteString(panelGroupStyle.Render(line))
		} else {
			b.WriteString(panelAttrStyle.Render(line))
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

func (m *PanelListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *PanelListModel) SetFocused(f bool) {
	m.focused = f
}
