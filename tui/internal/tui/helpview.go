package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type HelpViewModel struct {
	active bool
	width  int
	height int
}

func NewHelpViewModel() HelpViewModel {
	return HelpViewModel{}
}

func (m HelpViewModel) Active() bool {
	return m.active
}

func (m *HelpViewModel) Toggle() {
	m.active = !m.active
}

func (m *HelpViewModel) Close() {
	m.active = false
}

func (m *HelpViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

var (
	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	helpSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF8C00"))

	helpBulletStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA"))

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	helpPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(1, 2)
)

func (m HelpViewModel) View() string {
	w := m.width
	h := m.height
	if w < 40 {
		w = 40
	}
	if h < 20 {
		h = 20
	}

	panelW := w * 3 / 5
	if panelW < 36 {
		panelW = 36
	}
	if panelW > 72 {
		panelW = 72
	}

	var b strings.Builder

	b.WriteString(helpTitleStyle.Render("tapass 帮助"))
	b.WriteString("\n\n")

	b.WriteString(helpSectionStyle.Render("── KV 规则 ──"))
	b.WriteString("\n")
	b.WriteString(helpBulletStyle.Render("• Key 以 / 分隔路径，最后一段为属性名"))
	b.WriteString("\n")
	b.WriteString(helpBulletStyle.Render("  例: /vault1/entry1/PASSWD"))
	b.WriteString("\n")
	b.WriteString(helpBulletStyle.Render("      路径 vault1/entry1，属性 PASSWD"))
	b.WriteString("\n")
	b.WriteString(helpBulletStyle.Render("• 特殊属性名（同名即生效）："))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n",
		helpKeyStyle.Render("PASSWD"),
		helpDescStyle.Render("密码属性（隐藏显示、一键复制）")))
	b.WriteString(fmt.Sprintf("  %s  %s\n",
		helpKeyStyle.Render("SSH"),
		helpDescStyle.Render("SSH 密钥对（公钥/私钥解析）")))
	b.WriteString(fmt.Sprintf("  %s  %s\n",
		helpKeyStyle.Render("TOTP"),
		helpDescStyle.Render("动态验证码（otpauth:// URI 或裸 secret）")))

	b.WriteString("\n\n")
	b.WriteString(helpSectionStyle.Render("── 快捷键 ──"))
	b.WriteString("\n")

	keys := []struct {
		key  string
		desc string
	}{
		{"j/k", "上下移动"},
		{"h", "返回上级 / 切换到左栏"},
		{"l", "进入分组 / 查看详情"},
		{"Tab", "切换左右栏"},
		{"n", "新建属性"},
		{"e", "编辑属性"},
		{"y", "复制属性值"},
		{"d", "删除属性（需二次确认）"},
		{"c", "数据库设置"},
		{"Ctrl+S", "保存"},
		{"/", "搜索"},
		{"q", "退出"},
		{"?", "帮助"},
	}

	maxKeyW := 0
	for _, k := range keys {
		w := lipgloss.Width(helpKeyStyle.Render(k.key))
		if w > maxKeyW {
			maxKeyW = w
		}
	}

	for _, k := range keys {
		keyRendered := helpKeyStyle.Render(k.key)
		padding := strings.Repeat(" ", maxKeyW-lipgloss.Width(keyRendered)+2)
		b.WriteString("  " + keyRendered + padding + helpDescStyle.Render(k.desc))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpBulletStyle.Render("按 Esc / ? / q 关闭帮助"))

	panel := helpPanelStyle.Width(panelW - 4).Render(b.String())

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, panel)
}
