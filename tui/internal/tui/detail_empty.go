package tui

import (
	"strings"
)

type EmptyDetailView struct {
	width  int
	height int
}

func (v *EmptyDetailView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *EmptyDetailView) View() string {
	width := v.width
	if width < 1 {
		width = 30
	}

	var b strings.Builder
	b.WriteString(detailTitleStyle.Width(width - 4).Render("Detail"))
	b.WriteString("\n\n")
	b.WriteString(menuStyle.Render("Select an attribute"))
	return b.String()
}
