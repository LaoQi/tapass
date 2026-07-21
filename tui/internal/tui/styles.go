package tui

import "charm.land/lipgloss/v2"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED")).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("#374151"))

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6B7280")).
			Padding(0, 1).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	menuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	focusBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF8C00"))

	blurBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151"))

	totpCodeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#34D399"))

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	keyEnabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	keyDisabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B5563"))

	copySuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#34D399")).
				Bold(true)

	passGenCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	passGenLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF"))

	passGenValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#34D399")).
				Bold(true)
)
