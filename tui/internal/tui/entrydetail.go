package tui

import (
	"crypto/hmac"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/mattn/go-runewidth"
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
	entryPath    string
	db           *model.DB
	state        detailState
	editMode     editMode
	mode         detailMode
	attrList     []AttrInfo
	selectedAttr string
	selectedEntry *model.Entry
	keyInput     textinput.Model
	valueArea    textarea.Model
	editKey      string
	totpCode     string
	totpRemaining int
	totpDigits   int
	totpPeriod   int
	err          error
	pendingDeleteKey string
	copySuccess      bool
	width            int
	height           int
	focused          bool
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

func (m EntryDetailModel) refresh() EntryDetailModel {
	if m.selectedAttr == "" || m.db == nil {
		m.selectedEntry = nil
		return m
	}
	fullKey := m.entryPath + "/" + m.selectedAttr
	if e, ok := m.db.Get(fullKey); ok {
		m.selectedEntry = &e
		if m.selectedAttr == "TOTP" {
			m = m.updateTOTP()
		}
	} else {
		m.selectedEntry = nil
	}
	return m
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

type totpParams struct {
	secret    []byte
	digits    int
	period    int
	algorithm string
	steam     bool
}

func (m EntryDetailModel) updateTOTP() EntryDetailModel {
	if m.selectedEntry == nil {
		return m
	}

	raw := string(m.selectedEntry.Value)
	params, err := parseOtpAuthURI(raw)
	if err != nil {
		secret := strings.ToUpper(strings.ReplaceAll(raw, " ", ""))
		secret = strings.TrimRight(secret, "=")
		key, decErr := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
		if decErr != nil {
			m.totpCode = "invalid secret"
			m.totpRemaining = 0
			return m
		}
		params = totpParams{secret: key, digits: 6, period: 30, algorithm: "SHA1"}
	}

	m.totpDigits = params.digits
	m.totpPeriod = params.period

	ts := time.Now().Unix() / int64(params.period)
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(ts))

	h := hmac.New(newHash(params.algorithm), params.secret)
	h.Write(counter)
	hash := h.Sum(nil)

	offset := hash[len(hash)-1] & 0x0F
	fullCode := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF

	if params.steam {
		const steamChars = "23456789BCDFGHJKMNPQRTVWXY"
		code := fullCode
		var result strings.Builder
		for i := 0; i < 5; i++ {
			result.WriteByte(steamChars[code%uint32(len(steamChars))])
			code /= uint32(len(steamChars))
		}
		m.totpCode = result.String()
	} else {
		divisor := uint32(1)
		for i := 0; i < params.digits; i++ {
			divisor *= 10
		}
		code := fullCode % divisor
		fmtStr := fmt.Sprintf("%%0%dd", params.digits)
		m.totpCode = fmt.Sprintf(fmtStr, code)
	}

	m.totpRemaining = int(params.period - int(time.Now().Unix()%int64(params.period)))
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
					m = m.updateTOTP()
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
					if m.selectedAttr == "TOTP" {
						copyText = m.totpCode
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

	var b strings.Builder

	if m.state == detailEditKV {
		return tea.NewView(m.renderEditKVView(&b, width, height))
	}

	if m.state == detailConfirmDelete {
		return tea.NewView(m.renderConfirmDeleteView(&b, width, height))
	}

	if m.mode == detailModeAttrList {
		return tea.NewView(m.renderAttrListView(&b, width, height))
	}

	if m.selectedAttr == "" || m.selectedEntry == nil {
		b.WriteString(detailTitleStyle.Width(width - 4).Render("Detail"))
		b.WriteString("\n\n")
		b.WriteString(menuStyle.Render("Select an attribute"))
		return tea.NewView(m.wrapBorder(b.String(), width, height))
	}

	b.WriteString(detailTitleStyle.Width(width - 4).Render(m.selectedAttr))
	b.WriteString("\n\n")

	ts := time.UnixMilli(int64(m.selectedEntry.Timestamp)).Format("2006-01-02 15:04:05")
	b.WriteString(timestampStyle.Render(ts))
	b.WriteString("\n\n")

	if m.selectedAttr == "TOTP" {
		m.renderTOTPView(&b, width)
	} else {
		m.renderTextView(&b, width, height)
	}

	if m.copySuccess {
		b.WriteString("\n")
		b.WriteString(copySuccessStyle.Render("已复制到剪贴板"))
	}

	return tea.NewView(m.wrapBorder(b.String(), width, height))
}

func (m EntryDetailModel) renderTOTPView(b *strings.Builder, width int) {
	codeStyle := totpCodeStyle.Copy().Width(width - 4).Align(lipgloss.Center)
	b.WriteString(codeStyle.Render(m.totpCode))
	b.WriteString("\n\n")

	barWidth := width - 8
	if barWidth < 10 {
		barWidth = 10
	}
	period := m.totpPeriod
	if period < 1 {
		period = 30
	}
	filled := (m.totpRemaining * barWidth) / period
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	b.WriteString(lipgloss.NewStyle().Width(width - 4).Align(lipgloss.Center).Render(bar))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Width(width - 4).Align(lipgloss.Center).Render(fmt.Sprintf("%ds", m.totpRemaining)))
	b.WriteString("\n\n")

	maxWidth := width - 4
	if maxWidth < 10 {
		maxWidth = 10
	}
	for _, line := range wrapLine(string(m.selectedEntry.Value), maxWidth) {
		b.WriteString(menuStyle.Render(line))
		b.WriteString("\n")
	}
}

func (m EntryDetailModel) renderTextView(b *strings.Builder, width, height int) {
	value := string(m.selectedEntry.Value)
	maxWidth := width - 4
	if maxWidth < 10 {
		maxWidth = 10
	}

	var wrapped []string
	for _, line := range strings.Split(value, "\n") {
		wrapped = append(wrapped, wrapLine(line, maxWidth)...)
	}

	maxLines := height - 8
	if maxLines < 1 {
		maxLines = 1
	}

	for i, line := range wrapped {
		if i >= maxLines {
			b.WriteString(menuStyle.Render(fmt.Sprintf("... %d more lines", len(wrapped)-maxLines)))
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func (m EntryDetailModel) renderAttrListView(b *strings.Builder, width, height int) string {
	b.WriteString(detailTitleStyle.Width(width - 4).Render("Attributes"))
	b.WriteString("\n\n")

	if len(m.attrList) == 0 {
		b.WriteString(menuStyle.Render("  (no attributes)"))
		return m.wrapBorder(b.String(), width, height)
	}

	maxNameWidth := width - 16
	if maxNameWidth < 6 {
		maxNameWidth = 6
	}

	for _, attr := range m.attrList {
		name := attr.Name
		if runewidth.StringWidth(name) > maxNameWidth {
			name = truncateString(name, maxNameWidth-1) + "…"
		}
		ts := time.UnixMilli(int64(attr.Timestamp)).Format("2006-01-02 15:04")
		b.WriteString(panelAttrStyle.Render(name))
		b.WriteString("  ")
		b.WriteString(timestampStyle.Render(ts))
		b.WriteString("\n")
	}

	return m.wrapBorder(b.String(), width, height)
}


func (m EntryDetailModel) renderEditKVView(b *strings.Builder, width, height int) string {
	editW := width - 6
	if editW < 10 {
		editW = 10
	}

	if m.editMode == editModeNew {
		keyStyle := inputStyle.Width(editW)
		if m.keyInput.Focused() {
			keyStyle = keyStyle.BorderForeground(lipgloss.Color("#7C3AED"))
		}
		b.WriteString(keyStyle.Render(m.keyInput.View()))
		b.WriteString(detailTitleStyle.Width(width - 4).Render(""))
	} else {
		b.WriteString(detailTitleStyle.Width(width - 4).Render(m.selectedAttr))
		b.WriteString("\n\n")
		ts := time.UnixMilli(int64(m.selectedEntry.Timestamp)).Format("2006-01-02 15:04:05")
		b.WriteString(timestampStyle.Render(ts))
	}

	b.WriteString("\n\n")
	b.WriteString(m.valueArea.View())

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return m.wrapBorder(b.String(), width, height)
}

func (m EntryDetailModel) renderConfirmDeleteView(b *strings.Builder, width, height int) string {
	b.WriteString(errorStyle.Render("Confirm delete?"))
	b.WriteString("\n\n")
	b.WriteString(menuStyle.Render("[d/y] confirm  [any] cancel"))
	return m.wrapBorder(b.String(), width, height)
}

func (m EntryDetailModel) wrapBorder(content string, width, height int) string {
	style := blurBorderStyle
	if m.focused {
		style = focusBorderStyle
	}
	return style.Width(width - 2).Height(height - 2).Render(content)
}


