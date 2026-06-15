package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"github.com/atotto/clipboard"
	"github.com/LaoQi/tapass/tui/internal/model"
)

type detailState int

const (
	detailView detailState = iota
	detailEditKV
	detailConfirmDelete
)

type editMode int

const (
	editModeNew editMode = iota
	editModeEdit
)

type detailMode int

const (
	detailModeAttrList detailMode = iota
	detailModeDetail
)

type AttrInfo struct {
	Name      string
	Timestamp uint64
}

type EntryDetailModel struct {
	entryPath       string
	db              *model.DB
	state           detailState
	editMode        editMode
	mode            detailMode
	attrList        []AttrInfo
	selectedAttr    string
	selectedEntry   *model.Entry
	keyInput        textinput.Model
	valueArea       textarea.Model
	editKey         string
	err             error
	pendingDeleteKey string
	copySuccess     bool
	width           int
	height          int
	focused         bool
	totpView        *TOTPDetailView
}

func NewEntryDetailModel(entryPath string, db *model.DB) EntryDetailModel {
	keyInput := textinput.New()
	keyInput.CharLimit = 512

	valueArea := textarea.New()
	valueArea.CharLimit = 65536
	valueArea.ShowLineNumbers = false
	valueArea.SetWidth(40)
	valueArea.SetHeight(5)

	m := EntryDetailModel{
		db:        db,
		state:     detailView,
		mode:      detailModeDetail,
		keyInput:  keyInput,
		valueArea: valueArea,
	}
	if entryPath != "" {
		m.entryPath = entryPath
	}
	return m
}

func (m EntryDetailModel) EntryPath() string {
	return m.entryPath
}

func (m EntryDetailModel) Init() tea.Cmd {
	if m.selectedAttr == "TOTP" && m.selectedEntry != nil {
		return tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg{}
		})
	}
	return nil
}

func (m EntryDetailModel) State() detailState {
	return m.state
}

func (m EntryDetailModel) HasSelectedEntry() bool {
	return m.selectedEntry != nil
}

func (m EntryDetailModel) IsTOTP() bool {
	return m.selectedAttr == "TOTP"
}

func (m EntryDetailModel) TOTPCode() string {
	if m.totpView != nil {
		return m.totpView.Code()
	}
	return ""
}

func (m EntryDetailModel) newTOTPView(value string) *TOTPDetailView {
	v := &TOTPDetailView{}
	v.SetValue(value)
	v.SetSize(m.width, m.height)
	v.ComputeCode()
	return v
}

func (m EntryDetailModel) refresh() EntryDetailModel {
	if m.selectedAttr == "" || m.db == nil {
		m.selectedEntry = nil
		return m
	}
	fullKey := m.entryPath + "/" + m.selectedAttr
	if e, ok := m.db.Get(fullKey); ok {
		m.selectedEntry = &e
		if m.selectedAttr == "TOTP" {
			m.totpView = m.newTOTPView(string(e.Value))
		}
	} else {
		m.selectedEntry = nil
	}
	return m
}

func (m EntryDetailModel) updateTOTP() EntryDetailModel {
	if m.selectedEntry == nil {
		return m
	}
	m.totpView = m.newTOTPView(string(m.selectedEntry.Value))
	return m
}

func (m EntryDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case syncRightMsg:
		if msg.ClearOnly {
			m.entryPath = ""
			m.selectedAttr = ""
			m.selectedEntry = nil
			m.state = detailView
			m.copySuccess = false
			m.mode = detailModeAttrList
			m.attrList = nil
			m.totpView = nil
			return m, nil
		}
		if msg.EntryPath != "" {
			m.entryPath = msg.EntryPath
			m.selectedAttr = msg.SelectedAttr
			m.selectedEntry = nil
			m.copySuccess = false
			if m.db != nil && msg.SelectedAttr != "" {
				fullKey := msg.EntryPath + "/" + msg.SelectedAttr
				if e, ok := m.db.Get(fullKey); ok {
					m.selectedEntry = &e
					if msg.SelectedAttr == "TOTP" {
						m.totpView = m.newTOTPView(string(e.Value))
					}
				}
			}
		} else {
			m.entryPath = ""
			m.selectedAttr = ""
			m.selectedEntry = nil
			m.copySuccess = false
			if msg.SetDetailMode {
				m.mode = detailModeAttrList
				m.attrList = msg.Attrs
			}
		}
		if msg.SetDetailMode {
			m.mode = detailModeDetail
			m.attrList = nil
		}
		return m, nil
	case dbEventMsg:
		if m.db == nil {
			return m, nil
		}
		switch msg.Event.Type {
		case model.EventAttrSet:
			if msg.Event.Key == m.entryPath+"/"+m.selectedAttr {
				m = m.refresh()
			}
		case model.EventAttrDeleted:
			if msg.Event.Key == m.entryPath+"/"+m.selectedAttr {
				m.selectedEntry = nil
				m.selectedAttr = ""
			}
		}
		return m, nil
	case refreshMsg:
		m = m.refresh()
		return m, nil
	case startNewMsg:
		m.state = detailEditKV
		m.editMode = editModeNew
		m.editKey = ""
		prefix := msg.Prefix
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		m.keyInput.SetValue(prefix)
		m.keyInput.CursorEnd()
		m.keyInput.Focus()
		m.valueArea.SetValue("")
		m.valueArea.Blur()
		m.err = nil
		m.copySuccess = false
		return m, nil
	case refreshTOTPMsg:
		if m.selectedAttr == "TOTP" && m.selectedEntry != nil {
			m = m.updateTOTP()
		}
		return m, nil
	case resizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == detailEditKV {
			m = m.resizeEditor()
		}
		if m.totpView != nil {
			m.totpView.SetSize(m.width, m.height)
		}
		return m, nil
	case setFocusMsg:
		m.focused = msg.Focused
		return m, nil
	case tickMsg:
		if m.selectedAttr == "TOTP" && m.selectedEntry != nil {
			m = m.updateTOTP()
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return tickMsg{}
			})
		}

	case copyClearMsg:
		m.copySuccess = false

	case tea.KeyPressMsg:
		m.err = nil
		switch m.state {
		case detailView:
			switch msg.String() {
		case "e":
			if m.selectedEntry != nil {
				m.state = detailEditKV
				m.editMode = editModeEdit
				m.editKey = m.entryPath + "/" + m.selectedAttr
				m.keyInput.SetValue(m.editKey)
				m.keyInput.CursorEnd()
				m.keyInput.Blur()
				m.valueArea.SetValue(string(m.selectedEntry.Value))
				m.valueArea.Focus()
				m.valueArea.CursorEnd()
				m.err = nil
				m.copySuccess = false
				m = m.resizeEditor()
				return m, nil
			}
			case "y":
				if m.selectedEntry != nil {
					var copyText string
					if m.selectedAttr == "TOTP" && m.totpView != nil {
						copyText = m.totpView.Code()
					} else {
						copyText = string(m.selectedEntry.Value)
					}
					_ = clipboard.WriteAll(copyText)
					m.copySuccess = true
					return m, tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
						return copyClearMsg{}
					})
				}
			case "d":
				if m.selectedEntry != nil && m.db != nil {
					m.pendingDeleteKey = m.entryPath + "/" + m.selectedAttr
					m.state = detailConfirmDelete
					return m, nil
				}
			}

		case detailEditKV:
			if msg.String() == "alt+s" {
				return m.saveKV()
			}
			if msg.String() == "esc" {
				m.state = detailView
				m.keyInput.Blur()
				m.valueArea.Blur()
				return m, nil
			}
			if msg.String() == "tab" {
				if m.keyInput.Focused() {
					m.keyInput.Blur()
					m.valueArea.Focus()
				} else {
					m.valueArea.Blur()
					m.keyInput.Focus()
				}
				return m, nil
			}

			if m.keyInput.Focused() {
				m.keyInput, cmd = m.keyInput.Update(msg)
				return m, cmd
			}
			m.valueArea, cmd = m.valueArea.Update(msg)
			return m, cmd

		case detailConfirmDelete:
			switch msg.String() {
			case "d", "y":
				if m.db != nil && m.pendingDeleteKey != "" {
					fullKey := m.pendingDeleteKey
					cmds := m.db.Delete(fullKey)
					m.selectedEntry = nil
					m.selectedAttr = ""
					m.pendingDeleteKey = ""
					m.state = detailView
					return m, tea.Batch(append([]tea.Cmd{func() tea.Msg { return AttrChangedMsg{Key: fullKey} }}, cmds...)...)
				}
				m.pendingDeleteKey = ""
				m.state = detailView
				return m, nil
			default:
				m.pendingDeleteKey = ""
				m.state = detailView
				return m, nil
			}
		}
	}

	return m, cmd
}

func (m EntryDetailModel) saveKV() (EntryDetailModel, tea.Cmd) {
	if m.db == nil {
		m.state = detailView
		return m, nil
	}

	var fullKey string
	if m.editMode == editModeEdit {
		fullKey = m.editKey
	} else {
		fullKey = m.keyInput.Value()
		if fullKey == "" || fullKey == "/" {
			m.err = fmt.Errorf("key cannot be empty")
			return m, nil
		}
		if !strings.HasPrefix(fullKey, "/") {
			fullKey = "/" + fullKey
		}
	}

	value := m.valueArea.Value()
	cmds := m.db.Set(fullKey, []byte(value))

	if m.editMode == editModeNew {
		parent := model.ParentPath(fullKey)
		attrName := fullKey[strings.LastIndex(fullKey, "/")+1:]
		m.entryPath = parent
		m.selectedAttr = attrName
		m.state = detailView
		m.keyInput.Blur()
		m.valueArea.Blur()
		m = m.refresh()
		return m, tea.Batch(append([]tea.Cmd{func() tea.Msg { return AttrChangedMsg{Key: fullKey} }}, cmds...)...)
	}

	m.state = detailView
	m.keyInput.Blur()
	m.valueArea.Blur()
	m = m.refresh()
	return m, tea.Batch(append([]tea.Cmd{func() tea.Msg { return AttrChangedMsg{Key: fullKey} }}, cmds...)...)
}

func (m EntryDetailModel) resizeEditor() EntryDetailModel {
	w := m.width
	if w < 1 {
		w = 30
	}
	h := m.height
	if h < 1 {
		h = 20
	}
	editW := w - 6
	if editW < 10 {
		editW = 10
	}
	editH := h - 8
	if m.editMode == editModeEdit {
		editH = h - 10
	}
	if editH < 1 {
		editH = 1
	}
	m.valueArea.SetWidth(editW)
	m.valueArea.SetHeight(editH)
	m.keyInput.SetWidth(editW)
	return m
}

func (m EntryDetailModel) View() tea.View {
	width := m.width
	if width < 1 {
		width = 30
	}
	height := m.height
	if height < 1 {
		height = 20
	}

	var content string

	switch {
	case m.state == detailEditKV:
		v := &EditKVView{}
		v.SetEditMode(m.editMode)
		v.SetKeyInput(m.keyInput)
		v.SetValueArea(m.valueArea)
		v.SetSelectedAttr(m.selectedAttr)
		if m.selectedEntry != nil {
			v.SetTimestamp(m.selectedEntry.Timestamp)
		}
		v.SetError(m.err)
		v.SetSize(width, height)
		content = v.View()

	case m.state == detailConfirmDelete:
		var b strings.Builder
		b.WriteString(errorStyle.Render("Confirm delete?"))
		b.WriteString("\n\n")
		b.WriteString(menuStyle.Render("[d/y] confirm  [any] cancel"))
		content = b.String()

	case m.mode == detailModeAttrList:
		v := &AttrListView{}
		v.SetAttrs(m.attrList)
		v.SetSize(width, height)
		content = v.View()

	case m.selectedAttr == "" || m.selectedEntry == nil:
		v := &EmptyDetailView{}
		v.SetSize(width, height)
		content = v.View()

	case m.selectedAttr == "TOTP":
		m.totpView.SetTimestamp(m.selectedEntry.Timestamp)
		m.totpView.SetCopySuccess(m.copySuccess)
		m.totpView.SetSize(width, height)
		content = m.totpView.View()

	default:
		v := &TextDetailView{}
		v.SetValue(string(m.selectedEntry.Value))
		v.SetTimestamp(m.selectedEntry.Timestamp)
		v.SetCopySuccess(m.copySuccess)
		v.SetSize(width, height)
		content = v.View()
	}

	if m.selectedAttr != "" && m.selectedAttr != "TOTP" && m.selectedEntry != nil && m.state == detailView && m.copySuccess {
		content += "\n" + copySuccessStyle.Render("已复制到剪贴板")
	}

	if m.selectedAttr == "TOTP" && m.selectedEntry != nil && m.state == detailView && m.copySuccess {
		content += "\n" + copySuccessStyle.Render("已复制到剪贴板")
	}

	return tea.NewView(wrapBorder(content, m.focused, width, height))
}
