package tui

import (
	"charm.land/bubbletea/v2"
	"github.com/LaoQi/tapass/tui/internal/model"
)

type WindowState int

const (
	StateWelcome WindowState = iota
	StateMainView
	StateHelp
	StateDBConfig
)

type AppState struct {
	DB     *model.DB
	DBPath string
}

type AppModel struct {
	state  WindowState
	app    AppState
	page   tea.Model
	width  int
	height int
	err    error
}

func NewApp(dbPath string) AppModel {
	w := NewWelcomeModel()
	page := w
	if dbPath != "" {
		np, _ := w.Update(initialPathMsg{Path: dbPath})
		page = np.(WelcomeModel)
	}
	return AppModel{
		state: StateWelcome,
		page:  page,
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.page.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.propagateSize()
		return m, nil

	case tea.KeyPressMsg:
		if m.state == StateHelp {
			switch msg.String() {
			case "esc", "?", "q":
				m = m.switchToMainView()
			}
			return m, nil
		}

		if msg.String() == "?" && m.state == StateMainView {
			mv := m.page.(MainViewModel)
			if mv.MainState() == StateBrowse && mv.focus != focusSearch && mv.rightPanel.State() == detailView {
				m.page = NewHelpViewModel()
				m.state = StateHelp
				m = m.propagateSize()
				return m, nil
			}
		}

		if msg.String() == "c" && m.state == StateMainView {
			mv := m.page.(MainViewModel)
			if mv.MainState() == StateBrowse && mv.focus != focusSearch && mv.rightPanel.State() != detailEditKV {
				return m, func() tea.Msg { return OpenDBConfigMsg{} }
			}
		}

	case OpenVaultMsg:
		m.app.DB = msg.DB
		m.app.DBPath = msg.Path
		m.state = StateMainView
		m.page = NewMainViewModel(m.app.DB, m.app.DBPath, "", m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case CreateVaultMsg:
		m.app.DB = msg.DB
		m.app.DBPath = msg.Path
		m.state = StateMainView
		m.page = NewMainViewModel(m.app.DB, m.app.DBPath, "", m.width, m.height)
		m = m.propagateSize()
		return m, nil

	case BackToMainMsg:
		m = m.switchToMainView()
		if mv, ok := m.page.(MainViewModel); ok {
			m.page = updateMainView(mv, refreshMsg{})
		}
		return m, nil

	case OpenDBConfigMsg:
		m.state = StateDBConfig
		m.page = NewDBConfigModel(m.app.DB, m.app.DBPath)
		m = m.propagateSize()
		return m, nil

	case AttrChangedMsg:
		if mv, ok := m.page.(MainViewModel); ok {
			mv = updateMainView(mv, mainViewModelEventMsg{Event: model.Event{Type: model.EventAttrSet, Key: msg.Key}})
			mv = updateMainView(mv, dirtyMsg{Dirty: m.app.DB.Dirty()})
			m.page = mv
		}
		return m, nil

	case PasswordChangedMsg:
		m = m.switchToMainView()
		if mv, ok := m.page.(MainViewModel); ok {
			mv = updateMainView(mv, dirtyMsg{Dirty: m.app.DB.Dirty()})
			m.page = mv
		}
		return m, nil

	case SaveVaultMsg:
		if m.app.DB == nil {
			return m, nil
		}
		if err := m.app.DB.Save(); err != nil {
			return m, func() tea.Msg { return ErrorMsg{Err: err} }
		}
		return m, func() tea.Msg { return VaultSavedMsg{QuitAfter: false} }

	case SaveAndQuitMsg:
		if m.app.DB == nil {
			return m, tea.Quit
		}
		if err := m.app.DB.Save(); err != nil {
			if mv, ok := m.page.(MainViewModel); ok {
				m.page = updateMainView(mv, cancelQuitMsg{})
			}
			return m, func() tea.Msg { return ErrorMsg{Err: err} }
		}
		return m, func() tea.Msg { return VaultSavedMsg{QuitAfter: true} }

	case VaultSavedMsg:
		if mv, ok := m.page.(MainViewModel); ok {
			m.page = updateMainView(mv, dirtyMsg{Dirty: m.app.DB.Dirty()})
		}
		if msg.QuitAfter {
			return m, tea.Quit
		}
		return m, nil

	case ErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	m.page, cmd = m.page.Update(msg)
	return m, cmd
}

func (m AppModel) View() tea.View {
	v := m.page.View()
	v.AltScreen = true
	return v
}

func (m AppModel) propagateSize() AppModel {
	if m.width < 1 || m.height < 1 {
		return m
	}
	m.page, _ = m.page.Update(resizeMsg{Width: m.width, Height: m.height})
	return m
}

func (m AppModel) switchToMainView() AppModel {
	m.state = StateMainView
	m.page = NewMainViewModel(m.app.DB, m.app.DBPath, "", m.width, m.height)
	if m.app.DB != nil && m.app.DB.Dirty() {
		mv := m.page.(MainViewModel)
		m.page = updateMainView(mv, dirtyMsg{Dirty: true})
	}
	m = m.propagateSize()
	return m
}

func updateMainView(mv MainViewModel, msg tea.Msg) MainViewModel {
	np, _ := mv.Update(msg)
	return np.(MainViewModel)
}
