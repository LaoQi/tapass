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

func (m MainViewModel) MainState() MainState {
	return m.state
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

	m = m.syncRightFromLeft()
	return m
}

func (m MainViewModel) syncRightFromLeft() MainViewModel {
	if m.leftPanel.ItemCount() == 0 {
		m.rightPanel, _ = m.rightPanel.Update(syncRightMsg{ClearOnly: true})
		return m
	}
	selected := m.leftPanel.SelectedItem()
	if selected.Depth > 0 {
		attrs := m.queryAttributes(selected.FullPath)
		m.rightPanel, _ = m.rightPanel.Update(syncRightMsg{
			EntryPath:    "",
			Attrs:        attrs,
			SetDetailMode: true,
		})
	} else {
		entryPath := model.ParentPath(selected.FullPath)
		m.rightPanel, _ = m.rightPanel.Update(syncRightMsg{
			EntryPath:    entryPath,
			SelectedAttr: selected.Name,
			SetDetailMode: true,
		})
	}
	return m
}

func (m MainViewModel) queryAttributes(prefix string) []AttrInfo {
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

	case dirtyMsg:
		m.dirty = msg.Dirty
		return m, nil
	case cancelQuitMsg:
		if m.state == StatePendingQuit {
			m.state = StateBrowse
		}
		return m, nil
	case mainViewModelEventMsg:
		m.leftPanel, _ = m.leftPanel.Update(dbEventMsg{Event: msg.Event})
		m.rightPanel, _ = m.rightPanel.Update(dbEventMsg{Event: msg.Event})
		if msg.Event.Type == model.EventAttrSet || msg.Event.Type == model.EventAttrDeleted {
			m = m.syncRightFromLeft()
		}
		return m, nil
	case refreshMsg:
		m.leftPanel, _ = m.leftPanel.Update(refreshMsg{})
		m.rightPanel, _ = m.rightPanel.Update(refreshMsg{})
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
		m.rightPanel = m.rightPanel.updateTOTP()
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
			m.leftPanel, _ = m.leftPanel.Update(searchFocusMsg{})
		} else {
			m.leftPanel, _ = m.leftPanel.Update(searchBlurMsg{})
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
		m.leftPanel, _ = m.leftPanel.Update(moveUpMsg{})
		m = m.syncRightFromLeft()
	}
	return m
}

func (m MainViewModel) handleSearchFocusKey(msg tea.KeyPressMsg) (MainViewModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focus = focusLeft
		m.leftPanel, _ = m.leftPanel.Update(searchBlurMsg{})
		return m, nil
	case "enter":
		m.focus = focusLeft
		m.leftPanel, _ = m.leftPanel.Update(searchBlurMsg{})
		return m, nil
	default:
		var cmd tea.Cmd
		m.leftPanel, cmd = m.leftPanel.Update(msg)
		m = m.syncRightFromLeft()
		return m, cmd
	}
}

func (m MainViewModel) exitSearch() MainViewModel {
	m.searchActive = false
	m.leftPanel, _ = m.leftPanel.Update(searchExitMsg{})
	m = m.syncRightFromLeft()
	m.focus = focusLeft
	return m
}

func (m MainViewModel) handleDown() MainViewModel {
	if m.focus == focusLeft {
		m.leftPanel, _ = m.leftPanel.Update(moveDownMsg{})
		m = m.syncRightFromLeft()
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
	m.rightPanel, _ = m.rightPanel.Update(startNewMsg{Prefix: prefix})
	m.focus = focusRight
	return m
}

func (m MainViewModel) handleLeft() MainViewModel {
	switch m.focus {
	case focusSearch:
		m.focus = focusLeft
		m.leftPanel, _ = m.leftPanel.Update(searchBlurMsg{})
	case focusRight:
		m.focus = focusLeft
	case focusLeft:
		if m.leftPanel.Prefix() != "" {
			parent := model.ParentPath(m.leftPanel.Prefix())
			m.leftPanel, _ = m.leftPanel.Update(setPrefixMsg{Prefix: parent})
			m = m.syncRightFromLeft()
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
		m.leftPanel, _ = m.leftPanel.Update(setPrefixMsg{Prefix: selected.FullPath})
		m = m.syncRightFromLeft()
	} else {
		m.focus = focusRight
	}
	return m
}

func (m MainViewModel) handleEnterSearch() MainViewModel {
	m.leftPanel, _ = m.leftPanel.Update(searchEnterMsg{})
	m.searchActive = true
	m.focus = focusSearch
	m.leftPanel, _ = m.leftPanel.Update(searchFocusMsg{})
	return m
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

	m.leftPanel = m.leftPanel.SetSize(leftWidth, mainHeight)
	m.rightPanel = m.rightPanel.SetSize(rightWidth, mainHeight)

	m.leftPanel = m.leftPanel.SetFocused(m.focus == focusLeft || m.focus == focusSearch)
	m.rightPanel = m.rightPanel.SetFocused(m.focus == focusRight)

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

func (m MainViewModel) SetSize(w, h int) MainViewModel {
	m.width = w
	m.height = h
	return m
}
