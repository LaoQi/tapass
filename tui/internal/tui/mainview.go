package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
)

type MainViewModel struct {
	db            *model.DB
	dbPath        string
	currentPrefix string
	selectedEntry string

	leftPanel   PanelListModel
	middlePanel PanelListModel
	rightPanel  EntryDetailModel

	focus      mainFocus
	totpActive bool
	dirty      bool
	pendingQuit bool

	width  int
	height int
}

type mainFocus int

const (
	focusLeft mainFocus = iota
	focusMiddle
	focusRight
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")).
			Bold(true)

	breadcrumbDirtyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EF4444")).
				Bold(true)
)

func isDetailKey(key string) bool {
	switch key {
	case "e", "a", "d":
		return true
	}
	return false
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func NewMainViewModel(db *model.DB, dbPath string, prefix string, w, h int) MainViewModel {
	m := MainViewModel{
		db:            db,
		dbPath:        dbPath,
		currentPrefix: prefix,
		focus:         focusLeft,
		width:         w,
		height:        h,
	}

	m.refreshPanels()
	return m
}

func (m *MainViewModel) refreshPanels() {
	leftItems := m.queryToListItems(m.currentPrefix)
	leftTitle := "/"
	if m.currentPrefix != "" {
		leftTitle = m.currentPrefix
	}
	m.leftPanel = NewPanelListModel(leftTitle, leftItems)

	var middleItems []model.ListItem
	var middleTitle string
	if len(leftItems) > 0 {
		selected := leftItems[0]
		middleTitle = selected.Name
		if selected.IsEntry {
			middleItems = m.attrsToListItems(selected.FullPath)
		} else {
			middleItems = m.queryToListItems(selected.FullPath)
		}
	}
	m.middlePanel = NewPanelListModel(middleTitle, middleItems)

	m.rightPanel = NewEntryDetailModel("", m.db)
}

func (m *MainViewModel) queryToListItems(prefix string) []model.ListItem {
	keys := m.db.QueryKeys(prefix)
	seen := make(map[string]bool)
	items := make([]model.ListItem, 0)

	for _, key := range keys {
		rest := strings.TrimPrefix(key, prefix+"/")
		name := rest
		if idx := strings.Index(rest, "/"); idx >= 0 {
			name = rest[:idx]
		}
		fullPath := prefix + "/" + name
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

func (m *MainViewModel) attrsToListItems(entryPath string) []model.ListItem {
	keys := m.db.QueryKeys(entryPath)
	items := make([]model.ListItem, 0)
	for _, key := range keys {
		rest := strings.TrimPrefix(key, entryPath+"/")
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

func (m MainViewModel) Init() tea.Cmd {
	return nil
}

func (m MainViewModel) Update(msg tea.Msg) (MainViewModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.rightPanel, cmd = m.rightPanel.Update(msg)
		if m.rightPanel.selectedAttr == "TOTP" && m.rightPanel.HasSelectedEntry() {
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return tickMsg{}
			})
		}
		m.totpActive = false
		return m, nil

	case tea.KeyMsg:
		if m.pendingQuit {
			switch msg.String() {
			case "y":
				m.pendingQuit = false
				return m, func() tea.Msg { return SaveAndQuitMsg{} }
			case "n":
				return m, tea.Quit
			default:
				m.pendingQuit = false
				return m, nil
			}
		}

		if m.focus == focusRight {
			if m.rightPanel.State() != detailView || isDetailKey(msg.String()) {
				m.rightPanel, cmd = m.rightPanel.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "j", "down":
			m = m.handleDown()
		case "k", "up":
			m = m.handleUp()
		case "h", "left":
			m = m.handleLeft()
		case "l", "right", "enter":
			m = m.handleRight()
		case "tab":
			m.focus = (m.focus + 1) % 3
			if m.focus == focusRight && m.selectedEntry == "" {
				m.focus = focusLeft
			}
		case "ctrl+s":
			if m.dirty {
				return m, func() tea.Msg { return SaveVaultMsg{} }
			}
		case "n":
			return m, func() tea.Msg { return OpenNewEntryMsg{} }
		case "q":
			if m.dirty {
				m.pendingQuit = true
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			m = m.handleLeft()
		}
	}

	if m.rightPanel.selectedAttr == "TOTP" && m.rightPanel.HasSelectedEntry() && !m.totpActive {
		m.totpActive = true
		m.rightPanel.updateTOTP()
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg{}
		})
	}

	if m.rightPanel.selectedAttr != "TOTP" {
		m.totpActive = false
	}

	return m, cmd
}

func (m MainViewModel) handleUp() MainViewModel {
	switch m.focus {
	case focusLeft:
		m.leftPanel.MoveUp()
		m.onLeftSelectionChange()
	case focusMiddle:
		m.middlePanel.MoveUp()
		m.onMiddleSelectionChange()
	}
	return m
}

func (m MainViewModel) handleDown() MainViewModel {
	switch m.focus {
	case focusLeft:
		m.leftPanel.MoveDown()
		m.onLeftSelectionChange()
	case focusMiddle:
		m.middlePanel.MoveDown()
		m.onMiddleSelectionChange()
	}
	return m
}

func (m MainViewModel) handleLeft() MainViewModel {
	switch m.focus {
	case focusRight:
		m.focus = focusMiddle
	case focusMiddle:
		m.focus = focusLeft
	case focusLeft:
		if m.currentPrefix != "" {
			m.currentPrefix = model.ParentPath(m.currentPrefix)
			m.selectedEntry = ""
			m.refreshFromPrefix()
		}
	}
	return m
}

func (m MainViewModel) handleRight() MainViewModel {
	switch m.focus {
	case focusLeft:
		if m.leftPanel.ItemCount() == 0 {
			return m
		}
		selected := m.leftPanel.SelectedItem()
		if selected.IsEntry {
			m.selectedEntry = selected.FullPath
			m.middlePanel.SetTitle(selected.Name)
			m.middlePanel.SetItems(m.attrsToListItems(selected.FullPath))
			m.rightPanel = NewEntryDetailModel(selected.FullPath, m.db)
			if m.middlePanel.ItemCount() > 0 {
				attrItem := m.middlePanel.SelectedItem()
				m.rightPanel.SelectAttr(attrItem.Name)
			}
			m.focus = focusMiddle
			return m
		}
		m.currentPrefix = selected.FullPath
		m.selectedEntry = ""
		m.refreshFromPrefix()
		return m

	case focusMiddle:
		if m.middlePanel.ItemCount() == 0 {
			return m
		}
		selected := m.middlePanel.SelectedItem()

		if m.selectedEntry != "" {
			m.focus = focusRight
			m.rightPanel.SelectAttr(selected.Name)
			return m
		}

		if selected.IsEntry {
			m.selectedEntry = selected.FullPath
			m.middlePanel.SetTitle(selected.Name)
			m.middlePanel.SetItems(m.attrsToListItems(selected.FullPath))
			m.rightPanel = NewEntryDetailModel(selected.FullPath, m.db)
			if m.middlePanel.ItemCount() > 0 {
				attrItem := m.middlePanel.SelectedItem()
				m.rightPanel.SelectAttr(attrItem.Name)
			}
			m.focus = focusRight
			return m
		}

		m.currentPrefix = selected.FullPath
		m.selectedEntry = ""
		m.refreshFromPrefix()
		return m

	case focusRight:
		return m
	}

	return m
}

func (m *MainViewModel) refreshFromPrefix() {
	leftItems := m.queryToListItems(m.currentPrefix)
	leftTitle := "/"
	if m.currentPrefix != "" {
		leftTitle = m.currentPrefix
	}
	m.leftPanel = NewPanelListModel(leftTitle, leftItems)

	var middleItems []model.ListItem
	var middleTitle string
	if len(leftItems) > 0 {
		selected := leftItems[0]
		middleTitle = selected.Name
		if selected.IsEntry {
			middleItems = m.attrsToListItems(selected.FullPath)
		} else {
			middleItems = m.queryToListItems(selected.FullPath)
		}
	}
	m.middlePanel = NewPanelListModel(middleTitle, middleItems)

	m.rightPanel = NewEntryDetailModel("", m.db)
	m.focus = focusLeft
}

func (m *MainViewModel) onLeftSelectionChange() {
	if m.leftPanel.ItemCount() == 0 {
		m.selectedEntry = ""
		m.middlePanel.SetTitle("")
		m.middlePanel.SetItems(nil)
		m.rightPanel = NewEntryDetailModel("", m.db)
		return
	}

	selected := m.leftPanel.SelectedItem()
	m.selectedEntry = ""
	m.middlePanel.SetTitle(selected.Name)

	if selected.IsEntry {
		m.middlePanel.SetItems(m.attrsToListItems(selected.FullPath))
	} else {
		m.middlePanel.SetItems(m.queryToListItems(selected.FullPath))
	}

	m.rightPanel = NewEntryDetailModel("", m.db)
}

func (m *MainViewModel) onMiddleSelectionChange() {
	if m.middlePanel.ItemCount() == 0 {
		m.rightPanel = NewEntryDetailModel("", m.db)
		return
	}

	selected := m.middlePanel.SelectedItem()

	if m.selectedEntry != "" {
		m.rightPanel = NewEntryDetailModel(m.selectedEntry, m.db)
		m.rightPanel.SelectAttr(selected.Name)
		return
	}

	m.rightPanel = NewEntryDetailModel("", m.db)
}

func (m MainViewModel) buildStatusBar() string {
	if m.pendingQuit {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Bold(true).Render("[y] save & quit") + "  " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true).Render("[n] quit without saving") + "  " +
			keyEnabledStyle.Render("[esc] cancel")
	}

	var parts []string

	canNav := m.focus != focusRight || m.rightPanel.State() == detailView
	canBack := m.focus == focusLeft && m.currentPrefix != "" || m.focus == focusMiddle || m.focus == focusRight
	canOpen := canNav && (m.focus == focusLeft && m.leftPanel.ItemCount() > 0 || m.focus == focusMiddle && m.middlePanel.ItemCount() > 0)

	if m.focus == focusRight {
		switch m.rightPanel.State() {
		case detailView:
			canEdit := m.rightPanel.HasSelectedEntry()
			parts = append(parts, m.renderKey("[e] edit", canEdit))
			parts = append(parts, m.renderKey("[a] add", true))
			parts = append(parts, m.renderKey("[d] delete", canEdit))
			parts = append(parts, m.renderKey("[h] back", true))
		case detailEditAttr:
			return keyEnabledStyle.Render("[enter] save") + "  " + keyEnabledStyle.Render("[esc] cancel")
		case detailAddAttr:
			return keyEnabledStyle.Render("[enter] next") + "  " + keyEnabledStyle.Render("[esc] cancel")
		case detailConfirmDelete:
			return keyEnabledStyle.Render("[d/y] confirm") + "  " + keyEnabledStyle.Render("[any] cancel")
		}
	} else {
		parts = append(parts, m.renderKey("[h] back", canBack))
		parts = append(parts, m.renderKey("[l] open", canOpen))
		parts = append(parts, m.renderKey("[j/k] nav", canNav && (m.focus == focusLeft && m.leftPanel.ItemCount() > 1 || m.focus == focusMiddle && m.middlePanel.ItemCount() > 1)))
	}

	parts = append(parts, m.renderKey("[n] new", true))
	if m.dirty {
		parts = append(parts, m.renderKey("[Ctrl+S] save", true))
	}
	parts = append(parts, m.renderKey("[c] config", true))
	parts = append(parts, m.renderKey("[q] quit", true))

	return strings.Join(parts, "  ")
}

func (m MainViewModel) renderKey(text string, enabled bool) string {
	if enabled {
		return keyEnabledStyle.Render(text)
	}
	return keyDisabledStyle.Render(text)
}

func (m MainViewModel) View() string {
	w := m.width
	h := m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}

	leftWidth := clampInt(w/4, 18, 34)
	rightWidth := clampInt(w/2, 22, w-44)
	middleWidth := w - leftWidth - rightWidth
	if middleWidth < 12 {
		middleWidth = 12
		rightWidth = w - leftWidth - middleWidth
	}

	mainHeight := h - 2

	m.leftPanel.SetSize(leftWidth, mainHeight)
	m.middlePanel.SetSize(middleWidth, mainHeight)
	m.rightPanel.SetSize(rightWidth, mainHeight)

	m.leftPanel.SetFocused(m.focus == focusLeft)
	m.middlePanel.SetFocused(m.focus == focusMiddle)
	m.rightPanel.SetFocused(m.focus == focusRight)

	leftView := m.leftPanel.View()
	middleView := m.middlePanel.View()
	rightView := m.rightPanel.View()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftView, middleView, rightView)

	titleText := m.dbPath
	if titleText == "" {
		titleText = "tapass"
	}
	titleStyle := breadcrumbStyle
	if m.dirty {
		titleText += " [未保存]"
		titleStyle = breadcrumbDirtyStyle
	}
	titleLine := titleStyle.Width(w).Render(titleText)

	var statusText string
	statusText = m.buildStatusBar()
	status := statusBarStyle.Width(w).Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Top, titleLine, mainContent, status)
}

func (m *MainViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m MainViewModel) CurrentPrefix() string {
	return m.currentPrefix
}

func (m *MainViewModel) SetCurrentPrefix(prefix string) {
	m.currentPrefix = prefix
}

func (m MainViewModel) SelectedEntryPath() string {
	return m.selectedEntry
}

func (m *MainViewModel) RestoreSelection(entryPath string, focusPanel mainFocus) {
	if entryPath == "" {
		return
	}

	keys := m.db.QueryKeys(entryPath)
	if len(keys) == 0 {
		return
	}

	m.selectedEntry = entryPath

	entryName := entryPath
	if idx := strings.LastIndex(entryPath, "/"); idx >= 0 {
		entryName = entryPath[idx+1:]
	}

	for i := 0; i < m.leftPanel.ItemCount(); i++ {
		if m.leftPanel.items[i].FullPath == entryPath {
			m.leftPanel.cursor = i
			break
		}
	}

	m.middlePanel.SetTitle(entryName)
	m.middlePanel.SetItems(m.attrsToListItems(entryPath))
	m.rightPanel = NewEntryDetailModel(entryPath, m.db)
	if m.middlePanel.ItemCount() > 0 {
		attrItem := m.middlePanel.SelectedItem()
		m.rightPanel.SelectAttr(attrItem.Name)
	}

	m.focus = focusPanel
}

func (m *MainViewModel) SetDirty(d bool) {
	m.dirty = d
}
