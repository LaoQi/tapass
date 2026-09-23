package model

import (
	"testing"

	"charm.land/bubbletea/v2"
	"github.com/LaoQi/tapass/tools/vault"
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

// 参数变更必须走 Rekey（重新派生），改参后库仍可被正常打开。
func TestRekey(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/rekey.tap"

	db, err := CreateDB(path, "old-password")
	if err != nil {
		t.Fatalf("CreateDB failed: %v", err)
	}
	db.Set("/grp/e/PASSWD", []byte("secret"))
	if err := db.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	var received Event
	db.OnChange(func(evt Event) []tea.Cmd {
		received = evt
		return nil
	})

	cmds, err := db.Rekey("old-password", "new-password", Argon2Params{
		TimeCost:    2,
		MemoryCost:  8192,
		Parallelism: 1,
	})
	if err != nil {
		t.Fatalf("Rekey failed: %v", err)
	}
	if len(cmds) != 0 {
		t.Errorf("expected 0 cmds from listener returning nil, got %d", len(cmds))
	}
	if received.Type != EventConfigChanged {
		t.Errorf("expected EventConfigChanged, got %d", received.Type)
	}

	if err := db.Save(); err != nil {
		t.Fatalf("Save after rekey failed: %v", err)
	}

	// 新密码 + 新参数可打开，旧密码不可打开
	reopened, err := OpenDB(path, "new-password")
	if err != nil {
		t.Fatalf("OpenDB with new password failed: %v", err)
	}
	if _, err := OpenDB(path, "old-password"); err == nil {
		t.Error("expected old password to fail after rekey")
	}

	cfg := reopened.Config()
	if cfg.Argon2.TimeCost != 2 || cfg.Argon2.MemoryCost != 8192 || cfg.Argon2.Parallelism != 1 {
		t.Errorf("unexpected params after rekey: %+v", cfg.Argon2)
	}

	e, ok := reopened.Get("/grp/e/PASSWD")
	if !ok || string(e.Value) != "secret" {
		t.Errorf("entry lost after rekey: %v %q", ok, string(e.Value))
	}
}

// 参数非法（无法精确执行）时 Rekey 必须失败且不改动库。
func TestRekeyRejectsInvalidParams(t *testing.T) {
	dir := t.TempDir()
	db, err := CreateDB(dir+"/bad.tap", "pw")
	if err != nil {
		t.Fatal(err)
	}
	// memory 不是 4*parallelism 的倍数 → 本实现无法精确执行
	if _, err := db.Rekey("pw", "pw", Argon2Params{TimeCost: 2, MemoryCost: 1002, Parallelism: 2}); err == nil {
		t.Error("expected error for non lane-aligned memory")
	}
	if _, err := db.Rekey("pw", "pw", Argon2Params{TimeCost: 0, MemoryCost: 8192, Parallelism: 1}); err == nil {
		t.Error("expected error for time cost 0")
	}
	// 超出本机能力（约 4 TiB）→ 必须被拒绝而不是 OOM
	if _, err := db.Rekey("pw", "pw", Argon2Params{TimeCost: 1, MemoryCost: 0xFFFFFF00, Parallelism: 1}); err == nil {
		t.Error("expected error for params exceeding local capability")
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
	if !db.Dirty() {
		t.Error("expected DB to be dirty after CreateDB")
	}

	if err := db.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if db.Dirty() {
		t.Error("expected DB to be clean after Save")
	}

	db2, err := OpenDB(path, password)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	if db2.Path() != path {
		t.Errorf("expected path %s, got %s", path, db2.Path())
	}
	if db2.Dirty() {
		t.Error("expected DB to be clean after OpenDB")
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
	if !db.Dirty() {
		t.Error("expected DB to be dirty after Set")
	}

	if err := db.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if db.Dirty() {
		t.Error("expected DB to be clean after Save")
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

	db, err := CreateDB(path, "correct")
	if err != nil {
		t.Fatalf("CreateDB failed: %v", err)
	}
	if err := db.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
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

// 回归：OnChange 返回的退订函数必须真正移除监听器（此前比较循环变量地址，永不生效）。
func TestOnChangeUnsubscribe(t *testing.T) {
	db := newTestDB(map[string]vault.Entry{})

	firstCalls := 0
	secondCalls := 0

	unsubscribe := db.OnChange(func(evt Event) []tea.Cmd {
		firstCalls++
		return nil
	})
	db.OnChange(func(evt Event) []tea.Cmd {
		secondCalls++
		return nil
	})

	db.Set("/k1", []byte("v"))
	if firstCalls != 1 || secondCalls != 1 {
		t.Fatalf("expected both listeners called once, got first=%d second=%d", firstCalls, secondCalls)
	}

	unsubscribe()

	db.Set("/k2", []byte("v"))
	if firstCalls != 1 {
		t.Errorf("unsubscribed listener still called: %d", firstCalls)
	}
	if secondCalls != 2 {
		t.Errorf("expected remaining listener called twice, got %d", secondCalls)
	}

	// 重复退订应幂等（不影响其它监听器）
	unsubscribe()
	db.Set("/k3", []byte("v"))
	if secondCalls != 3 {
		t.Errorf("expected remaining listener called 3 times, got %d", secondCalls)
	}
	if len(db.listeners) != 1 {
		t.Errorf("expected 1 listener left, got %d", len(db.listeners))
	}
}
