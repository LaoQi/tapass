package model

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/tapass/tapass-tools/vault"
)

func newTestDB(entries map[string]vault.Entry) *DB {
	v := &vault.Vault{
		Hdr:     &vault.Header{},
		SubKeys: &vault.SubKeys{},
	}
	for _, e := range entries {
		v.AddEntry(e)
	}
	return newDB(v, "")
}

func TestQueryKeysBasic(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("user"), Type: vault.TypeText},
		"/group2/entry2/PASSWD":   {Key: "/group2/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	})

	keys := db.QueryKeys("")
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d: %v", len(keys), keys)
	}
	if keys[0] != "/group1/entry1/PASSWD" {
		t.Errorf("expected '/group1/entry1/PASSWD', got '%s'", keys[0])
	}
	if keys[1] != "/group1/entry1/username" {
		t.Errorf("expected '/group1/entry1/username', got '%s'", keys[1])
	}
	if keys[2] != "/group2/entry2/PASSWD" {
		t.Errorf("expected '/group2/entry2/PASSWD', got '%s'", keys[2])
	}
}

func TestQueryKeysSubLevel(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("user"), Type: vault.TypeText},
		"/group1/entry2/PASSWD":   {Key: "/group1/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	})

	keys := db.QueryKeys("/group1")
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys under /group1, got %d: %v", len(keys), keys)
	}
	if keys[0] != "/group1/entry1/PASSWD" {
		t.Errorf("expected '/group1/entry1/PASSWD', got '%s'", keys[0])
	}
	if keys[1] != "/group1/entry1/username" {
		t.Errorf("expected '/group1/entry1/username', got '%s'", keys[1])
	}
	if keys[2] != "/group1/entry2/PASSWD" {
		t.Errorf("expected '/group1/entry2/PASSWD', got '%s'", keys[2])
	}
}

func TestGet(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/group1/entry1/PASSWD": {Key: "/group1/entry1/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
	})

	e, ok := db.Get("/group1/entry1/PASSWD")
	if !ok {
		t.Fatal("expected to find entry")
	}
	if string(e.Value) != "secret" {
		t.Errorf("expected value 'secret', got '%s'", string(e.Value))
	}

	e.Value[0] = 'X'
	e2, _ := db.Get("/group1/entry1/PASSWD")
	if string(e2.Value) != "secret" {
		t.Error("Get should return a copy, but original was modified")
	}
}

func TestSet(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{})

	cmds := db.Set("/test/key", []byte("value"))
	if len(cmds) != 0 {
		t.Errorf("expected 0 cmds without listeners, got %d", len(cmds))
	}

	e, ok := db.Get("/test/key")
	if !ok {
		t.Fatal("expected to find entry after Set")
	}
	if string(e.Value) != "value" {
		t.Errorf("expected 'value', got '%s'", string(e.Value))
	}
}

func TestDelete(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/test/key": {Key: "/test/key", Value: []byte("val"), Type: vault.TypeText},
	})

	db.Delete("/test/key")

	_, ok := db.Get("/test/key")
	if ok {
		t.Error("expected entry to be deleted")
	}
}

func TestOnChange(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{})

	var received Event
	db.OnChange(func(evt Event) []tea.Cmd {
		received = evt
		return nil
	})

	db.Set("/test/key", []byte("val"))

	if received.Type != EventAttrSet {
		t.Errorf("expected EventAttrSet, got %d", received.Type)
	}
	if received.Key != "/test/key" {
		t.Errorf("expected key '/test/key', got '%s'", received.Key)
	}
}

func TestQueryKeysGroupBeforeEntry(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/aaa/entry1/PASSWD": {Key: "/aaa/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
		"/bbb/PASSWD":        {Key: "/bbb/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	})

	keys := db.QueryKeys("")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
	}
	if keys[0] != "/aaa/entry1/PASSWD" {
		t.Errorf("expected '/aaa/entry1/PASSWD' first, got '%s'", keys[0])
	}
	if keys[1] != "/bbb/PASSWD" {
		t.Errorf("expected '/bbb/PASSWD' second, got '%s'", keys[1])
	}
}

func TestQueryKeysNestedGroup(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/group1/subgroup/entry1/PASSWD": {Key: "/group1/subgroup/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
		"/group1/entry2/PASSWD":          {Key: "/group1/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	})

	keys := db.QueryKeys("")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
	}
	if keys[0] != "/group1/entry2/PASSWD" {
		t.Errorf("expected '/group1/entry2/PASSWD', got '%s'", keys[0])
	}
	if keys[1] != "/group1/subgroup/entry1/PASSWD" {
		t.Errorf("expected '/group1/subgroup/entry1/PASSWD', got '%s'", keys[1])
	}

	keys = db.QueryKeys("/group1")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys under /group1, got %d: %v", len(keys), keys)
	}
	if keys[0] != "/group1/entry2/PASSWD" {
		t.Errorf("expected '/group1/entry2/PASSWD', got '%s'", keys[0])
	}
	if keys[1] != "/group1/subgroup/entry1/PASSWD" {
		t.Errorf("expected '/group1/subgroup/entry1/PASSWD', got '%s'", keys[1])
	}

	keys = db.QueryKeys("/group1/subgroup")
	if len(keys) != 1 {
		t.Fatalf("expected 1 key under /group1/subgroup, got %d: %v", len(keys), keys)
	}
	if keys[0] != "/group1/subgroup/entry1/PASSWD" {
		t.Errorf("expected '/group1/subgroup/entry1/PASSWD', got '%s'", keys[0])
	}
}

func TestQueryBasic(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/g1/e1/PASSWD": {Key: "/g1/e1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
		"/g1/e1/user":   {Key: "/g1/e1/user", Value: []byte("u1"), Type: vault.TypeText},
	})

	results := db.Query("/g1/e1")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Key != "/g1/e1/PASSWD" {
		t.Errorf("expected '/g1/e1/PASSWD', got '%s'", results[0].Key)
	}
	if results[1].Key != "/g1/e1/user" {
		t.Errorf("expected '/g1/e1/user', got '%s'", results[1].Key)
	}
	if string(results[0].Entry.Value) != "p1" {
		t.Errorf("expected value 'p1', got '%s'", string(results[0].Entry.Value))
	}
}

func TestQueryKeysEntryAttrs(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/entry/PASSWD":   {Key: "/entry/PASSWD", Value: []byte("secret"), Type: vault.TypeText},
		"/entry/username": {Key: "/entry/username", Value: []byte("user"), Type: vault.TypeText},
	})

	keys := db.QueryKeys("/entry")
	if len(keys) != 2 {
		t.Fatalf("expected 2 attr keys, got %d: %v", len(keys), keys)
	}
	if keys[0] != "/entry/PASSWD" {
		t.Errorf("expected '/entry/PASSWD', got '%s'", keys[0])
	}
	if keys[1] != "/entry/username" {
		t.Errorf("expected '/entry/username', got '%s'", keys[1])
	}
}

func TestConfig(t *testing.T) {
	v := &vault.Vault{
		Hdr: &vault.Header{
			Argon2: vault.Argon2Params{TimeCost: 6, MemoryCost: 16384, Parallelism: 1},
		},
		SubKeys: &vault.SubKeys{},
	}
	db := newDB(v, "")

	cfg := db.Config()
	if cfg.Argon2.TimeCost != 6 {
		t.Errorf("expected TimeCost 6, got %d", cfg.Argon2.TimeCost)
	}
	if cfg.Argon2.MemoryCost != 16384 {
		t.Errorf("expected MemoryCost 16384, got %d", cfg.Argon2.MemoryCost)
	}
	if cfg.Argon2.Parallelism != 1 {
		t.Errorf("expected Parallelism 1, got %d", cfg.Argon2.Parallelism)
	}
}

func TestSetConfig(t *testing.T) {
	v := &vault.Vault{
		Hdr:     &vault.Header{},
		SubKeys: &vault.SubKeys{},
	}
	db := newDB(v, "")

	var received Event
	db.OnChange(func(evt Event) []tea.Cmd {
		received = evt
		return nil
	})

	cmds := db.SetConfig(Config{Argon2: Argon2Params{TimeCost: 10, MemoryCost: 32768, Parallelism: 2}})
	if len(cmds) != 0 {
		t.Errorf("expected 0 cmds from listener returning nil, got %d", len(cmds))
	}
	if received.Type != EventConfigChanged {
		t.Errorf("expected EventConfigChanged, got %d", received.Type)
	}

	cfg := db.Config()
	if cfg.Argon2.TimeCost != 10 {
		t.Errorf("expected TimeCost 10, got %d", cfg.Argon2.TimeCost)
	}
	if cfg.Argon2.MemoryCost != 32768 {
		t.Errorf("expected MemoryCost 32768, got %d", cfg.Argon2.MemoryCost)
	}
	if cfg.Argon2.Parallelism != 2 {
		t.Errorf("expected Parallelism 2, got %d", cfg.Argon2.Parallelism)
	}
}

func TestHasChildEntries(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{
		"/group1/entry1/PASSWD":   {Key: "/group1/entry1/PASSWD", Value: []byte("p1"), Type: vault.TypeText},
		"/group1/entry1/username": {Key: "/group1/entry1/username", Value: []byte("u1"), Type: vault.TypeText},
		"/group1/entry2/PASSWD":   {Key: "/group1/entry2/PASSWD", Value: []byte("p2"), Type: vault.TypeText},
	})

	if !db.HasChildEntries("/group1/entry1") {
		t.Error("expected /group1/entry1 to have child entries")
	}
	if db.HasChildEntries("/group1/entry1/PASSWD") {
		t.Error("expected /group1/entry1/PASSWD to have no child entries")
	}
	if db.HasChildEntries("/group1") {
		t.Error("expected /group1 to have no direct child entries (only sub-groups)")
	}
	if db.HasChildEntries("/nonexistent") {
		t.Error("expected /nonexistent to have no child entries")
	}
}

func TestCreateDBAndOpenDB(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.tap"
	password := "testpassword123"

	db, err := CreateDB(path, password)
	if err != nil {
		t.Fatalf("CreateDB failed: %v", err)
	}
	if db.Path() != path {
		t.Errorf("expected path %s, got %s", path, db.Path())
	}

	db2, err := OpenDB(path, password)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	if db2.Path() != path {
		t.Errorf("expected path %s, got %s", path, db2.Path())
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.tap"
	password := "testpassword123"

	db, err := CreateDB(path, password)
	if err != nil {
		t.Fatalf("CreateDB failed: %v", err)
	}

	db.Set("/group1/entry1/PASSWD", []byte("secret"))

	if err := db.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	db2, err := OpenDB(path, password)
	if err != nil {
		t.Fatalf("OpenDB after Save failed: %v", err)
	}

	e, ok := db2.Get("/group1/entry1/PASSWD")
	if !ok {
		t.Fatal("expected to find entry after Save + OpenDB")
	}
	if string(e.Value) != "secret" {
		t.Errorf("expected 'secret', got '%s'", string(e.Value))
	}
}

func TestOpenDBWrongPassword(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.tap"

	_, err := CreateDB(path, "correct")
	if err != nil {
		t.Fatalf("CreateDB failed: %v", err)
	}

	_, err = OpenDB(path, "wrong")
	if err == nil {
		t.Error("expected error with wrong password")
	}
}

func TestOpenDBFileNotFound(t *testing.T) {
	_, err := OpenDB("/nonexistent/path/vault.tap", "password")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
