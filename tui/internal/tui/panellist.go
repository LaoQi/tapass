package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
	"github.com/tapass/tapass-tui/internal/model"
)

const iconDisplayWidth = 3

type PanelListModel struct {
	db          *model.DB
	prefix      string
	rawKeys     []string
	items       []model.ListItem
	cursor      int
	width       int
	height      int
	title       string
	focused     bool
	searchMode  bool
	searchInput textinput.Model
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

func NewPanelListModel(db *model.DB, prefix string) PanelListModel {
	si := textinput.New()
	si.Prompt = "/"
	//si.Placeholder = "搜索..."
	si.CharLimit = 64
	m := PanelListModel{
		db:          db,
		prefix:      prefix,
		searchInput: si,
	}
	m = m.deriveTitle()
	m = m.doQuery(true)
	return m
}

func (m PanelListModel) deriveTitle() PanelListModel {
	if m.prefix == "" {
		m.title = "/"
	} else {
		m.title = m.prefix
	}
	return m
}

func (m PanelListModel) doQuery(resetCursor bool) PanelListModel {
	if m.db == nil {
		m.rawKeys = nil
		m.items = nil
		m.cursor = 0
		return m
	}

	var savedPath string
	if !resetCursor && m.cursor < len(m.items) {
		savedPath = m.items[m.cursor].FullPath
	}

	m.rawKeys = m.db.QueryKeys(m.prefix)
	m = m.rebuildItems()

	if resetCursor || savedPath == "" {
		m.cursor = 0
		if m.cursor >= len(m.items) {
			m.cursor = 0
		}
		return m
	}

	m.cursor = 0
	for i, item := range m.items {
		if item.FullPath == savedPath {
			m.cursor = i
			break
		}
	}
	return m
}

func (m PanelListModel) buildItems(keys []string) []model.ListItem {
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
		return orderMap[items[i].FullPath] < orderMap[items[j].FullPath]
	})

	return items
}

func (m PanelListModel) rebuildItems() PanelListModel {
	keys := m.rawKeys
	query := m.searchInput.Value()
	if m.searchMode && query != "" {
		lowerQuery := strings.ToLower(query)
		filtered := make([]string, 0, len(keys))
		for _, key := range keys {
			if strings.Contains(strings.ToLower(key), lowerQuery) {
				filtered = append(filtered, key)
			}
		}
		keys = filtered
	}
	m.items = m.buildItems(keys)
	if m.cursor >= len(m.items) {
		m.cursor = 0
	}
	return m
}

func (m PanelListModel) Prefix() string {
	return m.prefix
}

func (m PanelListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case searchEnterMsg:
		m.searchMode = true
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		m = m.rebuildItems()
		return m, nil
	case searchExitMsg:
		m.searchMode = false
		m.searchInput.Blur()
		m = m.rebuildItems()
		return m, nil
	case searchFocusMsg:
		m.searchInput.Focus()
		return m, nil
	case searchBlurMsg:
		m.searchInput.Blur()
		return m, nil
	case moveUpMsg:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case moveDownMsg:
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
		return m, nil
	case setPrefixMsg:
		m.prefix = msg.Prefix
		m = m.deriveTitle()
		m = m.doQuery(true)
		return m, nil
	case refreshMsg:
		m = m.doQuery(false)
		return m, nil
	case dbEventMsg:
		switch msg.Event.Type {
		case model.EventAttrSet, model.EventAttrDeleted:
			if strings.HasPrefix(msg.Event.Key, m.prefix+"/") {
				m = m.doQuery(false)
			}
		}
		return m, nil
	case resizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case setFocusMsg:
		m.focused = msg.Focused
		return m, nil
	}

	if m.searchMode && m.searchInput.Focused() {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch keyMsg.String() {
			case "esc", "enter":
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m = m.rebuildItems()
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m PanelListModel) Init() tea.Cmd { return nil }

func (m PanelListModel) View() tea.View {
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

	if m.searchMode {
		borderColor := lipgloss.Color("#7C3AED")
		if !m.searchInput.Focused() {
			borderColor = lipgloss.Color("#374151")
		}
		searchStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(borderColor)
		b.WriteString(searchStyle.Width(contentWidth - 2).Render(" " + m.searchInput.View()))
		b.WriteString("\n")
	}

	if len(m.items) == 0 {
		if m.searchMode {
			b.WriteString("\n  (无匹配)")
		} else {
			b.WriteString("\n  (empty)")
		}
		content := b.String()
		return tea.NewView(m.wrapBorder(content, width, height))
	}

	displayHeight := height - 4
	if m.searchMode {
		displayHeight = height - 5
	}
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
		if m.searchMode && item.Depth > 0 {
			label = item.FullPath
		}
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

	return tea.NewView(m.wrapBorder(b.String(), width, height))
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


