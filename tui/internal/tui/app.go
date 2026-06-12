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
	help     HelpViewModel
	width    int
	height   int
	err      error
}

func NewApp(dbPath string) AppModel {
	w := NewWelcomeModel()
	if dbPath != "" {
		w, _ = w.Update(initialPathMsg{Path: dbPath})
	}
	return AppModel{
		state:   StateWelcome,
		welcome: w,
		help:    NewHelpViewModel(),
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
		if m.help.Active() {
			switch msg.String() {
			case "esc", "?", "q":
				m.help, _ = m.help.Update(helpCloseMsg{})
			}
			return m, nil
		}

		if msg.String() == "?" && m.state == StateMainView && m.mainview.MainState() == StateBrowse && m.mainview.focus != focusSearch && m.mainview.rightPanel.State() == detailView {
			m.help, _ = m.help.Update(helpToggleMsg{})
			return m, nil
		}

		if msg.String() == "c" && m.state == StateMainView && m.mainview.MainState() == StateBrowse && m.mainview.focus != focusSearch && m.mainview.rightPanel.State() != detailEditKV {
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
		m.mainview, _ = m.mainview.Update(refreshMsg{})
		m = m.propagateSize()
		return m, nil

	case OpenDBConfigMsg:
		m.state = StateDBConfig
		m.dbconfig = NewDBConfigModel(m.db, m.dbPath)
		m = m.propagateSize()
		return m, nil

	case AttrChangedMsg:
		m.mainview, _ = m.mainview.Update(mainViewModelEventMsg{Event: model.Event{Type: model.EventAttrSet, Key: msg.Key}})
		m.mainview, _ = m.mainview.Update(dirtyMsg{Dirty: true})
		return m, nil

	case PasswordChangedMsg:
		m.state = StateMainView
		m.mainview, _ = m.mainview.Update(dirtyMsg{Dirty: true})
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
			m.mainview, _ = m.mainview.Update(cancelQuitMsg{})
			return m, func() tea.Msg { return ErrorMsg{Err: err} }
		}
		return m, func() tea.Msg { return VaultSavedMsg{QuitAfter: true} }

	case VaultSavedMsg:
		m.mainview, _ = m.mainview.Update(dirtyMsg{Dirty: false})
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

	if m.help.Active() && m.state == StateMainView {
		content = m.help.View()
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

	m.welcome, _ = m.welcome.Update(setWelcomeSizeMsg{Width: w, Height: h})
	m.mainview = m.mainview.SetSize(w, h)
	m.dbconfig = m.dbconfig.SetSize(w, h)
	m.help, _ = m.help.Update(setHelpSizeMsg{Width: w, Height: h})
	return m
}


