package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func keyEnter() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }
func keyEsc() tea.KeyPressMsg   { return tea.KeyPressMsg{Code: tea.KeyEscape} }
func keyRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// 新建 vault 时目标文件已存在 → 必须先确认覆盖，不能直接进入密码输入。
func TestWelcomeCreateExistingFileRequiresConfirm(t *testing.T) {
	if got := keyEnter().String(); got != "enter" {
		t.Fatalf("unexpected key string %q", got)
	}
	if got := keyRune('y').String(); got != "y" {
		t.Fatalf("unexpected key string %q", got)
	}

	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.tap")
	if err := os.WriteFile(existing, []byte("old vault"), 0600); err != nil {
		t.Fatal(err)
	}

	m := NewWelcomeModel()
	m = updateWelcome(m, keyRune('n')) // 进入"新建"
	if m.state != WelcomeNewPath {
		t.Fatalf("expected WelcomeNewPath, got %d", m.state)
	}
	m.pathInput.SetValue(existing)

	m = updateWelcome(m, keyEnter())
	if m.state != WelcomeConfirmOverwrite {
		t.Fatalf("expected WelcomeConfirmOverwrite for existing file, got %d", m.state)
	}
	if view := m.View().Content; !strings.Contains(view, "File already exists") {
		t.Errorf("expected overwrite warning in view, got:\n%s", view)
	}

	// n → 返回路径输入，不进入密码输入
	m = updateWelcome(m, keyRune('n'))
	if m.state != WelcomeNewPath {
		t.Errorf("expected back to WelcomeNewPath, got %d", m.state)
	}

	// y → 进入密码输入（覆盖）
	m = updateWelcome(m, keyEnter())
	if m.state != WelcomeConfirmOverwrite {
		t.Fatalf("expected confirm again, got %d", m.state)
	}
	m = updateWelcome(m, keyRune('y'))
	if m.state != WelcomeNewPassword {
		t.Errorf("expected WelcomeNewPassword after confirm, got %d", m.state)
	}

	// 确认前原文件必须未被改动
	content, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "old vault" {
		t.Errorf("existing file modified before confirmation: %q", string(content))
	}
}

// 目标不存在 → 直接进入密码输入，无额外确认。
func TestWelcomeCreateNewFileNoConfirm(t *testing.T) {
	m := NewWelcomeModel()
	m = updateWelcome(m, keyRune('n'))
	m.pathInput.SetValue(filepath.Join(t.TempDir(), "new.tap"))

	m = updateWelcome(m, keyEnter())
	if m.state != WelcomeNewPassword {
		t.Fatalf("expected WelcomeNewPassword, got %d", m.state)
	}
}

// 空路径与目录路径必须报错，且不进入密码输入。
func TestWelcomeCreateInvalidPath(t *testing.T) {
	m := NewWelcomeModel()
	m = updateWelcome(m, keyRune('n'))
	m.pathInput.SetValue("   ")
	m = updateWelcome(m, keyEnter())
	if m.state != WelcomeNewPath {
		t.Errorf("expected to stay on WelcomeNewPath for empty path, got %d", m.state)
	}
	if m.err == nil {
		t.Error("expected error for empty path")
	}

	m2 := NewWelcomeModel()
	m2 = updateWelcome(m2, keyRune('n'))
	m2.pathInput.SetValue(t.TempDir()) // 目录
	m2 = updateWelcome(m2, keyEnter())
	if m2.state != WelcomeNewPath {
		t.Errorf("expected to stay on WelcomeNewPath for directory path, got %d", m2.state)
	}
	if m2.err == nil || !strings.Contains(m2.err.Error(), "directory") {
		t.Errorf("expected directory error, got %v", m2.err)
	}
}

// esc 在确认界面返回路径输入。
func TestWelcomeConfirmOverwriteEsc(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.tap")
	if err := os.WriteFile(existing, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	m := NewWelcomeModel()
	m = updateWelcome(m, keyRune('n'))
	m.pathInput.SetValue(existing)
	m = updateWelcome(m, keyEnter())
	m = updateWelcome(m, keyEsc())
	if m.state != WelcomeNewPath {
		t.Fatalf("expected WelcomeNewPath after esc, got %d", m.state)
	}
}

func updateWelcome(m WelcomeModel, msg tea.Msg) WelcomeModel {
	np, _ := m.Update(msg)
	return np.(WelcomeModel)
}
