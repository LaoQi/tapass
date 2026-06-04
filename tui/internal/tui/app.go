package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/tapass/tapass-tui/internal/store"
	"github.com/tapass/tapass-tools/vault"
)

type WindowState int

const (
	StateWelcome WindowState = iota
	StateMainView
	StateDBConfig
	StateNewEntry
)

type AppModel struct {
	state   WindowState
	store   store.Store
	vault   *vault.Vault
	entries map[string]vault.Entry
	dbPath  string

	welcome  WelcomeModel
	mainview MainViewModel
	dbconfig DBConfigModel
	newEntry NewEntryDialogModel
	width    int
	height   int
	err      error
}

func NewApp(s store.Store) AppModel {
	return AppModel{
		state:   StateWelcome,
		store:   s,
		welcome: NewWelcomeModel(),
	}
}

func (m *AppModel) SetInitialDBPath(path string) {
	if path != "" {
		m.welcome.SetInitialPath(path)
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
		switch m.state {
		case StateWelcome:
			m.welcome, cmd = m.welcome.Update(msg)
		case StateMainView:
			m.mainview, cmd = m.mainview.Update(msg)
		case StateDBConfig:
			m.dbconfig, cmd = m.dbconfig.Update(msg)
		case StateNewEntry:
			m.newEntry, cmd = m.newEntry.Update(msg)
		}
		return m, cmd

	case tea.KeyMsg:

	case OpenVaultMsg:
		m.vault = msg.Vault
		m.dbPath = msg.Path
		m.entries = m.vault.List()
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.entries, m.vault, m.dbPath, "", m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case CreateVaultMsg:
		m.vault = msg.Vault
		m.dbPath = msg.Path
		m.entries = m.vault.List()
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.entries, m.vault, m.dbPath, "", m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case BackToMainMsg:
		m.state = StateMainView
		m.entries = m.vault.List()
		m.mainview = NewMainViewModel(m.entries, m.vault, m.dbPath, "", m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case OpenDBConfigMsg:
		m.state = StateDBConfig
		m.dbconfig = NewDBConfigModel(m.vault, m.dbPath)
		m = m.propagateSize()
		return m, nil

	case EntryUpdatedMsg:
		m.entries = m.vault.List()
		m.mainview = NewMainViewModel(m.entries, m.vault, m.dbPath, "", m.width, m.height)
		m.mainview.SetDirty(true)
		m.state = StateMainView
		m = m.propagateSize()
		return m, nil

	case OpenNewEntryMsg:
		m.state = StateNewEntry
		currentPrefix := ""
		if m.mainview.entries != nil {
			currentPrefix = m.mainview.CurrentPrefix()
		}
		m.newEntry = NewNewEntryDialog(currentPrefix, m.vault)
		m = m.propagateSize()
		return m, nil

	case NewEntryCreatedMsg:
		m.entries = m.vault.List()
		m.state = StateMainView
		prefix := modelParentPath(msg.EntryPathPrefix)
		m.mainview = NewMainViewModel(m.entries, m.vault, m.dbPath, prefix, m.width, m.height)
		m.mainview.SetDirty(true)
		m = m.propagateSize()
		return m, nil

	case DeleteEntryMsg:
		if m.vault != nil {
			entryPath := m.mainview.SelectedEntryPath()
			if entryPath != "" {
				prefix := entryPath + "/"
				for key := range m.entries {
					if len(key) > len(prefix) && key[:len(prefix)] == prefix {
						m.vault.Delete(key)
					}
				}
			}
		}
		m.entries = m.vault.List()
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.entries, m.vault, m.dbPath, "", m.width, m.height)
		m.mainview.SetDirty(true)
		m = m.propagateSize()
		return m, nil

	case PasswordChangedMsg:
		m.state = StateMainView
		m.mainview.SetDirty(true)
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

func (m AppModel) propagateSize() AppModel {
	w := m.width
	h := m.height
	if w < 1 || h < 1 {
		return m
	}

	m.welcome.SetSize(w, h)
	m.mainview.SetSize(w, h)
	m.dbconfig.SetSize(w, h)
	m.newEntry.SetSize(w, h)
	return m
}

func modelParentPath(path string) string {
	if path == "" {
		return ""
	}
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			lastSlash = i
			break
		}
	}
	if lastSlash <= 0 {
		return ""
	}
	return path[:lastSlash]
}

type OpenVaultMsg struct {
	Vault *vault.Vault
	Path  string
}

type CreateVaultMsg struct {
	Vault *vault.Vault
	Path  string
}

type BackToMainMsg struct{}

type OpenDBConfigMsg struct{}

type EntryUpdatedMsg struct{}

type OpenNewEntryMsg struct{}

type DeleteEntryMsg struct{}

type PasswordChangedMsg struct{}

type NewEntryCreatedMsg struct {
	EntryPathPrefix string
	GroupPath       string
}

type ErrorMsg struct {
	Err error
}
