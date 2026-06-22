package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

type EditKVView struct {
	editMode     editMode
	keyInput     textinput.Model
	valueArea    textarea.Model
	selectedAttr string
	timestamp    uint64
	err          error
	width        int
	height       int
}

func (v *EditKVView) SetEditMode(m editMode) {
	v.editMode = m
}

func (v *EditKVView) SetKeyInput(k textinput.Model) {
	v.keyInput = k
}

func (v *EditKVView) SetValueArea(a textarea.Model) {
	v.valueArea = a
}

func (v *EditKVView) SetSelectedAttr(s string) {
	v.selectedAttr = s
}

func (v *EditKVView) SetTimestamp(ts uint64) {
	v.timestamp = ts
}

func (v *EditKVView) SetError(e error) {
	v.err = e
}

func (v *EditKVView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *EditKVView) View() string {
	width := v.width
	if width < 1 {
		width = 30
	}
	height := v.height
	if height < 1 {
		height = 20
	}

	var b strings.Builder

	editW := width - 6
	if editW < 10 {
		editW = 10
	}

	if v.editMode == editModeNew {
		keyStyle := inputStyle.Width(editW)
		if v.keyInput.Focused() {
			keyStyle = keyStyle.BorderForeground(lipgloss.Color("#7C3AED"))
		}
		b.WriteString(keyStyle.Render(v.keyInput.View()))
		b.WriteString(detailTitleStyle.Width(width - 4).Render(""))
	} else {
		b.WriteString(detailTitleStyle.Width(width - 4).Render(v.selectedAttr))
		b.WriteString("\n\n")
		ts := time.UnixMilli(int64(v.timestamp)).Format("2006-01-02 15:04:05")
		b.WriteString(timestampStyle.Render(ts))
	}

	b.WriteString("\n\n")
	b.WriteString(v.valueArea.View())

	if v.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", v.err)))
	}

	return b.String()
}
