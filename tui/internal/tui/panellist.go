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

type panelMode int

const (
	modeGroup panelMode = iota
	modeAttr
)

type PanelListModel struct {
	db      *model.DB
	prefix  string
	mode    panelMode
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

func NewPanelListModel(db *model.DB, prefix string, mode panelMode) PanelListModel {
	m := PanelListModel{
		db:     db,
		prefix: prefix,
		mode:   mode,
	}
	m.deriveTitle()
	m.doQuery(true)
	return m
}

func (m *PanelListModel) deriveTitle() {
	if m.mode == modeAttr {
		if m.prefix == "" {
			m.title = ""
			return
		}
		if idx := strings.LastIndex(m.prefix, "/"); idx >= 0 {
			m.title = m.prefix[idx+1:]
		} else {
			m.title = m.prefix
		}
		return
	}
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

	switch m.mode {
	case modeGroup:
		m.items = m.queryGroupItems()
	case modeAttr:
		m.items = m.queryAttrItems()
	default:
		m.items = nil
	}

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

func (m *PanelListModel) queryGroupItems() []model.ListItem {
	keys := m.db.QueryKeys(m.prefix)
	seen := make(map[string]bool)
	items := make([]model.ListItem, 0)

	for _, key := range keys {
		rest := strings.TrimPrefix(key, m.prefix+"/")
		name := rest
		if idx := strings.Index(rest, "/"); idx >= 0 {
			name = rest[:idx]
		}
		fullPath := m.prefix + "/" + name
		if seen[fullPath] {
			continue
		}
		seen[fullPath] = true
		isEntry := m.db.HasChildEntries(fullPath)
		items = append(items, model.ListItem{
			Name:     name,
			FullPath: fullPath,
			IsEntry:  isEntry,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IsEntry != items[j].IsEntry {
			return !items[i].IsEntry
		}
		return items[i].Name < items[j].Name
	})

	return items
}

func (m *PanelListModel) queryAttrItems() []model.ListItem {
	keys := m.db.QueryKeys(m.prefix)
	items := make([]model.ListItem, 0)
	for _, key := range keys {
		rest := strings.TrimPrefix(key, m.prefix+"/")
		if rest == "" || strings.Contains(rest, "/") {
			continue
		}
		items = append(items, model.ListItem{
			Name:     rest,
			FullPath: key,
			IsAttr:   true,
		})
	}
	sort.Slice(items, func(i, j int) bool {
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

func (m *PanelListModel) SetMode(mode panelMode) {
	m.mode = mode
	m.deriveTitle()
	m.doQuery(true)
}

func (m *PanelListModel) Prefix() string {
	return m.prefix
}

func (m *PanelListModel) Mode() panelMode {
	return m.mode
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
	// lipgloss border Width(w) 实际可用为 w-2（2列内部开销），因此标题渲染宽度需减2
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
		if item.IsAttr {
			icon = "🔖 "
		} else if item.IsEntry {
			icon = "📄 "
		}

		label := item.Name
		// lipgloss border 设置 Width(w-2) 时实际可用内容宽度为 w-4（2列边框+2列内部开销），
		// 因此 label 最大宽度 = (w-4) - iconDisplayWidth = w - 5 - iconDisplayWidth
		maxLen := width - 5 - iconDisplayWidth
		if maxLen > 0 && runewidth.StringWidth(label) > maxLen {
			label = truncateString(label, maxLen-1) + "…"
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
