package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
)

type MainViewModel struct {
	tree          *model.Node
	selectedGroup *model.Node
	selectedEntry *model.Node
	sidebar       SidebarModel
	entrylist     EntryListModel
	width         int
	height        int
	focus         mainFocus
	pendingDelete bool
}

type mainFocus int

const (
	focusSidebar mainFocus = iota
	focusEntryList
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)
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

func NewMainViewModel(tree *model.Node, selectedGroup *model.Node, w, h int) MainViewModel {
	if selectedGroup == nil && tree != nil {
		selectedGroup = tree
		for _, child := range tree.Children {
			if child.IsGroup {
				selectedGroup = child
				break
			}
		}
	}
	return MainViewModel{
		tree:          tree,
		selectedGroup: selectedGroup,
		sidebar:       NewSidebarModel(tree, selectedGroup),
		entrylist:     NewEntryListModel(selectedGroup),
		focus:         focusSidebar,
		width:         w,
		height:        h,
	}
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
		m.propagateSize()
		return m, nil

	case SidebarSelectMsg:
		m.focus = focusEntryList
		return m, nil

	case tea.KeyMsg:
		if m.pendingDelete {
			switch msg.String() {
			case "d", "y":
				m.pendingDelete = false
				if m.focus == focusEntryList {
					if sel := m.entrylist.SelectedEntry(); sel != nil {
						return m, func() tea.Msg { return DeleteEntryMsg{} }
					}
				}
			default:
				m.pendingDelete = false
			}
			return m, nil
		}
		switch msg.String() {
		case "tab":
			if m.focus == focusSidebar {
				m.focus = focusEntryList
			} else {
				m.focus = focusSidebar
			}
			return m, nil
		case "ctrl+s":
			return m, func() tea.Msg { return OpenDBConfigMsg{} }
		case "n":
			return m, func() tea.Msg { return OpenNewEntryMsg{} }
		case "d":
			if m.focus == focusEntryList {
				if sel := m.entrylist.SelectedEntry(); sel != nil {
					m.pendingDelete = true
				}
			}
		case "q":
			return m, tea.Quit
		}
	}

	if m.focus == focusSidebar {
		m.sidebar, cmd = m.sidebar.Update(msg)
		sn := m.sidebar.SelectedNode()
		if sn != nil && sn.IsGroup && sn != m.selectedGroup {
			m.selectedGroup = sn
			m.entrylist = NewEntryListModel(m.selectedGroup)
		}
	} else {
		m.entrylist, cmd = m.entrylist.Update(msg)
		if m.focus == focusEntryList {
			if sel := m.entrylist.SelectedEntry(); sel != nil {
				m.selectedEntry = sel
			}
		}
	}

	return m, cmd
}

func (m MainViewModel) View() string {
	if m.tree == nil {
		return "No vault loaded"
	}

	w := m.width
	h := m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}

	sidebarWidth := clampInt(w/4, 16, 32)
	separatorWidth := 1
	contentWidth := w - sidebarWidth - separatorWidth
	if contentWidth < 10 {
		contentWidth = 10
	}

	mainHeight := h - 1

	m.sidebar.SetSize(sidebarWidth, mainHeight)
	m.entrylist.SetSize(contentWidth, mainHeight)

	sidebarView := m.sidebar.View()
	entryView := m.entrylist.View()

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151")).
		Render("│")

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, separator, entryView)

	var statusText string
	if m.pendingDelete {
		statusText = errorStyle.Render("Confirm delete? [d/y] confirm  [any] cancel")
	} else {
		statusText = "[Tab] switch  [Enter] open  [n] new  [d] delete  [Ctrl+S] settings  [q] quit"
		if m.selectedGroup != nil {
			statusText = fmt.Sprintf("%s | %s", m.selectedGroup.ID, statusText)
		}
	}
	status := statusBarStyle.Width(w).Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Top, mainContent, status)
}

func (m *MainViewModel) SetSelectedGroup(node *model.Node) {
	m.selectedGroup = node
	m.sidebar.SetSelected(node)
	m.entrylist = NewEntryListModel(node)
}

func (m *MainViewModel) propagateSize() {
	if m.width < 1 || m.height < 1 {
		return
	}
	sidebarWidth := clampInt(m.width/4, 16, 32)
	separatorWidth := 1
	contentWidth := m.width - sidebarWidth - separatorWidth
	mainHeight := m.height - 1
	m.sidebar.SetSize(sidebarWidth, mainHeight)
	m.entrylist.SetSize(contentWidth, mainHeight)
}
