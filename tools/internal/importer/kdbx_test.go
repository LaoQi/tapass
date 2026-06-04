package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gokeepasslib "github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
	"github.com/tapass/tapass-tools/vault"
)

func createTestKDBX(t *testing.T, path, password string) {
	t.Helper()

	db := gokeepasslib.NewDatabase(gokeepasslib.WithDatabaseKDBXVersion3())
	db.Credentials = gokeepasslib.NewPasswordCredentials(password)

	root := &db.Content.Root.Groups[0]
	root.Name = "Root"
	root.Entries = nil

	recycleBin := gokeepasslib.NewGroup()
	recycleBin.Name = "Recycle Bin"
	recycleBin.UUID = [16]byte{0xDE, 0xAD, 0xBE, 0xEF, 0xDE, 0xAD, 0xBE, 0xEF, 0xDE, 0xAD, 0xBE, 0xEF, 0xDE, 0xAD, 0xBE, 0xEF}
	db.Content.Meta.RecycleBinUUID = recycleBin.UUID
	db.Content.Meta.RecycleBinEnabled = w.NewBoolWrapper(true)

	deletedEntry := gokeepasslib.NewEntry()
	deletedEntry.Values = append(deletedEntry.Values,
		gokeepasslib.ValueData{Key: "Title", Value: gokeepasslib.V{Content: "已删除的条目"}},
		gokeepasslib.ValueData{Key: "Password", Value: gokeepasslib.V{Content: "deleted"}},
	)
	recycleBin.Entries = append(recycleBin.Entries, deletedEntry)
	root.Groups = append(root.Groups, recycleBin)

	workGroup := gokeepasslib.NewGroup()
	workGroup.Name = "工作账号"
	workEntry := gokeepasslib.NewEntry()
	workEntry.Values = append(workEntry.Values,
		gokeepasslib.ValueData{Key: "Title", Value: gokeepasslib.V{Content: "公司邮箱"}},
		gokeepasslib.ValueData{Key: "UserName", Value: gokeepasslib.V{Content: "user@company.com"}},
		gokeepasslib.ValueData{Key: "Password", Value: gokeepasslib.V{Content: "MySecret123", Protected: w.NewBoolWrapper(true)}},
		gokeepasslib.ValueData{Key: "URL", Value: gokeepasslib.V{Content: "https://mail.company.com"}},
		gokeepasslib.ValueData{Key: "Notes", Value: gokeepasslib.V{Content: "工作邮箱"}},
		gokeepasslib.ValueData{Key: "服务器地址", Value: gokeepasslib.V{Content: "imap.company.com"}},
	)
	workGroup.Entries = append(workGroup.Entries, workEntry)
	root.Groups = append(root.Groups, workGroup)

	personalGroup := gokeepasslib.NewGroup()
	personalGroup.Name = "个人账号"
	githubEntry := gokeepasslib.NewEntry()
	githubEntry.Values = append(githubEntry.Values,
		gokeepasslib.ValueData{Key: "Title", Value: gokeepasslib.V{Content: "GitHub"}},
		gokeepasslib.ValueData{Key: "UserName", Value: gokeepasslib.V{Content: "myuser"}},
		gokeepasslib.ValueData{Key: "Password", Value: gokeepasslib.V{Content: "ghp_test123", Protected: w.NewBoolWrapper(true)}},
		gokeepasslib.ValueData{Key: "TimeOtp-Secret-Base32", Value: gokeepasslib.V{Content: "JBSWY3DPEHPK3PXP"}},
		gokeepasslib.ValueData{Key: "TimeOtp-Period", Value: gokeepasslib.V{Content: "30"}},
		gokeepasslib.ValueData{Key: "TimeOtp-Length", Value: gokeepasslib.V{Content: "6"}},
	)
	personalGroup.Entries = append(personalGroup.Entries, githubEntry)
	root.Groups = append(root.Groups, personalGroup)

	if err := db.LockProtectedEntries(); err != nil {
		t.Fatalf("lock protected entries: %v", err)
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create kdbx file: %v", err)
	}
	defer file.Close()

	encoder := gokeepasslib.NewEncoder(file)
	if err := encoder.Encode(db); err != nil {
		t.Fatalf("encode kdbx: %v", err)
	}
}

func TestImportKDBXBasic(t *testing.T) {
	dir := t.TempDir()
	kdbxPath := filepath.Join(dir, "test.kdbx")
	tapPath := filepath.Join(dir, "test.tap")

	createTestKDBX(t, kdbxPath, "kdbxpassword")

	stats, err := ImportKDBX(kdbxPath, tapPath, "kdbxpassword", "tappassword")
	if err != nil {
		t.Fatalf("ImportKDBX failed: %v", err)
	}

	if stats.Groups != 2 {
		t.Errorf("expected 2 groups, got %d", stats.Groups)
	}
	if stats.Entries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.Entries)
	}
	if stats.Skipped != 1 {
		t.Errorf("expected 1 skipped (recycle bin), got %d", stats.Skipped)
	}
	if stats.TOTP != 1 {
		t.Errorf("expected 1 TOTP, got %d", stats.TOTP)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "tappassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	tapData2, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err = vault.Open(tapData2, "tappassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	val, ok := v.Get("/工作账号/公司邮箱/PASSWD")
	if !ok || string(val) != "MySecret123" {
		t.Errorf("expected PASSWD 'MySecret123', got '%s', ok=%v", string(val), ok)
	}

	val, ok = v.Get("/工作账号/公司邮箱/username")
	if !ok || string(val) != "user@company.com" {
		t.Errorf("expected username 'user@company.com', got '%s'", string(val))
	}

	val, ok = v.Get("/工作账号/公司邮箱/服务器地址")
	if !ok || string(val) != "imap.company.com" {
		t.Errorf("expected custom field 'imap.company.com', got '%s'", string(val))
	}

	val, ok = v.Get("/工作账号/公司邮箱/url")
	if !ok || string(val) != "https://mail.company.com" {
		t.Errorf("expected url, got '%s'", string(val))
	}

	val, ok = v.Get("/工作账号/公司邮箱/notes")
	if !ok || string(val) != "工作邮箱" {
		t.Errorf("expected notes, got '%s'", string(val))
	}

	_, ok = v.Get("/Recycle Bin/已删除的条目/PASSWD")
	if ok {
		t.Error("recycle bin entry should be skipped")
	}
}

func TestImportKDBXTOTP(t *testing.T) {
	dir := t.TempDir()
	kdbxPath := filepath.Join(dir, "test.kdbx")
	tapPath := filepath.Join(dir, "test.tap")

	createTestKDBX(t, kdbxPath, "testpass")

	_, err := ImportKDBX(kdbxPath, tapPath, "testpass", "tappass")
	if err != nil {
		t.Fatalf("ImportKDBX failed: %v", err)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "tappass")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	val, ok := v.Get("/个人账号/GitHub/TOTP")
	if !ok {
		t.Fatal("expected TOTP attribute")
	}

	totp := string(val)
	if !strings.Contains(totp, "otpauth://totp/") {
		t.Errorf("expected otpauth URI, got '%s'", totp)
	}
	if !strings.Contains(totp, "secret=JBSWY3DPEHPK3PXP") {
		t.Errorf("expected secret in URI, got '%s'", totp)
	}
}

func TestImportKDBXWrongPassword(t *testing.T) {
	dir := t.TempDir()
	kdbxPath := filepath.Join(dir, "test.kdbx")
	tapPath := filepath.Join(dir, "test.tap")

	createTestKDBX(t, kdbxPath, "correctpassword")

	_, err := ImportKDBX(kdbxPath, tapPath, "wrongpassword", "tappassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestIsKDBX(t *testing.T) {
	dir := t.TempDir()

	kdbxPath := filepath.Join(dir, "test.kdbx")
	createTestKDBX(t, kdbxPath, "test")

	if !IsKDBX(kdbxPath) {
		t.Error("expected IsKDBX to return true for .kdbx file")
	}

	xmlPath := filepath.Join(dir, "test.xml")
	if err := os.WriteFile(xmlPath, []byte(testXML), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	if IsKDBX(xmlPath) {
		t.Error("expected IsKDBX to return false for .xml file")
	}

	txtPath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(txtPath, []byte("not a kdbx file"), 0644); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	if IsKDBX(txtPath) {
		t.Error("expected IsKDBX to return false for .txt file")
	}
}
