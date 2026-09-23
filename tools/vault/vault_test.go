package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeVaultFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write vault file: %v", err)
	}
}

func readVaultFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read vault file: %v", err)
	}
	return data
}

func TestCreateAndOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tap")

	data, err := Create("testpassword")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	writeVaultFile(t, path, data)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("vault file was not created")
	}

	fileData := readVaultFile(t, path)
	v, err := Open(fileData, "testpassword")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = v
}

func TestOpenWrongPassword(t *testing.T) {
	data, err := Create("correctpassword")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = Open(data, "wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestSetGetDelete(t *testing.T) {
	data, err := Create("password")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	v, err := Open(data, "password")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	v.Set("/group1/entry1/username", []byte("GitHub"))
	v.Set("/group1/entry1/PASSWD", []byte("secret123"))

	val, ok := v.Get("/group1/entry1/username")
	if !ok {
		t.Fatal("Get username failed: key not found")
	}
	if string(val) != "GitHub" {
		t.Errorf("expected 'GitHub', got '%s'", string(val))
	}

	val, ok = v.Get("/group1/entry1/PASSWD")
	if !ok {
		t.Fatal("Get PASSWD failed: key not found")
	}
	if string(val) != "secret123" {
		t.Errorf("expected 'secret123', got '%s'", string(val))
	}

	v.Delete("/group1/entry1/PASSWD")

	_, ok = v.Get("/group1/entry1/PASSWD")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestList(t *testing.T) {
	data, err := Create("password")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	v, err := Open(data, "password")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	v.Set("/group1/entry1/username", []byte("GitHub"))
	v.Set("/group1/entry1/PASSWD", []byte("secret"))

	entries := v.List()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tap")

	data, err := Create("password")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	writeVaultFile(t, path, data)

	fileData := readVaultFile(t, path)
	v, err := Open(fileData, "password")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	v.Set("/entry1/username", []byte("TestEntry"))
	v.Set("/entry1/PASSWD", []byte("mypassword"))

	newData, err := v.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}
	writeVaultFile(t, path, newData)

	fileData2 := readVaultFile(t, path)
	v2, err := Open(fileData2, "password")
	if err != nil {
		t.Fatalf("Second Open failed: %v", err)
	}

	val, ok := v2.Get("/entry1/username")
	if !ok || string(val) != "TestEntry" {
		t.Errorf("expected 'TestEntry', got '%s'", string(val))
	}

	val, ok = v2.Get("/entry1/PASSWD")
	if !ok || string(val) != "mypassword" {
		t.Errorf("expected 'mypassword', got '%s'", string(val))
	}
}

func TestChangePassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tap")

	data, err := Create("oldpassword")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	writeVaultFile(t, path, data)

	fileData := readVaultFile(t, path)
	v, err := Open(fileData, "oldpassword")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	v.Set("/entry1/username", []byte("Test"))

	newData, err := v.ChangePassword("oldpassword", "newpassword")
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
	writeVaultFile(t, path, newData)

	oldData := readVaultFile(t, path)
	_, err = Open(oldData, "oldpassword")
	if err == nil {
		t.Fatal("old password should not work")
	}

	v2, err := Open(oldData, "newpassword")
	if err != nil {
		t.Fatalf("Open with new password failed: %v", err)
	}

	val, ok := v2.Get("/entry1/username")
	if !ok || string(val) != "Test" {
		t.Errorf("expected 'Test', got '%s'", string(val))
	}
}

// Sort 必须稳定：同一时间戳的条目保持原有相对顺序。
// 否则"同 key 同时间戳"的多条记录会后一条被排到前面，
// ResolveLatest 会解析出先写入的旧值（加密存储里的不确定性）。
func TestSortIsStable(t *testing.T) {
	const ts = 1700000000000

	var entries []Entry
	for i := 0; i < 40; i++ {
		entries = append(entries, NewEntryWithTimestamp(TypeText, fmt.Sprintf("/filler/%02d", i), []byte("x"), ts))
	}
	entries = append(entries, NewEntryWithTimestamp(TypeText, "/dup", []byte("first"), ts))
	for i := 40; i < 80; i++ {
		entries = append(entries, NewEntryWithTimestamp(TypeText, fmt.Sprintf("/filler/%02d", i), []byte("x"), ts))
	}
	entries = append(entries, NewEntryWithTimestamp(TypeText, "/dup", []byte("second"), ts))

	v := &Vault{Hdr: &Header{}, SubKeys: &SubKeys{}, Entries: entries}
	v.Sort()

	// 1) 严格稳定性：整体顺序不变（全部时间戳相同）
	for i := range v.Entries {
		if v.Entries[i].Key != entries[i].Key {
			t.Fatalf("Sort reordered equal-timestamp entries at index %d: %q -> %q",
				i, entries[i].Key, v.Entries[i].Key)
		}
	}

	// 2) 同 key 解析结果必须是后写入的值
	val, ok := v.Get("/dup")
	if !ok {
		t.Fatal("expected /dup to resolve")
	}
	if string(val) != "second" {
		t.Errorf("expected latest write to win, got %q", string(val))
	}
}

// Compact 必须保持幸存条目的原有相对顺序（此前用 map 遍历，顺序随机）。
func TestCompactKeepsOrder(t *testing.T) {
	const base = 1700000000000

	build := func() *Vault {
		var entries []Entry
		for i := 0; i < 30; i++ {
			key := fmt.Sprintf("/grp/entry%02d/PASSWD", i)
			entries = append(entries, NewEntryWithTimestamp(TypeText, key, []byte("v"), base+uint64(i*1000)))
			// 制造同 key 的历史版本与删除记录，供 Compact 清理
			entries = append(entries, NewEntryWithTimestamp(TypeText, key, []byte("old"), base+uint64(i*1000-500)))
			entries = append(entries, NewEntryWithTimestamp(TypeClear, fmt.Sprintf("/grp/deleted%02d", i), nil, base+uint64(i*1000)))
		}
		return &Vault{Hdr: &Header{}, SubKeys: &SubKeys{}, Entries: entries}
	}

	for run := 0; run < 20; run++ {
		v := build()
		want := make([]string, 0)
		for _, e := range v.Entries {
			if e.Type != TypeClear && !strings.Contains(e.Key, "/deleted") {
				// 每条 key 只保留最新：按首次出现的位置记录期望顺序
				seen := false
				for _, w := range want {
					if w == e.Key {
						seen = true
						break
					}
				}
				if !seen {
					want = append(want, e.Key)
				}
			}
		}

		v.Compact()
		if len(v.Entries) != len(want) {
			t.Fatalf("run %d: expected %d entries after compact, got %d", run, len(want), len(v.Entries))
		}
		for i, e := range v.Entries {
			if e.Key != want[i] {
				t.Fatalf("run %d: compact reordered entries at index %d: got %q want %q",
					run, i, e.Key, want[i])
			}
		}
	}
}
