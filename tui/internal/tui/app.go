package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/tapass/tapass-tui/internal/model"
	"github.com/tapass/tapass-tui/internal/store"
	"github.com/tapass/tapass-tools/vault"
)

type WindowState int

const (
	StateWelcome WindowState = iota
	StateMainView
	StateEntryDetail
	StateDBConfig
	StateNewEntry
)

type AppModel struct {
	state         WindowState
	store         store.Store
	vault         *vault.Vault
	tree          *model.Node
	selectedGroup *model.Node
	selectedEntry *model.Node
	dbPath        string

	welcome      WelcomeModel
	mainview     MainViewModel
	entrydetail  EntryDetailModel
	dbconfig     DBConfigModel
	newEntry     NewEntryDialogModel
	width        int
	height       int
	err          error
}

func NewApp(s store.Store) AppModel {
	return AppModel{
		state:   StateWelcome,
		store:   s,
		welcome: NewWelcomeModel(),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.welcome.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.propagateSize()
		return m, nil

	case tea.KeyMsg:

	case OpenVaultMsg:
		m.vault = msg.Vault
		m.dbPath = msg.Path
		m = m.rebuildTree()
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.tree, m.selectedGroup, m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case CreateVaultMsg:
		m.vault = msg.Vault
		m.dbPath = msg.Path
		m = m.rebuildTree()
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.tree, m.selectedGroup, m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case SelectEntryMsg:
		m.selectedEntry = msg.Node
		m.state = StateEntryDetail
		m.entrydetail = NewEntryDetailModel(msg.Node, msg.GroupPath, m.vault)
		m = m.propagateSize()
		return m, nil

	case BackToMainMsg:
		m.state = StateMainView
		m = m.rebuildTree()
		m.mainview = NewMainViewModel(m.tree, m.selectedGroup, m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case OpenDBConfigMsg:
		m.state = StateDBConfig
		m.dbconfig = NewDBConfigModel(m.vault, m.dbPath)
		m = m.propagateSize()
		return m, nil

	case EntryUpdatedMsg:
		m = m.rebuildTree()
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.tree, m.selectedGroup, m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case OpenNewEntryMsg:
		m.state = StateNewEntry
		groupPath := ""
		if m.selectedGroup != nil {
			groupPath = m.selectedGroup.Path
		}
		m.newEntry = NewNewEntryDialog(groupPath, m.vault)
		m = m.propagateSize()
		return m, nil

	case NewEntryCreatedMsg:
		entryNode := model.GetNodeByPath(m.tree, msg.EntryPathPrefix)
		if entryNode == nil {
			entryNode = model.NewNode(msg.EntryPathPrefix, msg.EntryPathPrefix)
		}
		m.selectedEntry = entryNode
		m.state = StateEntryDetail
		m.entrydetail = NewEntryDetailModel(entryNode, msg.GroupPath, m.vault)
		m = m.propagateSize()
		return m, nil

	case DeleteEntryMsg:
		if m.vault != nil && m.selectedEntry != nil {
			for k := range m.selectedEntry.Attrs {
				m.vault.Delete(m.selectedEntry.Path + "/" + k)
			}
		}
		m = m.rebuildTree()
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.tree, m.selectedGroup, m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case ErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	switch m.state {
	case StateWelcome:
		m.welcome, cmd = m.welcome.Update(msg)
	case StateMainView:
		m.mainview, cmd = m.mainview.Update(msg)
	case StateEntryDetail:
		m.entrydetail, cmd = m.entrydetail.Update(msg)
	case StateDBConfig:
		m.dbconfig, cmd = m.dbconfig.Update(msg)
	case StateNewEntry:
		m.newEntry, cmd = m.newEntry.Update(msg)
	}

	return m, cmd
}

func (m AppModel) View() string {
	switch m.state {
	case StateWelcome:
		return m.welcome.View()
	case StateMainView:
		return m.mainview.View()
	case StateEntryDetail:
		return m.entrydetail.View()
	case StateDBConfig:
		return m.dbconfig.View()
	case StateNewEntry:
		return m.newEntry.View()
	}
	return ""
}

func (m *AppModel) SetStore(s store.Store) {
	m.store = s
	m.welcome.SetStore(s)
}

func (m AppModel) rebuildTree() AppModel {
	m.tree = model.BuildTree(m.vault.List())
	if m.selectedGroup == nil && m.tree != nil {
		m.selectedGroup = m.tree
		for _, child := range m.tree.Children {
			if child.IsGroup {
				m.selectedGroup = child
				break
			}
		}
	}
	return m
}

func (m AppModel) propagateSize() AppModel {
	w := m.width
	h := m.height
	if w < 1 || h < 1 {
		return m
	}

	m.welcome.SetSize(w, h)
	m.entrydetail.SetSize(w, h)
	m.dbconfig.SetSize(w, h)
	m.newEntry.SetSize(w, h)
	return m
}

type OpenVaultMsg struct {
	Vault *vault.Vault
	Path  string
}

type CreateVaultMsg struct {
	Vault *vault.Vault
	Path  string
}

type SelectEntryMsg struct {
	Node      *model.Node
	GroupPath string
}

type BackToMainMsg struct{}

type OpenDBConfigMsg struct{}

type EntryUpdatedMsg struct{}

type OpenNewEntryMsg struct{}

type DeleteEntryMsg struct{}

type SidebarSelectMsg struct{}

type NewEntryCreatedMsg struct {
	EntryPathPrefix string
	GroupPath       string
}

type ErrorMsg struct {
	Err error
}
