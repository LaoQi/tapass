package tui

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/tapass/tapass-tools/vault"
)

type detailState int

const (
	detailView detailState = iota
	detailEditAttr
	detailAddAttr
)

type tickMsg struct{}

type EntryDetailModel struct {
	entryPath     string
	entries       map[string]vault.Entry
	v             *vault.Vault
	state         detailState
	selectedAttr  string
	selectedEntry *vault.Entry
	editInput     textinput.Model
	editKey       string
	newKeyInput   textinput.Model
	totpCode      string
	totpRemaining int
	totpDigits    int
	totpPeriod    int
	err           error
	width         int
	height        int
	focused       bool
}

func NewEntryDetailModel(entryPath string, entries map[string]vault.Entry, v *vault.Vault) EntryDetailModel {
	editInput := textinput.New()
	editInput.CharLimit = 4096

	newKeyInput := textinput.New()
	newKeyInput.Placeholder = "attribute name"
	newKeyInput.CharLimit = 256

	return EntryDetailModel{
		entryPath:  entryPath,
		entries:    entries,
		v:          v,
		state:      detailView,
		editInput:  editInput,
		newKeyInput: newKeyInput,
	}
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

func (m *EntryDetailModel) SelectAttr(name string) {
	m.selectedAttr = name
	m.selectedEntry = nil

	if m.entryPath == "" || m.entries == nil || name == "" {
		return
	}

	fullKey := m.entryPath + "/" + name
	if e, ok := m.entries[fullKey]; ok {
		m.selectedEntry = &e
		if name == "TOTP" {
			m.updateTOTP()
		}
	}
}

type totpParams struct {
	secret    []byte
	digits    int
	period    int
	algorithm string
	steam     bool
}

func parseOtpAuthURI(raw string) (totpParams, error) {
	p := totpParams{digits: 6, period: 30, algorithm: "SHA1"}

	if !strings.HasPrefix(raw, "otpauth://totp/") {
		return p, fmt.Errorf("not an otpauth://totp/ URI")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return p, fmt.Errorf("parse URI: %w", err)
	}

	q := u.Query()

	secretStr := q.Get("secret")
	if secretStr == "" {
		return p, fmt.Errorf("missing secret parameter")
	}
	secretUpper := strings.ToUpper(strings.ReplaceAll(secretStr, " ", ""))
	p.secret, err = base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(secretUpper, "="))
	if err != nil {
		return p, fmt.Errorf("decode secret: %w", err)
	}

	if d := q.Get("digits"); d != "" {
		if strings.EqualFold(d, "S") {
			p.steam = true
			p.digits = 5
		} else if v, err := strconv.Atoi(d); err == nil && v > 0 {
			p.digits = v
		}
	}

	if pr := q.Get("period"); pr != "" {
		if v, err := strconv.Atoi(pr); err == nil && v > 0 {
			p.period = v
		}
	}

	if alg := q.Get("algorithm"); alg != "" {
		p.algorithm = strings.ToUpper(alg)
	}

	return p, nil
}

func newHash(algorithm string) func() hash.Hash {
	switch algorithm {
	case "SHA256":
		return sha256.New
	case "SHA512":
		return sha512.New
	default:
		return sha1.New
	}
}

func (m *EntryDetailModel) updateTOTP() {
	if m.selectedEntry == nil {
		return
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
			return
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
}

func (m EntryDetailModel) Update(msg tea.Msg) (EntryDetailModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if m.selectedAttr == "TOTP" && m.selectedEntry != nil {
			m.updateTOTP()
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return tickMsg{}
			})
		}

	case tea.KeyMsg:
		m.err = nil
		switch m.state {
		case detailView:
			switch msg.String() {
			case "e":
				if m.selectedEntry != nil {
					m.state = detailEditAttr
					m.editKey = m.selectedAttr
					m.editInput.SetValue(string(m.selectedEntry.Value))
					m.editInput.Focus()
					return m, nil
				}
			case "a":
				m.state = detailAddAttr
				m.newKeyInput.SetValue("")
				m.newKeyInput.Focus()
				return m, nil
			case "d":
				if m.selectedEntry != nil && m.v != nil {
					fullKey := m.entryPath + "/" + m.selectedAttr
					m.v.Delete(fullKey)
					m.selectedEntry = nil
					m.selectedAttr = ""
					return m, func() tea.Msg { return EntryUpdatedMsg{} }
				}
			}

		case detailEditAttr:
			switch msg.String() {
			case "enter":
				if m.v != nil {
					fullKey := m.entryPath + "/" + m.editKey
					m.v.Set(fullKey, []byte(m.editInput.Value()))
					if m.selectedAttr == m.editKey {
						m.SelectAttr(m.editKey)
					}
				}
				m.state = detailView
				m.editInput.Blur()
				return m, func() tea.Msg { return EntryUpdatedMsg{} }
			case "esc":
				m.state = detailView
				m.editInput.Blur()
				return m, nil
			}
			m.editInput, cmd = m.editInput.Update(msg)
			return m, cmd

		case detailAddAttr:
			switch msg.String() {
			case "enter":
				key := m.newKeyInput.Value()
				if key == "" {
					m.err = fmt.Errorf("attribute name cannot be empty")
					return m, nil
				}
				m.state = detailEditAttr
				m.editKey = key
				m.editInput.SetValue("")
				m.editInput.Focus()
				return m, nil
			case "esc":
				m.state = detailView
				m.newKeyInput.Blur()
				return m, nil
			}
			m.newKeyInput, cmd = m.newKeyInput.Update(msg)
			return m, cmd
		}
	}

	return m, cmd
}

func (m EntryDetailModel) View() string {
	width := m.width
	if width < 1 {
		width = 30
	}
	height := m.height
	if height < 1 {
		height = 20
	}

	var b strings.Builder

	if m.state != detailView {
		return m.renderEditView(&b, width, height)
	}

	if m.selectedAttr == "" || m.selectedEntry == nil {
		b.WriteString(detailTitleStyle.Render("Detail"))
		b.WriteString("\n\n")
		b.WriteString(menuStyle.Render("Select an attribute"))
		return m.wrapBorder(b.String(), width, height)
	}

	b.WriteString(detailTitleStyle.Render(m.selectedAttr))
	b.WriteString("\n\n")

	ts := time.UnixMilli(int64(m.selectedEntry.Timestamp)).Format("2006-01-02 15:04:05")
	b.WriteString(timestampStyle.Render(ts))
	b.WriteString("\n\n")

	if m.selectedAttr == "TOTP" {
		m.renderTOTPView(&b, width)
	} else {
		m.renderTextView(&b, width, height)
	}

	return m.wrapBorder(b.String(), width, height)
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

func wrapLine(s string, width int) []string {
	if len(s) <= width {
		return []string{s}
	}
	var lines []string
	for len(s) > width {
		lines = append(lines, s[:width])
		s = s[width:]
	}
	if s != "" {
		lines = append(lines, s)
	}
	return lines
}

func (m EntryDetailModel) renderEditView(b *strings.Builder, width, height int) string {
	switch m.state {
	case detailEditAttr:
		b.WriteString(fmt.Sprintf("Editing: %s\n\n", m.editKey))
		editWidth := width - 4
		if editWidth < 10 {
			editWidth = 10
		}
		b.WriteString(inputStyle.Width(editWidth).Render(m.editInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[enter] save  [esc] cancel"))

	case detailAddAttr:
		b.WriteString("New attribute name:\n\n")
		editWidth := width - 4
		if editWidth < 10 {
			editWidth = 10
		}
		b.WriteString(inputStyle.Width(editWidth).Render(m.newKeyInput.View()))
		b.WriteString("\n")
		b.WriteString(menuStyle.Render("[enter] next  [esc] cancel"))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return m.wrapBorder(b.String(), width, height)
}

func (m EntryDetailModel) wrapBorder(content string, width, height int) string {
	style := blurBorderStyle
	if m.focused {
		style = focusBorderStyle
	}
	return style.Width(width - 2).Height(height - 2).Render(content)
}

func (m *EntryDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *EntryDetailModel) SetFocused(f bool) {
	m.focused = f
}

func (m *EntryDetailModel) RefreshFromEntries(entries map[string]vault.Entry) {
	m.entries = entries
	if m.selectedAttr != "" {
		m.SelectAttr(m.selectedAttr)
	}
}
