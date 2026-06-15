package tui

import (
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

type AttrListView struct {
	attrs  []AttrInfo
	width  int
	height int
}

func (v *AttrListView) SetAttrs(attrs []AttrInfo) {
	v.attrs = attrs
}

func (v *AttrListView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *AttrListView) View() string {
	width := v.width
	if width < 1 {
		width = 30
	}

	var b strings.Builder
	b.WriteString(detailTitleStyle.Width(width - 4).Render("Attributes"))
	b.WriteString("\n\n")

	if len(v.attrs) == 0 {
		b.WriteString(menuStyle.Render("  (no attributes)"))
		return b.String()
	}

	maxNameWidth := width - 16
	if maxNameWidth < 6 {
		maxNameWidth = 6
	}

	for _, attr := range v.attrs {
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

	return b.String()
}
