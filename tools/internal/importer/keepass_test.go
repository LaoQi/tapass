package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tapass/tapass-tools/vault"
)

const testXML = `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile>
  <Meta>
    <Generator>KeePass</Generator>
    <RecycleBinEnabled>True</RecycleBinEnabled>
    <RecycleBinUUID>RECYCLEBIN==</RecycleBinUUID>
  </Meta>
  <Root>
    <Group>
      <UUID>ROOTUUID==</UUID>
      <Name>Root</Name>
      <Group>
        <UUID>GROUP1UUID==</UUID>
        <Name>工作账号</Name>
        <Entry>
          <UUID>ENTRY1UUID==</UUID>
          <Times>
            <LastModificationTime>2024-06-15T10:30:00Z</LastModificationTime>
            <CreationTime>2024-01-01T08:00:00Z</CreationTime>
          </Times>
          <String>
            <Key>Title</Key>
            <Value>公司邮箱</Value>
          </String>
          <String>
            <Key>UserName</Key>
            <Value>user@company.com</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>MySecret123</Value>
          </String>
          <String>
            <Key>URL</Key>
            <Value>https://mail.company.com</Value>
          </String>
          <String>
            <Key>Notes</Key>
            <Value>工作邮箱</Value>
          </String>
          <String>
            <Key>服务器地址</Key>
            <Value>imap.company.com</Value>
          </String>
        </Entry>
      </Group>
      <Group>
        <UUID>GROUP2UUID==</UUID>
        <Name>个人账号</Name>
        <Entry>
          <UUID>ENTRY2UUID==</UUID>
          <Times>
            <LastModificationTime>2024-12-01T14:00:00Z</LastModificationTime>
          </Times>
          <String>
            <Key>Title</Key>
            <Value>GitHub</Value>
          </String>
          <String>
            <Key>UserName</Key>
            <Value>myuser</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>ghp_test123</Value>
          </String>
          <String>
            <Key>TimeOtp-Secret-Base32</Key>
            <Value>JBSWY3DPEHPK3PXP</Value>
          </String>
          <String>
            <Key>TimeOtp-Period</Key>
            <Value>30</Value>
          </String>
          <String>
            <Key>TimeOtp-Length</Key>
            <Value>6</Value>
          </String>
        </Entry>
      </Group>
      <Group>
        <UUID>RECYCLEBIN==</UUID>
        <Name>Recycle Bin</Name>
        <Entry>
          <UUID>DELETEDUUID==</UUID>
          <String>
            <Key>Title</Key>
            <Value>已删除的条目</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>deleted</Value>
          </String>
        </Entry>
      </Group>
    </Group>
  </Root>
</KeePassFile>`

func TestImportBasic(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(testXML), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	stats, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
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
	v, err := vault.Open(tapData, "testpassword")
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

func TestImportTOTPTimeOtp(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(testXML), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	_, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
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
	if !strings.Contains(totp, "digits=6") {
		t.Errorf("expected digits=6 in URI, got '%s'", totp)
	}
	if !strings.Contains(totp, "period=30") {
		t.Errorf("expected period=30 in URI, got '%s'", totp)
	}

	_, hasOtpField := v.Get("/个人账号/GitHub/TimeOtp-Secret-Base32")
	if hasOtpField {
		t.Error("TimeOtp-Secret-Base32 should not be stored as separate attribute")
	}
}

func TestImportTOTPPlugin(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile>
  <Meta>
    <Generator>KeePass</Generator>
    <RecycleBinEnabled>False</RecycleBinEnabled>
  </Meta>
  <Root>
    <Group>
      <UUID>ROOT==</UUID>
      <Name>Root</Name>
      <Group>
        <UUID>G1==</UUID>
        <Name>服务</Name>
        <Entry>
          <UUID>E1==</UUID>
          <Times>
            <LastModificationTime>2025-03-10T09:00:00Z</LastModificationTime>
          </Times>
          <String>
            <Key>Title</Key>
            <Value>AWS</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>aws-pass</Value>
          </String>
          <String>
            <Key>TOTP Seed</Key>
            <Value>GEZDGNBVGY3TQOJQ</Value>
          </String>
          <String>
            <Key>TOTP Settings</Key>
            <Value>30;6</Value>
          </String>
        </Entry>
      </Group>
    </Group>
  </Root>
</KeePassFile>`

	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(xml), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	stats, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if stats.TOTP != 1 {
		t.Errorf("expected 1 TOTP, got %d", stats.TOTP)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	val, ok := v.Get("/服务/AWS/TOTP")
	if !ok {
		t.Fatal("expected TOTP attribute for plugin format")
	}

	totp := string(val)
	if !strings.Contains(totp, "otpauth://totp/") {
		t.Errorf("expected otpauth URI, got '%s'", totp)
	}
	if !strings.Contains(totp, "secret=GEZDGNBVGY3TQOJQ") {
		t.Errorf("expected secret in URI, got '%s'", totp)
	}
	if !strings.Contains(totp, "period=30") {
		t.Errorf("expected period=30 in URI, got '%s'", totp)
	}
	if !strings.Contains(totp, "digits=6") {
		t.Errorf("expected digits=6 in URI, got '%s'", totp)
	}

	_, hasSeed := v.Get("/服务/AWS/TOTP Seed")
	if hasSeed {
		t.Error("TOTP Seed should not be stored as separate attribute")
	}
	_, hasSettings := v.Get("/服务/AWS/TOTP Settings")
	if hasSettings {
		t.Error("TOTP Settings should not be stored as separate attribute")
	}
}

func TestImportTOTPOtpKey(t *testing.T) {
	otpAuthXML := `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile>
  <Meta>
    <Generator>KeePass</Generator>
    <RecycleBinEnabled>False</RecycleBinEnabled>
  </Meta>
  <Root>
    <Group>
      <UUID>ROOT==</UUID>
      <Name>Root</Name>
      <Group>
        <UUID>G1==</UUID>
        <Name>云服务</Name>
        <Entry>
          <UUID>E1==</UUID>
          <Times>
            <LastModificationTime>2025-05-20T12:00:00Z</LastModificationTime>
          </Times>
          <String>
            <Key>Title</Key>
            <Value>Google</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>google-pass</Value>
          </String>
          <String>
            <Key>otp</Key>
            <Value>otpauth://totp/Google:myuser@gmail.com?secret=JBSWY3DPEHPK3PXP&amp;issuer=Google&amp;algorithm=SHA1&amp;digits=6&amp;period=30</Value>
          </String>
        </Entry>
      </Group>
    </Group>
  </Root>
</KeePassFile>`

	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(otpAuthXML), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	stats, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if stats.TOTP != 1 {
		t.Errorf("expected 1 TOTP, got %d", stats.TOTP)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	val, ok := v.Get("/云服务/Google/TOTP")
	if !ok {
		t.Fatal("expected TOTP attribute for otp key")
	}

	totp := string(val)
	if !strings.HasPrefix(totp, "otpauth://totp/") {
		t.Errorf("expected otpauth URI, got '%s'", totp)
	}
}

func TestImportKeeOtpFormat(t *testing.T) {
	keeOtpXML := `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile>
  <Meta>
    <Generator>KeePass</Generator>
    <RecycleBinEnabled>False</RecycleBinEnabled>
  </Meta>
  <Root>
    <Group>
      <UUID>ROOT==</UUID>
      <Name>Root</Name>
      <Group>
        <UUID>G1==</UUID>
        <Name>测试</Name>
        <Entry>
          <UUID>E1==</UUID>
          <Times>
            <LastModificationTime>2025-01-01T00:00:00Z</LastModificationTime>
          </Times>
          <String>
            <Key>Title</Key>
            <Value>KeeOtp条目</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>pass123</Value>
          </String>
          <String>
            <Key>otp</Key>
            <Value>key=JBSWY3DPEHPK3PXP&amp;size=8&amp;step=60&amp;otpHashMode=SHA256</Value>
          </String>
        </Entry>
      </Group>
    </Group>
  </Root>
</KeePassFile>`

	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(keeOtpXML), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	stats, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if stats.TOTP != 1 {
		t.Errorf("expected 1 TOTP, got %d", stats.TOTP)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	val, ok := v.Get("/测试/KeeOtp条目/TOTP")
	if !ok {
		t.Fatal("expected TOTP attribute for KeeOtp format")
	}

	totp := string(val)
	if !strings.Contains(totp, "secret=JBSWY3DPEHPK3PXP") {
		t.Errorf("expected secret in URI, got '%s'", totp)
	}
	if !strings.Contains(totp, "digits=8") {
		t.Errorf("expected digits=8 in URI, got '%s'", totp)
	}
	if !strings.Contains(totp, "period=60") {
		t.Errorf("expected period=60 in URI, got '%s'", totp)
	}
	if !strings.Contains(totp, "algorithm=SHA256") {
		t.Errorf("expected algorithm=SHA256 in URI, got '%s'", totp)
	}
}

func TestImportTimestamp(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(testXML), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	_, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	expectedTime, err := time.Parse(time.RFC3339, "2024-06-15T10:30:00Z")
	if err != nil {
		t.Fatalf("parse expected time: %v", err)
	}
	expectedTs := uint64(expectedTime.UnixMilli())

	entries := v.List()
	passwdEntry, ok := entries["/工作账号/公司邮箱/PASSWD"]
	if !ok {
		t.Fatal("expected PASSWD entry")
	}

	if passwdEntry.Timestamp != expectedTs {
		t.Errorf("expected timestamp %d (2024-06-15T10:30:00Z), got %d", expectedTs, passwdEntry.Timestamp)
	}

	expectedTime2, err := time.Parse(time.RFC3339, "2024-12-01T14:00:00Z")
	if err != nil {
		t.Fatalf("parse expected time2: %v", err)
	}
	expectedTs2 := uint64(expectedTime2.UnixMilli())

	totpEntry, ok := entries["/个人账号/GitHub/TOTP"]
	if !ok {
		t.Fatal("expected TOTP entry")
	}

	if totpEntry.Timestamp != expectedTs2 {
		t.Errorf("expected timestamp %d (2024-12-01T14:00:00Z), got %d", expectedTs2, totpEntry.Timestamp)
	}
}

func TestImportNestedGroups(t *testing.T) {
	nestedXML := `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile>
  <Meta>
    <Generator>KeePass</Generator>
    <RecycleBinEnabled>False</RecycleBinEnabled>
  </Meta>
  <Root>
    <Group>
      <UUID>ROOT==</UUID>
      <Name>Root</Name>
      <Group>
        <UUID>G1==</UUID>
        <Name>一级分组</Name>
        <Group>
          <UUID>G2==</UUID>
          <Name>二级分组</Name>
          <Entry>
            <UUID>E1==</UUID>
            <Times>
              <LastModificationTime>2025-01-01T00:00:00Z</LastModificationTime>
            </Times>
            <String>
              <Key>Title</Key>
              <Value>深层条目</Value>
            </String>
            <String>
              <Key>Password</Key>
              <Value>deep123</Value>
            </String>
          </Entry>
        </Group>
      </Group>
    </Group>
  </Root>
</KeePassFile>`

	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(nestedXML), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	stats, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if stats.Entries != 1 {
		t.Errorf("expected 1 entry, got %d", stats.Entries)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	val, ok := v.Get("/一级分组/二级分组/深层条目/PASSWD")
	if !ok || string(val) != "deep123" {
		t.Errorf("expected nested PASSWD 'deep123', got '%s', ok=%v", string(val), ok)
	}
}

func TestImportTimestampBase64(t *testing.T) {
	base64XML := `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile>
  <Meta>
    <Generator>KeePass</Generator>
    <RecycleBinEnabled>False</RecycleBinEnabled>
  </Meta>
  <Root>
    <Group>
      <UUID>ROOT==</UUID>
      <Name>Root</Name>
      <Group>
        <UUID>G1==</UUID>
        <Name>测试</Name>
        <Entry>
          <UUID>E1==</UUID>
          <Times>
            <LastModificationTime>KGT/3Q4AAAA=</LastModificationTime>
          </Times>
          <String>
            <Key>Title</Key>
            <Value>Base64时间测试</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>test123</Value>
          </String>
        </Entry>
      </Group>
    </Group>
  </Root>
</KeePassFile>`

	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(base64XML), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	_, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	expectedTime, err := time.Parse(time.RFC3339, "2024-06-15T10:30:00Z")
	if err != nil {
		t.Fatalf("parse expected time: %v", err)
	}
	expectedTs := uint64(expectedTime.UnixMilli())

	entries := v.List()
	passwdEntry, ok := entries["/测试/Base64时间测试/PASSWD"]
	if !ok {
		t.Fatal("expected PASSWD entry")
	}

	if passwdEntry.Timestamp != expectedTs {
		t.Errorf("expected timestamp %d (2024-06-15T10:30:00Z), got %d", expectedTs, passwdEntry.Timestamp)
	}
}

func TestImportSteamTOTPTimeOtp(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile>
  <Meta>
    <Generator>KeePass</Generator>
    <RecycleBinEnabled>False</RecycleBinEnabled>
  </Meta>
  <Root>
    <Group>
      <UUID>ROOT==</UUID>
      <Name>Root</Name>
      <Group>
        <UUID>G1==</UUID>
        <Name>游戏</Name>
        <Entry>
          <UUID>E1==</UUID>
          <Times>
            <LastModificationTime>2025-06-01T12:00:00Z</LastModificationTime>
          </Times>
          <String>
            <Key>Title</Key>
            <Value>Steam</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>steam-pass</Value>
          </String>
          <String>
            <Key>TimeOtp-Secret-Base32</Key>
            <Value>JBSWY3DPEHPK3PXP</Value>
          </String>
          <String>
            <Key>TimeOtp-Period</Key>
            <Value>30</Value>
          </String>
          <String>
            <Key>TimeOtp-Length</Key>
            <Value>S</Value>
          </String>
        </Entry>
      </Group>
    </Group>
  </Root>
</KeePassFile>`

	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(xml), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	stats, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if stats.TOTP != 1 {
		t.Errorf("expected 1 TOTP, got %d", stats.TOTP)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	val, ok := v.Get("/游戏/Steam/TOTP")
	if !ok {
		t.Fatal("expected TOTP attribute for Steam")
	}

	totp := string(val)
	if !strings.Contains(totp, "digits=S") {
		t.Errorf("expected digits=S in URI for Steam TOTP, got '%s'", totp)
	}
	if !strings.Contains(totp, "secret=JBSWY3DPEHPK3PXP") {
		t.Errorf("expected secret in URI, got '%s'", totp)
	}
}

func TestImportSteamTOTPPlugin(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile>
  <Meta>
    <Generator>KeePass</Generator>
    <RecycleBinEnabled>False</RecycleBinEnabled>
  </Meta>
  <Root>
    <Group>
      <UUID>ROOT==</UUID>
      <Name>Root</Name>
      <Group>
        <UUID>G1==</UUID>
        <Name>游戏</Name>
        <Entry>
          <UUID>E1==</UUID>
          <Times>
            <LastModificationTime>2025-06-01T12:00:00Z</LastModificationTime>
          </Times>
          <String>
            <Key>Title</Key>
            <Value>Steam2</Value>
          </String>
          <String>
            <Key>Password</Key>
            <Value>steam-pass2</Value>
          </String>
          <String>
            <Key>TOTP Seed</Key>
            <Value>GEZDGNBVGY3TQOJQ</Value>
          </String>
          <String>
            <Key>TOTP Settings</Key>
            <Value>30;S</Value>
          </String>
        </Entry>
      </Group>
    </Group>
  </Root>
</KeePassFile>`

	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "test.xml")
	tapPath := filepath.Join(dir, "test.tap")

	if err := os.WriteFile(xmlPath, []byte(xml), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	stats, err := Import(xmlPath, tapPath, "testpassword")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if stats.TOTP != 1 {
		t.Errorf("expected 1 TOTP, got %d", stats.TOTP)
	}

	tapData, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	v, err := vault.Open(tapData, "testpassword")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	val, ok := v.Get("/游戏/Steam2/TOTP")
	if !ok {
		t.Fatal("expected TOTP attribute for Steam plugin format")
	}

	totp := string(val)
	if !strings.Contains(totp, "digits=S") {
		t.Errorf("expected digits=S in URI for Steam TOTP, got '%s'", totp)
	}
	if !strings.Contains(totp, "period=30") {
		t.Errorf("expected period=30 in URI, got '%s'", totp)
	}
}
