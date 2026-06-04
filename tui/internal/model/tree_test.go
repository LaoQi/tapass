package model

import (
	"testing"

	"github.com/tapass/tapass-tools/vault"
)

func TestBuildTreeEmpty(t *testing.T) {
	entries := make(map[string]vault.Entry)
	tree := BuildTree(entries)
	if tree == nil {
		t.Fatal("tree should not be nil")
	}
	if len(tree.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(tree.Children))
	}
}

func TestBuildTreeBasic(t *testing.T) {
	entries := map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("user"), Type: vault.TypeText},
		"/group2/entry2/PASSWD":   {Key: "/group2/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	}

	tree := BuildTree(entries)

	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 top-level children, got %d", len(tree.Children))
	}

	g1 := tree.Children[0]
	if g1.ID != "group1" {
		t.Errorf("expected group ID 'group1', got '%s'", g1.ID)
	}
	if !g1.IsGroup {
		t.Error("expected group1 to be a group")
	}
	if g1.Path != "/group1" {
		t.Errorf("expected group1 Path '/group1', got '%s'", g1.Path)
	}

	if len(g1.Children) != 1 {
		t.Fatalf("expected 1 child in group1, got %d", len(g1.Children))
	}

	e1 := g1.Children[0]
	if e1.ID != "entry1" {
		t.Errorf("expected entry ID 'entry1', got '%s'", e1.ID)
	}
	if e1.Path != "/group1/entry1" {
		t.Errorf("expected entry1 Path '/group1/entry1', got '%s'", e1.Path)
	}

	if _, ok := e1.Attrs["PASSWD"]; !ok {
		t.Error("expected PASSWD attribute")
	}
	if _, ok := e1.Attrs["username"]; !ok {
		t.Error("expected username attribute")
	}
}

func TestBuildTreeAutoInferGroup(t *testing.T) {
	entries := map[string]vault.Entry{
		"/vault1/entry1/PASSWD": {Key: "/vault1/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
	}

	tree := BuildTree(entries)

	if len(tree.Children) != 1 {
		t.Fatalf("expected 1 top-level child, got %d", len(tree.Children))
	}

	v1 := tree.Children[0]
	if !v1.IsGroup {
		t.Error("vault1 should be auto-inferred as group (has children)")
	}
}

func TestGetNodeByPath(t *testing.T) {
	entries := map[string]vault.Entry{
		"/group1/entry1/PASSWD": {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
	}

	tree := BuildTree(entries)

	node := GetNodeByPath(tree, "/group1")
	if node == nil {
		t.Fatal("expected to find group1")
	}
	if node.ID != "group1" {
		t.Errorf("expected 'group1', got '%s'", node.ID)
	}

	node = GetNodeByPath(tree, "/group1/entry1")
	if node == nil {
		t.Fatal("expected to find entry1")
	}
	if node.ID != "entry1" {
		t.Errorf("expected 'entry1', got '%s'", node.ID)
	}

	node = GetNodeByPath(tree, "group1")
	if node == nil || node.ID != "group1" {
		t.Error("GetNodeByPath should normalize path without leading /")
	}
}

func TestGetEntries(t *testing.T) {
	entries := map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("u1"), Type: vault.TypeText},
		"/group1/entry2/PASSWD":   {Key: "/group1/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	}

	tree := BuildTree(entries)

	g1 := GetNodeByPath(tree, "/group1")
	if g1 == nil {
		t.Fatal("group1 not found")
	}

	allEntries := g1.GetEntries()
	if len(allEntries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(allEntries))
	}
}

func TestBuildTreeRootLevelEntry(t *testing.T) {
	entries := map[string]vault.Entry{
		"/entry1/PASSWD":   {Key: "/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
		"/entry1/username": {Key: "/entry1/username", Value: []byte("u1"), Type: vault.TypeText},
	}

	tree := BuildTree(entries)

	if len(tree.Children) != 1 {
		t.Fatalf("expected 1 top-level child, got %d", len(tree.Children))
	}

	e1 := tree.Children[0]
	if e1.ID != "entry1" {
		t.Errorf("expected ID 'entry1', got '%s'", e1.ID)
	}
	if e1.IsGroup {
		t.Error("entry1 should not be a group (no children)")
	}
	if e1.Path != "/entry1" {
		t.Errorf("expected Path '/entry1', got '%s'", e1.Path)
	}
	if _, ok := e1.Attrs["PASSWD"]; !ok {
		t.Error("expected PASSWD attribute")
	}
	if _, ok := e1.Attrs["username"]; !ok {
		t.Error("expected username attribute")
	}

	rootEntries := tree.DirectEntries()
	if len(rootEntries) != 1 {
		t.Errorf("expected 1 direct entry under root, got %d", len(rootEntries))
	}
}

func TestBuildTreeMixedRootAndGroup(t *testing.T) {
	entries := map[string]vault.Entry{
		"/entry1/PASSWD":         {Key: "/entry1/PASSWD", Value: []byte("rootp"), Type: vault.TypeText},
		"/group1/entry2/PASSWD":  {Key: "/group1/entry2/PASSWD", Value: []byte("g1p"), Type: vault.TypeText},
	}

	tree := BuildTree(entries)

	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 top-level children, got %d", len(tree.Children))
	}

	rootEntries := tree.DirectEntries()
	if len(rootEntries) != 1 {
		t.Errorf("expected 1 direct entry under root, got %d", len(rootEntries))
	}
	if rootEntries[0].ID != "entry1" {
		t.Errorf("expected direct entry 'entry1', got '%s'", rootEntries[0].ID)
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
