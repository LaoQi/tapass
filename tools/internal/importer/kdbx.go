package importer

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	gokeepasslib "github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
	"github.com/tapass/tapass-tools/vault"
)

func ImportKDBX(kdbxPath, tapPath, kdbxPassword, tapPassword string) (*ImportStats, error) {
	file, err := os.Open(kdbxPath)
	if err != nil {
		return nil, fmt.Errorf("open kdbx: %w", err)
	}
	defer file.Close()

	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(kdbxPassword)
	if err := gokeepasslib.NewDecoder(file).Decode(db); err != nil {
		return nil, fmt.Errorf("decrypt kdbx: %w", err)
	}

	if err := db.UnlockProtectedEntries(); err != nil {
		return nil, fmt.Errorf("unlock protected entries: %w", err)
	}

	if len(db.Content.Root.Groups) == 0 {
		return nil, fmt.Errorf("kdbx has no groups")
	}

	kf := convertDatabase(db)

	vaultData, err := vault.Create(tapPassword)
	if err != nil {
		return nil, fmt.Errorf("create vault: %w", err)
	}

	v, err := vault.Open(vaultData, tapPassword)
	if err != nil {
		return nil, fmt.Errorf("open vault: %w", err)
	}

	stats := &ImportStats{}
	recycleBinUUID := strings.TrimSpace(kf.Meta.RecycleBinUUID)
	walkGroups(v, &kf.Root.Group, "", recycleBinUUID, true, stats)

	v.Sort()
	outData, err := v.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal vault: %w", err)
	}
	if err := os.WriteFile(tapPath, outData, 0600); err != nil {
		return nil, fmt.Errorf("write vault: %w", err)
	}

	return stats, nil
}

func convertDatabase(db *gokeepasslib.Database) *KeePassFile {
	kf := &KeePassFile{}

	meta := db.Content.Meta
	kf.Meta.Generator = meta.Generator
	kf.Meta.RecycleBinUUID = uuidToBase64(meta.RecycleBinUUID)
	if meta.RecycleBinEnabled.Bool {
		kf.Meta.RecycleBinEnabled = "True"
	} else {
		kf.Meta.RecycleBinEnabled = "False"
	}

	if len(db.Content.Root.Groups) > 0 {
		kf.Root.Group = *convertGroup(&db.Content.Root.Groups[0], true)
	}

	return kf
}

func convertGroup(g *gokeepasslib.Group, isRoot bool) *Group {
	og := &Group{}
	og.UUID = uuidToBase64(g.UUID)
	og.Name = g.Name

	for i := range g.Groups {
		og.Groups = append(og.Groups, *convertGroup(&g.Groups[i], false))
	}

	for i := range g.Entries {
		og.Entries = append(og.Entries, *convertEntry(&g.Entries[i]))
	}

	return og
}

func convertEntry(e *gokeepasslib.Entry) *Entry {
	oe := &Entry{}
	oe.UUID = uuidToBase64(e.UUID)

	oe.Times = EntryTimes{
		LastModificationTime: timeWrapperToString(e.Times.LastModificationTime),
		CreationTime:         timeWrapperToString(e.Times.CreationTime),
		LastAccessTime:       timeWrapperToString(e.Times.LastAccessTime),
		ExpiryTime:           timeWrapperToString(e.Times.ExpiryTime),
		Expires:              boolToString(e.Times.Expires.Bool),
		UsageCount:           fmt.Sprintf("%d", e.Times.UsageCount),
		LocationChanged:      timeWrapperToString(e.Times.LocationChanged),
	}

	for _, vd := range e.Values {
		oe.Strings = append(oe.Strings, ValueData{
			Key:   vd.Key,
			Value: vd.Value.Content,
		})
	}

	return oe
}

func uuidToBase64(u gokeepasslib.UUID) string {
	if u == (gokeepasslib.UUID{}) {
		return ""
	}
	return base64.StdEncoding.EncodeToString(u[:])
}

func timeWrapperToString(tw *w.TimeWrapper) string {
	if tw == nil {
		return ""
	}
	if tw.Formatted {
		return tw.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	total := tw.Time.Unix() - (-62135596800)
	buf := make([]byte, 8)
	le := uint64(total)
	buf[0] = byte(le)
	buf[1] = byte(le >> 8)
	buf[2] = byte(le >> 16)
	buf[3] = byte(le >> 24)
	buf[4] = byte(le >> 32)
	buf[5] = byte(le >> 40)
	buf[6] = byte(le >> 48)
	buf[7] = byte(le >> 56)
	return base64.StdEncoding.EncodeToString(buf)
}

func boolToString(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func IsKDBX(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return strings.HasSuffix(strings.ToLower(path), ".kdbx")
	}
	defer f.Close()

	magic := make([]byte, 4)
	n, err := f.Read(magic)
	if err != nil || n < 4 {
		return strings.HasSuffix(strings.ToLower(path), ".kdbx")
	}

	if magic[0] == 0x03 && magic[1] == 0xD9 && magic[2] == 0xA2 && magic[3] == 0x9A {
		return true
	}

	return strings.HasSuffix(strings.ToLower(path), ".kdbx")
}
