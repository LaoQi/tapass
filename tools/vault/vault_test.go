package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tap")

	if err := Create(path, "testpassword"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("vault file was not created")
	}

	v, err := Open(path, "testpassword")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if v.Path != path {
		t.Errorf("expected path %s, got %s", path, v.Path)
	}
}

func TestOpenWrongPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tap")

	if err := Create(path, "correctpassword"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err := Open(path, "wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestSetGetDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tap")

	if err := Create(path, "password"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	v, err := Open(path, "password")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := v.Set("/group1/entry1/username", []byte("GitHub")); err != nil {
		t.Fatalf("Set username failed: %v", err)
	}
	if err := v.Set("/group1/entry1/PASSWD", []byte("secret123")); err != nil {
		t.Fatalf("Set PASSWD failed: %v", err)
	}

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

	if err := v.Delete("/group1/entry1/PASSWD"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, ok = v.Get("/group1/entry1/PASSWD")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tap")

	if err := Create(path, "password"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	v, err := Open(path, "password")
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

	if err := Create(path, "password"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	v, err := Open(path, "password")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	v.Set("/entry1/username", []byte("TestEntry"))
	v.Set("/entry1/PASSWD", []byte("mypassword"))

	v2, err := Open(path, "password")
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

	if err := Create(path, "oldpassword"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	v, err := Open(path, "oldpassword")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	v.Set("/entry1/username", []byte("Test"))

	if err := v.ChangePassword("oldpassword", "newpassword"); err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}

	_, err = Open(path, "oldpassword")
	if err == nil {
		t.Fatal("old password should not work")
	}

	v2, err := Open(path, "newpassword")
	if err != nil {
		t.Fatalf("Open with new password failed: %v", err)
	}

	val, ok := v2.Get("/entry1/username")
	if !ok || string(val) != "Test" {
		t.Errorf("expected 'Test', got '%s'", string(val))
	}
}
