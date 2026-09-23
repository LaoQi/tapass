package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaoQi/tapass/tui/internal/model"
)

func newTestMainView(t *testing.T, entries map[string]string) MainViewModel {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.tap")
	db, err := model.CreateDB(dbPath, "test-password")
	if err != nil {
		t.Fatalf("CreateDB failed: %v", err)
	}
	for k, v := range entries {
		db.Set(k, []byte(v))
	}

	m := NewMainViewModel(db, dbPath, "", 100, 30)
	return updateMainView(m, resizeMsg{Width: 100, Height: 30})
}

func (m MainViewModel) enterPrefix(prefix string) MainViewModel {
	m.leftPanel = updateLeft(m.leftPanel, setPrefixMsg{Prefix: prefix})
	return m.syncRightFromLeft()
}

// 回归：左栏选中的是分组/条目（Depth>0）时，右栏必须显示属性列表，
// 而不是被覆盖成空的详情视图。
func TestRightPanelShowsAttrListForGroup(t *testing.T) {
	m := newTestMainView(t, map[string]string{
		"/grp/entry/PASSWD":   "secret",
		"/grp/entry/username": "alice",
	})

	if it := m.leftPanel.SelectedItem(); it.Depth == 0 {
		t.Fatalf("expected a group item, got %+v", it)
	}
	rootView := m.rightPanel.View().Content
	if !strings.Contains(rootView, "Attributes") {
		t.Errorf("root: expected attribute list view, got:\n%s", rootView)
	}
	if m.rightPanel.mode != detailModeAttrList {
		t.Errorf("root: expected detailModeAttrList, got %d", m.rightPanel.mode)
	}

	// 进入 /grp：选中条目 entry，右栏显示该条目的属性列表
	m = m.enterPrefix("/grp")
	if it := m.leftPanel.SelectedItem(); it.FullPath != "/grp/entry" {
		t.Fatalf("expected /grp/entry selected, got %+v", it)
	}
	view := m.rightPanel.View().Content
	if !strings.Contains(view, "Attributes") {
		t.Errorf("group selected: expected attribute list view, got:\n%s", view)
	}
	if !strings.Contains(view, "PASSWD") || !strings.Contains(view, "username") {
		t.Errorf("group selected: expected attribute names in list, got:\n%s", view)
	}
	if got := m.rightPanel.EntryPath(); got != "/grp/entry" {
		t.Errorf("expected entry path /grp/entry in right panel, got %q", got)
	}
}

// 左栏选中的是属性（Depth==0）时，右栏显示该属性详情。
func TestRightPanelShowsAttrDetail(t *testing.T) {
	m := newTestMainView(t, map[string]string{
		"/grp/entry/PASSWD":   "secret",
		"/grp/entry/username": "alice",
	}).enterPrefix("/grp/entry")

	if it := m.leftPanel.SelectedItem(); it.Depth != 0 {
		t.Fatalf("expected an attribute item, got %+v", it)
	}
	if m.rightPanel.mode != detailModeDetail {
		t.Errorf("expected detailModeDetail, got %d", m.rightPanel.mode)
	}
	if sel := m.leftPanel.SelectedItem().Name; sel != "PASSWD" {
		t.Fatalf("expected PASSWD selected, got %q", sel)
	}
	view := m.rightPanel.View().Content
	if !strings.Contains(view, "secret") {
		t.Errorf("expected attribute value in detail view, got:\n%s", view)
	}
	if strings.Contains(view, "Attributes") {
		t.Errorf("detail view should not be the attribute list:\n%s", view)
	}
}

// 左栏无选中项时右栏为空属性列表。
func TestRightPanelEmptyWhenNoItems(t *testing.T) {
	m := newTestMainView(t, nil)

	if m.leftPanel.ItemCount() != 0 {
		t.Fatalf("expected empty list, got %d items", m.leftPanel.ItemCount())
	}
	view := m.rightPanel.View().Content
	if !strings.Contains(view, "no attributes") {
		t.Errorf("expected empty attribute list, got:\n%s", view)
	}
}

// 属性变更后（mainview 事件流）右栏列表随之刷新。
func TestRightPanelListRefreshesOnEvent(t *testing.T) {
	m := newTestMainView(t, map[string]string{
		"/grp/entry/PASSWD": "secret",
	}).enterPrefix("/grp")

	if !strings.Contains(m.rightPanel.View().Content, "PASSWD") {
		t.Fatalf("expected PASSWD in list before change")
	}

	m.db.Set("/grp/entry/otp", []byte("1234"))
	m = updateMainView(m, mainViewModelEventMsg{Event: model.Event{Type: model.EventAttrSet, Key: "/grp/entry/otp"}})

	if !strings.Contains(m.rightPanel.View().Content, "otp") {
		t.Errorf("expected refreshed attribute list to contain new attribute, got:\n%s", m.rightPanel.View().Content)
	}
}
