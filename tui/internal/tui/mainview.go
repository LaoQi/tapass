package tui

import (
	"sort"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
	"github.com/tapass/tapass-tools/vault"
)

type MainViewModel struct {
	entries       map[string]vault.Entry
	vault         *vault.Vault
	dbPath        string
	currentPrefix string
	selectedEntry string

	leftPanel   PanelListModel
	middlePanel PanelListModel
	rightPanel  EntryDetailModel

	focus      mainFocus
	totpActive bool
	dirty      bool

	width  int
	height int

	pendingDelete bool
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

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func NewMainViewModel(entries map[string]vault.Entry, v *vault.Vault, dbPath string, prefix string, w, h int) MainViewModel {
	m := MainViewModel{
		entries:       entries,
		vault:         v,
		dbPath:        dbPath,
		currentPrefix: prefix,
		focus:         focusLeft,
		width:         w,
		height:        h,
	}

	leftItems := model.ListChildren(entries, prefix)
	leftTitle := "/"
	if prefix != "" {
		leftTitle = prefix
	}
	m.leftPanel = NewPanelListModel(leftTitle, leftItems)

	var middleItems []model.ListItem
	var middleTitle string
	if len(leftItems) > 0 {
		selected := leftItems[0]
		middleTitle = selected.Name
		middleItems = model.ListChildren(entries, selected.FullPath)
		if selected.IsEntry {
			middleItems = attrsToListItems(entries, selected.FullPath)
		}
	}
	m.middlePanel = NewPanelListModel(middleTitle, middleItems)

	m.rightPanel = NewEntryDetailModel("", entries, v)

	return m
}

func attrsToListItems(entries map[string]vault.Entry, entryPath string) []model.ListItem {
	attrs := model.GetEntryAttrs(entries, entryPath)
	items := make([]model.ListItem, 0, len(attrs))
	for name := range attrs {
		items = append(items, model.ListItem{
			Name:     name,
			FullPath: entryPath + "/" + name,
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
		if m.rightPanel.selectedAttr == "TOTP" && m.rightPanel.selectedEntry != nil {
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return tickMsg{}
			})
		}
		m.totpActive = false
		return m, nil

	case tea.KeyMsg:
		if m.pendingDelete {
			switch msg.String() {
			case "d", "y":
				m.pendingDelete = false
				if m.selectedEntry != "" {
					return m, func() tea.Msg { return DeleteEntryMsg{} }
				}
			default:
				m.pendingDelete = false
			}
			return m, nil
		}

		if m.focus == focusRight && m.rightPanel.State() != detailView {
			m.rightPanel, cmd = m.rightPanel.Update(msg)
			return m, cmd
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
			return m, func() tea.Msg { return OpenDBConfigMsg{} }
		case "n":
			return m, func() tea.Msg { return OpenNewEntryMsg{} }
		case "d":
			if m.selectedEntry != "" {
				m.pendingDelete = true
			}
		case "q":
			return m, tea.Quit
		case "esc":
			m = m.handleLeft()
		}
	}

	if m.rightPanel.selectedAttr == "TOTP" && m.rightPanel.selectedEntry != nil && !m.totpActive {
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
			m.middlePanel.SetItems(attrsToListItems(m.entries, selected.FullPath))
			m.rightPanel = NewEntryDetailModel(selected.FullPath, m.entries, m.vault)
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
			m.middlePanel.SetItems(attrsToListItems(m.entries, selected.FullPath))
			m.rightPanel = NewEntryDetailModel(selected.FullPath, m.entries, m.vault)
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
	leftItems := model.ListChildren(m.entries, m.currentPrefix)
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
			middleItems = attrsToListItems(m.entries, selected.FullPath)
		} else {
			middleItems = model.ListChildren(m.entries, selected.FullPath)
		}
	}
	m.middlePanel = NewPanelListModel(middleTitle, middleItems)

	m.rightPanel = NewEntryDetailModel("", m.entries, m.vault)
	m.focus = focusLeft
}

func (m *MainViewModel) onLeftSelectionChange() {
	if m.leftPanel.ItemCount() == 0 {
		m.selectedEntry = ""
		m.middlePanel.SetTitle("")
		m.middlePanel.SetItems(nil)
		m.rightPanel = NewEntryDetailModel("", m.entries, m.vault)
		return
	}

	selected := m.leftPanel.SelectedItem()
	m.selectedEntry = ""
	m.middlePanel.SetTitle(selected.Name)

	if selected.IsEntry {
		m.middlePanel.SetItems(attrsToListItems(m.entries, selected.FullPath))
	} else {
		m.middlePanel.SetItems(model.ListChildren(m.entries, selected.FullPath))
	}

	m.rightPanel = NewEntryDetailModel("", m.entries, m.vault)
}

func (m *MainViewModel) onMiddleSelectionChange() {
	if m.middlePanel.ItemCount() == 0 {
		m.rightPanel = NewEntryDetailModel("", m.entries, m.vault)
		return
	}

	selected := m.middlePanel.SelectedItem()

	if m.selectedEntry != "" {
		m.rightPanel = NewEntryDetailModel(m.selectedEntry, m.entries, m.vault)
		m.rightPanel.SelectAttr(selected.Name)
		return
	}

	m.rightPanel = NewEntryDetailModel("", m.entries, m.vault)
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
	if m.pendingDelete {
		statusText = errorStyle.Render("Confirm delete? [d/y] confirm  [any] cancel")
	} else {
		statusText = "[h] back  [l] open  [j/k] nav  [n] new  [d] delete  [Ctrl+S] settings  [q] quit"
	}
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

func (m MainViewModel) SelectedEntryPath() string {
	return m.selectedEntry
}

func (m *MainViewModel) SetDirty(d bool) {
	m.dirty = d
}
