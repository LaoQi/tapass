package tui

import (
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tapass/tapass-tui/internal/model"
)

type MainState int

const (
	StateBrowse MainState = iota
	StatePendingQuit
)

type MainViewModel struct {
	db     *model.DB
	dbPath string

	leftPanel  PanelListModel
	rightPanel EntryDetailModel

	state        MainState
	focus        mainFocus
	searchActive bool
	totpActive   bool
	dirty        bool

	width  int
	height int
}

type mainFocus int

const (
	focusSearch mainFocus = iota
	focusLeft
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
	case "e", "d", "y":
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

func (m MainViewModel) MainState() MainState {
	return m.state
}

func (m *MainViewModel) CancelQuit() {
	if m.state == StatePendingQuit {
		m.state = StateBrowse
	}
}

func NewMainViewModel(db *model.DB, dbPath string, prefix string, w, h int) MainViewModel {
	m := MainViewModel{
		db:     db,
		dbPath: dbPath,
		focus:  focusLeft,
		width:  w,
		height: h,
	}

	m.leftPanel = NewPanelListModel(db, prefix)
	m.rightPanel = NewEntryDetailModel("", db)

	m.syncRightFromLeft()
	return m
}

func (m *MainViewModel) syncRightFromLeft() {
	if m.leftPanel.ItemCount() == 0 {
		m.rightPanel.SetEntryPath("")
		m.rightPanel.SetAttrList(nil)
		return
	}

	selected := m.leftPanel.SelectedItem()
	if selected.Depth > 0 {
		m.rightPanel.SetEntryPath("")
		m.rightPanel.SetDetailMode()
		attrs := m.queryAttributes(selected.FullPath)
		m.rightPanel.SetAttrList(attrs)
	} else {
		m.rightPanel.SetDetailMode()
		entryPath := model.ParentPath(selected.FullPath)
		m.rightPanel.SetEntryPath(entryPath)
		m.rightPanel.SelectAttr(selected.Name)
	}
}

func (m *MainViewModel) queryAttributes(prefix string) []AttrInfo {
	keys := m.db.QueryKeys(prefix)
	var attrs []AttrInfo
	for _, key := range keys {
		rest := strings.TrimPrefix(key, prefix+"/")
		if !strings.Contains(rest, "/") {
			if e, ok := m.db.Get(key); ok {
				attrs = append(attrs, AttrInfo{
					Name:      rest,
					Timestamp: e.Timestamp,
				})
			}
		}
	}
	return attrs
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

	case copyClearMsg:
		m.rightPanel, cmd = m.rightPanel.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		switch m.state {
		case StatePendingQuit:
			return m.handlePendingQuitKey(msg)
		case StateBrowse:
			return m.handleBrowseKey(msg)
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

func (m MainViewModel) handlePendingQuitKey(msg tea.KeyPressMsg) (MainViewModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.state = StateBrowse
		return m, func() tea.Msg { return SaveAndQuitMsg{} }
	case "n":
		return m, tea.Quit
	default:
		m.state = StateBrowse
		return m, nil
	}
}

func (m MainViewModel) handleBrowseKey(msg tea.KeyPressMsg) (MainViewModel, tea.Cmd) {
	if m.searchActive && msg.String() == "esc" {
		m = m.exitSearch()
		return m, nil
	}

	if m.focus == focusSearch {
		return m.handleSearchFocusKey(msg)
	}

	if m.focus == focusRight {
		if m.rightPanel.State() != detailView || isDetailKey(msg.String()) {
			var cmd tea.Cmd
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
		if m.focus == focusSearch {
			m.leftPanel.FocusSearchInput()
		} else {
			m.leftPanel.BlurSearchInput()
		}
	case "ctrl+s":
		if m.dirty {
			return m, func() tea.Msg { return SaveVaultMsg{} }
		}
	case "n":
		m = m.handleNewKV()
	case "e", "y", "d":
		if m.focus == focusLeft && m.leftPanel.ItemCount() > 0 && m.leftPanel.SelectedItem().Depth == 0 {
			m.focus = focusRight
			var cmd tea.Cmd
			m.rightPanel, cmd = m.rightPanel.Update(msg)
			return m, cmd
		}
	case "q":
		if m.dirty {
			m.state = StatePendingQuit
			return m, nil
		}
		return m, tea.Quit
	case "esc":
		m = m.handleLeft()
	case "/":
		m = m.handleEnterSearch()
	}
	return m, nil
}

func (m MainViewModel) handleUp() MainViewModel {
	if m.focus == focusLeft {
		m.leftPanel.MoveUp()
		m.syncRightFromLeft()
	}
	return m
}

func (m MainViewModel) handleSearchFocusKey(msg tea.KeyPressMsg) (MainViewModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focus = focusLeft
		m.leftPanel.BlurSearchInput()
		return m, nil
	case "enter":
		m.focus = focusLeft
		m.leftPanel.BlurSearchInput()
		return m, nil
	default:
		si := m.leftPanel.SearchInput()
		newSi, cmd := si.Update(msg)
		m.leftPanel.SetSearchInput(newSi)
		m.performSearch(newSi.Value())
		return m, cmd
	}
}

func (m MainViewModel) exitSearch() MainViewModel {
	m.searchActive = false
	m.leftPanel.ExitSearch()
	m.syncRightFromLeft()
	m.focus = focusLeft
	return m
}

func (m MainViewModel) handleDown() MainViewModel {
	if m.focus == focusLeft {
		m.leftPanel.MoveDown()
		m.syncRightFromLeft()
	}
	return m
}

func (m MainViewModel) handleNewKV() MainViewModel {
	var prefix string
	if m.focus == focusLeft {
		selected := m.leftPanel.SelectedItem()
		if selected.Depth > 0 {
			prefix = selected.FullPath
		} else {
			prefix = m.leftPanel.Prefix()
		}
	} else {
		prefix = m.rightPanel.EntryPath()
	}
	m.rightPanel.StartNew(prefix)
	m.focus = focusRight
	return m
}

func (m MainViewModel) handleLeft() MainViewModel {
	switch m.focus {
	case focusSearch:
		m.focus = focusLeft
		m.leftPanel.BlurSearchInput()
	case focusRight:
		m.focus = focusLeft
	case focusLeft:
		if m.leftPanel.Prefix() != "" {
			parent := model.ParentPath(m.leftPanel.Prefix())
			m.leftPanel.SetPrefix(parent)
			m.syncRightFromLeft()
		}
	}
	return m
}

func (m MainViewModel) handleRight() MainViewModel {
	if m.focus != focusLeft || m.leftPanel.ItemCount() == 0 {
		return m
	}

	selected := m.leftPanel.SelectedItem()
	if selected.Depth > 0 {
		m.leftPanel.SetPrefix(selected.FullPath)
		m.syncRightFromLeft()
	} else {
		m.focus = focusRight
	}
	return m
}

func (m MainViewModel) handleEnterSearch() MainViewModel {
	m.leftPanel.EnterSearch()
	m.searchActive = true
	m.focus = focusSearch
	m.leftPanel.FocusSearchInput()
	return m
}

func (m *MainViewModel) performSearch(query string) {
	m.leftPanel.ApplySearchFilter(query)
	m.syncRightFromLeft()
}

func (m *MainViewModel) HandleDBEvent(evt model.Event) {
	m.leftPanel.HandleEvent(evt)
	m.rightPanel.HandleEvent(evt)
	if evt.Type == model.EventAttrSet || evt.Type == model.EventAttrDeleted {
		m.syncRightFromLeft()
	}
}

func (m *MainViewModel) RefreshAll() {
	m.leftPanel.Refresh()
	m.rightPanel.Refresh()
}

func (m MainViewModel) buildStatusBar() string {
	if m.state == StatePendingQuit {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Bold(true).Render("[y] save & quit") + "  " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true).Render("[n] quit without saving") + "  " +
			keyEnabledStyle.Render("[esc] cancel")
	}

	var parts []string

	if m.searchActive {
		parts = append(parts, keyEnabledStyle.Render("[esc] exit search"))
		parts = append(parts, keyEnabledStyle.Render("[tab] switch focus"))
		if m.focus != focusSearch {
			canBack := m.leftPanel.Prefix() != ""
			canOpen := m.leftPanel.ItemCount() > 0
			canNav := m.leftPanel.ItemCount() > 1
			parts = append(parts, m.renderKey("[h] back", canBack))
			parts = append(parts, m.renderKey("[l] open", canOpen))
			parts = append(parts, m.renderKey("[j/k] nav", canNav))
		}
		parts = append(parts, m.renderKey("[n] new", true))
		if m.dirty {
			parts = append(parts, m.renderKey("[Ctrl+S] save", true))
		}
		parts = append(parts, m.renderKey("[c] config", true))
		parts = append(parts, m.renderKey("[?] help", true))
		parts = append(parts, m.renderKey("[q] quit", true))
		return strings.Join(parts, "  ")
	}

	if m.focus == focusRight {
		switch m.rightPanel.State() {
		case detailView:
			canEdit := m.rightPanel.HasSelectedEntry()
			parts = append(parts, m.renderKey("[e] edit", canEdit))
			parts = append(parts, m.renderKey("[y] copy", canEdit))
			parts = append(parts, m.renderKey("[d] delete", canEdit))
			parts = append(parts, m.renderKey("[h] back", true))
		case detailEditKV:
			return keyEnabledStyle.Render("[Alt+S] save") + "  " + keyEnabledStyle.Render("[Tab] switch field") + "  " + keyEnabledStyle.Render("[Esc] cancel")
		case detailConfirmDelete:
			return keyEnabledStyle.Render("[d/y] confirm") + "  " + keyEnabledStyle.Render("[any] cancel")
		}
	} else {
		canBack := m.leftPanel.Prefix() != ""
		canOpen := m.leftPanel.ItemCount() > 0
		canNav := m.leftPanel.ItemCount() > 1
		parts = append(parts, m.renderKey("[h] back", canBack))
		parts = append(parts, m.renderKey("[l] open", canOpen))
		parts = append(parts, m.renderKey("[j/k] nav", canNav))

		if m.leftPanel.ItemCount() > 0 && m.leftPanel.SelectedItem().Depth == 0 {
			parts = append(parts, m.renderKey("[e] edit", true))
			parts = append(parts, m.renderKey("[y] copy", true))
			parts = append(parts, m.renderKey("[d] delete", true))
		}
	}

	parts = append(parts, m.renderKey("[n] new", true))
	if m.dirty {
		parts = append(parts, m.renderKey("[Ctrl+S] save", true))
	}
	parts = append(parts, m.renderKey("[/] search", true))
	parts = append(parts, m.renderKey("[c] config", true))
	parts = append(parts, m.renderKey("[?] help", true))
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

	leftWidth := w / 3
	if leftWidth < 18 {
		leftWidth = 18
	}
	rightWidth := w - leftWidth
	if rightWidth < 22 {
		rightWidth = 22
		leftWidth = w - rightWidth
	}

	mainHeight := h - 2

	m.leftPanel.SetSize(leftWidth, mainHeight)
	m.rightPanel.SetSize(rightWidth, mainHeight)

	m.leftPanel.SetFocused(m.focus == focusLeft || m.focus == focusSearch)
	m.rightPanel.SetFocused(m.focus == focusRight)

	leftView := m.leftPanel.View()
	rightView := m.rightPanel.View()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)

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
	return m.leftPanel.Prefix()
}

func (m *MainViewModel) SetCurrentPrefix(prefix string) {
	m.leftPanel.SetPrefix(prefix)
	m.syncRightFromLeft()
}

func (m MainViewModel) SelectedEntryPath() string {
	if m.leftPanel.ItemCount() == 0 {
		return ""
	}
	selected := m.leftPanel.SelectedItem()
	if selected.Depth == 0 {
		return model.ParentPath(selected.FullPath)
	}
	return ""
}

func (m *MainViewModel) RestoreSelection(entryPath string, focusPanel mainFocus) {
	if entryPath == "" {
		return
	}

	keys := m.db.QueryKeys(entryPath)
	if len(keys) == 0 {
		return
	}

	parent := model.ParentPath(entryPath)
	m.leftPanel.SetPrefix(parent)

	for i := 0; i < m.leftPanel.ItemCount(); i++ {
		if m.leftPanel.items[i].FullPath == entryPath {
			m.leftPanel.cursor = i
			break
		}
	}

	m.syncRightFromLeft()
	m.focus = focusPanel
}

func (m *MainViewModel) SetDirty(d bool) {
	m.dirty = d
}
