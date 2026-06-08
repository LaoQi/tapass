package tui

import (
	"charm.land/bubbletea/v2"
	"github.com/tapass/tapass-tui/internal/model"
)

type WindowState int

const (
	StateWelcome WindowState = iota
	StateMainView
	StateDBConfig
)

type AppModel struct {
	state  WindowState
	db     *model.DB
	dbPath string

	welcome  WelcomeModel
	mainview MainViewModel
	dbconfig DBConfigModel
	width    int
	height   int
	err      error
}

func NewApp() AppModel {
	return AppModel{
		state:   StateWelcome,
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
		}
		return m, cmd

	case tea.KeyPressMsg:
		if msg.String() == "c" && m.state == StateMainView && m.mainview.rightPanel.State() != detailEditKV {
			return m, func() tea.Msg { return OpenDBConfigMsg{} }
		}

	case OpenVaultMsg:
		m.db = msg.DB
		m.dbPath = msg.Path
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.db, m.dbPath, "", m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case CreateVaultMsg:
		m.db = msg.DB
		m.dbPath = msg.Path
		m.state = StateMainView
		m.mainview = NewMainViewModel(m.db, m.dbPath, "", m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case BackToMainMsg:
		m.state = StateMainView
		m.mainview.RefreshAll()
		m = m.propagateSize()
		return m, nil

	case OpenDBConfigMsg:
		m.state = StateDBConfig
		m.dbconfig = NewDBConfigModel(m.db, m.dbPath)
		m = m.propagateSize()
		return m, nil

	case AttrChangedMsg:
		m.mainview.HandleDBEvent(model.Event{Type: model.EventAttrSet, Key: msg.Key})
		m.mainview.SetDirty(true)
		return m, nil

	case PasswordChangedMsg:
		m.state = StateMainView
		m.mainview.SetDirty(true)
		m = m.propagateSize()
		return m, nil

	case SaveVaultMsg:
		if m.db == nil {
			return m, nil
		}
		if err := m.db.Save(); err != nil {
			return m, func() tea.Msg { return ErrorMsg{Err: err} }
		}
		return m, func() tea.Msg { return VaultSavedMsg{QuitAfter: false} }

	case SaveAndQuitMsg:
		if m.db == nil {
			return m, tea.Quit
		}
		if err := m.db.Save(); err != nil {
			m.mainview.pendingQuit = false
			return m, func() tea.Msg { return ErrorMsg{Err: err} }
		}
		return m, func() tea.Msg { return VaultSavedMsg{QuitAfter: true} }

	case VaultSavedMsg:
		m.mainview.SetDirty(false)
		if msg.QuitAfter {
			return m, tea.Quit
		}
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
	}

	return m, cmd
}

func (m AppModel) View() tea.View {
	var content string
	switch m.state {
	case StateWelcome:
		content = m.welcome.View()
	case StateMainView:
		content = m.mainview.View()
	case StateDBConfig:
		content = m.dbconfig.View()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
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
	return m
}

type OpenVaultMsg struct {
	DB   *model.DB
	Path string
}

type CreateVaultMsg struct {
	DB   *model.DB
	Path string
}

type BackToMainMsg struct{}

type OpenDBConfigMsg struct{}

type AttrChangedMsg struct {
	Key string
}

type PasswordChangedMsg struct{}

type SaveVaultMsg struct{}

type SaveAndQuitMsg struct{}

type VaultSavedMsg struct {
	QuitAfter bool
}

type ErrorMsg struct {
	Err error
}
