package tui

import (
	"fmt"
	"strings"
)

type PassGenView struct {
	state *PassGenState
}

func (v *PassGenView) SetState(s *PassGenState) {
	v.state = s
}

func (v *PassGenView) View() string {
	if v.state == nil {
		return ""
	}

	s := v.state
	width := s.width
	if width < 1 {
		width = 30
	}

	maxContentW := width - 4
	if maxContentW < 10 {
		maxContentW = 10
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Width(width - 4).Render("Password Generator"))
	b.WriteString("\n\n")

	type ruleRow struct {
		label   string
		value   string
		isBool  bool
		boolVal bool
	}

	rows := []ruleRow{
		{label: "Length", value: fmt.Sprintf("%d", s.rules.Length), isBool: false},
		{label: "Uppercase", value: "A-Z", isBool: true, boolVal: s.rules.Uppercase},
		{label: "Lowercase", value: "a-z", isBool: true, boolVal: s.rules.Lowercase},
		{label: "Digits", value: "0-9", isBool: true, boolVal: s.rules.Digits},
		{label: "Symbols", value: "!@#$%...", isBool: true, boolVal: s.rules.Symbols},
		{label: "Exclude ambiguous", value: "0Oo1lI", isBool: true, boolVal: s.rules.ExcludeAmbiguous},
	}

	for i, row := range rows {
		var line string
		if row.isBool {
			check := "[ ]"
			if row.boolVal {
				check = "[x]"
			}
			line = fmt.Sprintf("  %s %s  %s", check, row.label, menuStyle.Render(row.value))
		} else {
			line = fmt.Sprintf("  Length: %d  [+/-]", s.rules.Length)
		}

		if i == s.cursor {
			line = passGenCursorStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if s.generated != "" {
		b.WriteString(passGenLabelStyle.Render("Generated:"))
		b.WriteString("\n")
		wrapped := wrapLine(s.generated, maxContentW)
		for _, wl := range wrapped {
			b.WriteString(passGenValueStyle.Render(wl))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(menuStyle.Render("Press [g] to generate"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(menuStyle.Render("[j/k] navigate  [space] toggle  [g] generate"))
	b.WriteString("\n")
	b.WriteString(menuStyle.Render("[enter] apply  [esc] cancel"))

	return b.String()
}
