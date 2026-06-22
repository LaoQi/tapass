package tui

type Renderer interface {
	View() string
}

func wrapBorder(content string, focused bool, width, height int) string {
	style := blurBorderStyle
	if focused {
		style = focusBorderStyle
	}
	return style.Width(width - 2).Height(height - 2).Render(content)
}
