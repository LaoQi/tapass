package vault

import (
	"os"
	"path/filepath"
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
