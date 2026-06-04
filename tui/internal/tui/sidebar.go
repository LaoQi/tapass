package tui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tui/internal/model"
)

type SidebarModel struct {
	tree      *model.Node
	selected  *model.Node
	cursor    int
	flattened []*model.Node
	width     int
	height    int
}

var (
	groupStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA"))

	entryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	expandedIcon  = "▼ "
	collapsedIcon = "▶ "
	entryIcon     = "  "
)

func NewSidebarModel(tree *model.Node, selected *model.Node) SidebarModel {
	m := SidebarModel{
		tree:     tree,
		selected: selected,
	}
	m.flatten()
	return m
}

func (m SidebarModel) Init() tea.Cmd {
	return nil
}

func (m SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.selected = m.flattened[m.cursor]
			}
		case "down", "j":
			if m.cursor < len(m.flattened)-1 {
				m.cursor++
				m.selected = m.flattened[m.cursor]
			}
		case "enter":
			if m.selected != nil {
				if m.selected.IsGroup {
					return m, func() tea.Msg { return SidebarSelectMsg{} }
				}
				return m, func() tea.Msg {
					return SelectEntryMsg{Node: m.selected, GroupPath: ""}
				}
			}
		case " ":
			if m.selected != nil && m.selected.IsGroup {
				m.selected.Expanded = !m.selected.Expanded
				m.flatten()
			}
		}
	}
	return m, nil
}

func (m SidebarModel) View() string {
	height := m.height
	if height < 1 {
		height = 24
	}
	width := m.width
	if width < 1 {
		width = 20
	}

	var b strings.Builder
	for i, node := range m.flattened {
		if i >= height {
			break
		}

		prefix := entryIcon
		if node.IsGroup {
			if node.Expanded {
				prefix = expandedIcon
			} else {
				prefix = collapsedIcon
			}
		}

		depth := nodeDepth(node, m.tree)
		indent := strings.Repeat("  ", depth)

		label := node.ID
		maxLen := width - depth*2 - len(prefix)
		if maxLen > 0 && len(label) > maxLen {
			label = label[:maxLen-1] + "…"
		}

		line := indent + prefix + label

		if node == m.selected {
			b.WriteString(selectedStyle.Render(line))
		} else if node.IsGroup {
			b.WriteString(groupStyle.Render(line))
		} else {
			b.WriteString(entryStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(b.String())
}

func (m *SidebarModel) flatten() {
	m.flattened = nil
	if m.tree == nil {
		return
	}
	m.flattenNode(m.tree, 0)
}

func (m *SidebarModel) flattenNode(node *model.Node, depth int) {
	for _, child := range node.Children {
		m.flattened = append(m.flattened, child)
		if child.IsGroup && child.Expanded {
			m.flattenNode(child, depth+1)
		}
	}
}

func nodeDepth(node, root *model.Node) int {
	return countDepth(node.Path, root)
}

func countDepth(path string, root *model.Node) int {
	if root == nil {
		return 0
	}
	for _, child := range root.Children {
		if child.Path == path {
			return strings.Count(path, "/")
		}
		if child.IsGroup {
			d := countDepth(path, child)
			if d > 0 {
				return d
			}
		}
	}
	return 0
}

func (m SidebarModel) SelectedNode() *model.Node {
	return m.selected
}

func (m *SidebarModel) SetSelected(node *model.Node) {
	m.selected = node
	for i, n := range m.flattened {
		if n == node {
			m.cursor = i
			break
		}
	}
}

func (m *SidebarModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
