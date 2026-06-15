package tui

import (
	"fmt"
	"strings"
)

type TextDetailView struct {
	value       string
	timestamp   uint64
	copySuccess bool
	width       int
	height      int
}

func (v *TextDetailView) SetValue(s string) {
	v.value = s
}

func (v *TextDetailView) SetTimestamp(ts uint64) {
	v.timestamp = ts
}

func (v *TextDetailView) SetCopySuccess(b bool) {
	v.copySuccess = b
}

func (v *TextDetailView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *TextDetailView) View() string {
	width := v.width
	if width < 1 {
		width = 30
	}
	height := v.height
	if height < 1 {
		height = 20
	}

	var b strings.Builder

	maxWidth := width - 4
	if maxWidth < 10 {
		maxWidth = 10
	}

	var wrapped []string
	for _, line := range strings.Split(v.value, "\n") {
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

	return b.String()
}
