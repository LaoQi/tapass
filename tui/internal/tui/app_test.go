package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaoQi/tapass/tui/internal/model"
)

func newTestAppWithDB(t *testing.T, db *model.DB) AppModel {
	t.Helper()

	app := NewApp("")
	app.app.DB = db
	app.app.DBPath = db.Path()
	app.state = StateMainView
	app.page = NewMainViewModel(db, db.Path(), "", 100, 30)
	app.width, app.height = 100, 30
	return app
}

// 回归：保存失败必须显示给用户（此前 AppModel.err 只赋值、从不渲染，失败完全静默）。
func TestAppShowsSaveErrorInStatusBar(t *testing.T) {
	// 目标目录不存在 → Save 必然失败
	db, err := model.CreateDB(filepath.Join(t.TempDir(), "missing", "v.tap"), "pw")
	if err != nil {
		t.Fatalf("CreateDB failed: %v", err)
	}
	app := newTestAppWithDB(t, db)

	np, cmd := app.Update(SaveVaultMsg{})
	if cmd == nil {
		t.Fatal("expected a command from SaveVaultMsg")
	}
	app = np.(AppModel)

	msg := cmd()
	errMsg, ok := msg.(ErrorMsg)
	if !ok {
		t.Fatalf("expected ErrorMsg from failed save, got %T", msg)
	}

	np, tick := app.Update(errMsg)
	app = np.(AppModel)
	if tick == nil {
		t.Error("expected auto-clear tick command after showing error")
	}

	view := app.View().Content
	if !strings.Contains(view, "[!]") {
		t.Errorf("expected error indicator in status bar, got:\n%s", view)
	}
	if !strings.Contains(view, "missing/v.tap") {
		t.Errorf("expected error detail (path) in status bar, got:\n%s", view)
	}

	// clearErrorMsg 之后错误消失，恢复常规状态栏
	np, _ = app.Update(clearErrorMsg{})
	view = np.(AppModel).View().Content
	if strings.Contains(view, "[!]") {
		t.Errorf("expected error to be cleared, got:\n%s", view)
	}
	if !strings.Contains(view, "[q] quit") {
		t.Errorf("expected normal status bar after clear, got:\n%s", view)
	}
}

// 复制失败必须如实提示（此前忽略 clipboard 错误、无条件显示"已复制到剪贴板"）。
func TestCopyFailureIsReported(t *testing.T) {
	m := newTestMainView(t, map[string]string{
		"/grp/entry/PASSWD": "secret",
	}).enterPrefix("/grp/entry")

	m.rightPanel.copyErr = "no clipboard utilities available"
	view := m.rightPanel.View().Content
	if !strings.Contains(view, "复制失败") {
		t.Errorf("expected copy failure notice, got:\n%s", view)
	}
	if strings.Contains(view, "已复制到剪贴板") {
		t.Errorf("must not claim success when clipboard write failed:\n%s", view)
	}

	// 成功后不再显示失败提示
	m.rightPanel = m.rightPanel.clearCopyState()
	m.rightPanel.copySuccess = true
	view = m.rightPanel.View().Content
	if !strings.Contains(view, "已复制到剪贴板") {
		t.Errorf("expected success notice, got:\n%s", view)
	}
	if strings.Contains(view, "复制失败") {
		t.Errorf("stale copy error leaked into success state:\n%s", view)
	}
}
