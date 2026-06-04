package model

import (
	"testing"

	"github.com/tapass/tapass-tools/vault"
)

func TestListChildrenEmpty(t *testing.T) {
	entries := make(map[string]vault.Entry)
	items := ListChildren(entries, "")
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestListChildrenBasic(t *testing.T) {
	entries := map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("user"), Type: vault.TypeText},
		"/group2/entry2/PASSWD":   {Key: "/group2/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	}

	items := ListChildren(entries, "")

	if len(items) != 2 {
		t.Fatalf("expected 2 top-level items, got %d", len(items))
	}

	g1 := items[0]
	if g1.Name != "group1" {
		t.Errorf("expected Name 'group1', got '%s'", g1.Name)
	}
	if g1.IsEntry {
		t.Error("group1 should not be an entry")
	}
	if g1.FullPath != "/group1" {
		t.Errorf("expected FullPath '/group1', got '%s'", g1.FullPath)
	}

	g2 := items[1]
	if g2.Name != "group2" {
		t.Errorf("expected Name 'group2', got '%s'", g2.Name)
	}
}

func TestListChildrenSubLevel(t *testing.T) {
	entries := map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("user"), Type: vault.TypeText},
		"/group1/entry2/PASSWD":   {Key: "/group1/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	}

	items := ListChildren(entries, "/group1")

	if len(items) != 2 {
		t.Fatalf("expected 2 items under /group1, got %d", len(items))
	}

	e1 := items[0]
	if e1.Name != "entry1" {
		t.Errorf("expected Name 'entry1', got '%s'", e1.Name)
	}
	if !e1.IsEntry {
		t.Error("entry1 should be an entry")
	}
	if e1.FullPath != "/group1/entry1" {
		t.Errorf("expected FullPath '/group1/entry1', got '%s'", e1.FullPath)
	}

	e2 := items[1]
	if e2.Name != "entry2" {
		t.Errorf("expected Name 'entry2', got '%s'", e2.Name)
	}
	if !e2.IsEntry {
		t.Error("entry2 should be an entry")
	}
}

func TestListChildrenMixedGroupAndEntry(t *testing.T) {
	entries := map[string]vault.Entry{
		"/entry1/PASSWD":         {Key: "/entry1/PASSWD", Value: []byte("rootp"), Type: vault.TypeText},
		"/group1/entry2/PASSWD":  {Key: "/group1/entry2/PASSWD", Value: []byte("g1p"), Type: vault.TypeText},
	}

	items := ListChildren(entries, "")

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].IsEntry {
		t.Error("group1 should come first (non-entry)")
	}
	if items[0].Name != "group1" {
		t.Errorf("expected 'group1', got '%s'", items[0].Name)
	}
	if !items[1].IsEntry {
		t.Error("entry1 should be an entry")
	}
	if items[1].Name != "entry1" {
		t.Errorf("expected 'entry1', got '%s'", items[1].Name)
	}
}

func TestListChildrenRootLevelEntry(t *testing.T) {
	entries := map[string]vault.Entry{
		"/entry1/PASSWD":   {Key: "/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
		"/entry1/username": {Key: "/entry1/username", Value: []byte("u1"), Type: vault.TypeText},
	}

	items := ListChildren(entries, "")

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	e1 := items[0]
	if e1.Name != "entry1" {
		t.Errorf("expected Name 'entry1', got '%s'", e1.Name)
	}
	if !e1.IsEntry {
		t.Error("entry1 should be an entry")
	}
	if e1.FullPath != "/entry1" {
		t.Errorf("expected FullPath '/entry1', got '%s'", e1.FullPath)
	}
}

func TestListChildrenDeepNesting(t *testing.T) {
	entries := map[string]vault.Entry{
		"/g1/g2/entry1/PASSWD": {Key: "/g1/g2/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
	}

	items := ListChildren(entries, "")
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "g1" || items[0].IsEntry {
		t.Errorf("expected g1 as group, got %s IsEntry=%v", items[0].Name, items[0].IsEntry)
	}

	items = ListChildren(entries, "/g1")
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "g2" || items[0].IsEntry {
		t.Errorf("expected g2 as group, got %s IsEntry=%v", items[0].Name, items[0].IsEntry)
	}

	items = ListChildren(entries, "/g1/g2")
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "entry1" || !items[0].IsEntry {
		t.Errorf("expected entry1 as entry, got %s IsEntry=%v", items[0].Name, items[0].IsEntry)
	}
}

func TestGetEntryAttrs(t *testing.T) {
	entries := map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("user"), Type: vault.TypeText},
		"/group1/entry2/PASSWD":   {Key: "/group1/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	}

	attrs := GetEntryAttrs(entries, "/group1/entry1")
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}
	if string(attrs["PASSWD"]) != "secret" {
		t.Errorf("expected PASSWD='secret', got '%s'", string(attrs["PASSWD"]))
	}
	if string(attrs["username"]) != "user" {
		t.Errorf("expected username='user', got '%s'", string(attrs["username"]))
	}
}

func TestParentPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"", ""},
		{"/", ""},
		{"/group1", ""},
		{"/group1/entry1", "/group1"},
		{"/g1/g2/entry1", "/g1/g2"},
	}
	for _, tt := range tests {
		result := ParentPath(tt.path)
		if result != tt.expected {
			t.Errorf("ParentPath(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestEntryPath(t *testing.T) {
	tests := []struct {
		groupPath string
		entryID   string
		expected  string
	}{
		{"", "entry1", "/entry1"},
		{"/", "entry1", "/entry1"},
		{"group1", "entry1", "/group1/entry1"},
		{"/group1", "entry1", "/group1/entry1"},
		{"/group1/sub", "entry1", "/group1/sub/entry1"},
	}

	for _, tt := range tests {
		result := EntryPath(tt.groupPath, tt.entryID)
		if result != tt.expected {
			t.Errorf("EntryPath(%q, %q) = %q, want %q", tt.groupPath, tt.entryID, result, tt.expected)
		}
	}
}

func TestIsEntryPath(t *testing.T) {
	entries := map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("user"), Type: vault.TypeText},
	}

	if !IsEntryPath(entries, "/group1/entry1") {
		t.Error("expected /group1/entry1 to be an entry path")
	}
	if IsEntryPath(entries, "/group1") {
		t.Error("expected /group1 to not be an entry path")
	}
	if IsEntryPath(entries, "/group1/entry1/PASSWD") {
		t.Error("expected /group1/entry1/PASSWD to not be an entry path")
	}
}

func TestListChildrenGroupBeforeEntry(t *testing.T) {
	entries := map[string]vault.Entry{
		"/aaa/entry1/PASSWD": {Key: "/aaa/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
		"/bbb/PASSWD":        {Key: "/bbb/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	}

	items := ListChildren(entries, "")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "aaa" {
		t.Errorf("expected 'aaa' group first, got '%s'", items[0].Name)
	}
	if items[0].IsEntry {
		t.Error("aaa should be a group")
	}
	if items[1].Name != "bbb" {
		t.Errorf("expected 'bbb' entry second, got '%s'", items[1].Name)
	}
	if !items[1].IsEntry {
		t.Error("bbb should be an entry")
	}
}
