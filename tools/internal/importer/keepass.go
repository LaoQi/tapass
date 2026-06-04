package importer

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tapass/tapass-tools/vault"
)

type KeePassFile struct {
	XMLName xml.Name `xml:"KeePassFile"`
	Meta    Meta     `xml:"Meta"`
	Root    Root     `xml:"Root"`
}

type Meta struct {
	Generator         string `xml:"Generator"`
	RecycleBinUUID    string `xml:"RecycleBinUUID"`
	RecycleBinEnabled string `xml:"RecycleBinEnabled"`
}

type Root struct {
	Group          Group          `xml:"Group"`
	DeletedObjects DeletedObjects `xml:"DeletedObjects"`
}

type Group struct {
	UUID    string  `xml:"UUID"`
	Name    string  `xml:"Name"`
	Groups  []Group `xml:"Group"`
	Entries []Entry `xml:"Entry"`
}

type Entry struct {
	UUID   string      `xml:"UUID"`
	Times  EntryTimes  `xml:"Times"`
	Strings []ValueData `xml:"String"`
}

type EntryTimes struct {
	LastModificationTime string `xml:"LastModificationTime"`
	CreationTime         string `xml:"CreationTime"`
	LastAccessTime       string `xml:"LastAccessTime"`
	ExpiryTime           string `xml:"ExpiryTime"`
	Expires              string `xml:"Expires"`
	UsageCount           string `xml:"UsageCount"`
	LocationChanged      string `xml:"LocationChanged"`
}

type ValueData struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type DeletedObjects struct {
	Objects []DeletedObject `xml:"DeletedObject"`
}

type DeletedObject struct {
	UUID        string `xml:"UUID"`
	DeletionTime string `xml:"DeletionTime"`
}

type ImportStats struct {
	Groups   int
	Entries  int
	Skipped  int
	TOTP     int
}

func Import(xmlPath, tapPath, password string) (*ImportStats, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("read xml: %w", err)
	}

	var kf KeePassFile
	if err := xml.Unmarshal(data, &kf); err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	if err := vault.Create(tapPath, password); err != nil {
		return nil, fmt.Errorf("create vault: %w", err)
	}

	v, err := vault.Open(tapPath, password)
	if err != nil {
		return nil, fmt.Errorf("open vault: %w", err)
	}

	stats := &ImportStats{}
	recycleBinUUID := strings.TrimSpace(kf.Meta.RecycleBinUUID)
	walkGroups(v, &kf.Root.Group, "", recycleBinUUID, true, stats)

	if err := v.SortAndWrite(); err != nil {
		return nil, fmt.Errorf("sort and write: %w", err)
	}

	return stats, nil
}

func walkGroups(v *vault.Vault, g *Group, parentPath, recycleBinUUID string, isRoot bool, stats *ImportStats) {
	if recycleBinUUID != "" && strings.TrimSpace(g.UUID) == recycleBinUUID {
		stats.Skipped++
		return
	}

	groupPath := parentPath
	if !isRoot && g.Name != "" {
		if parentPath != "" {
			groupPath = parentPath + "/" + sanitizeName(g.Name)
		} else {
			groupPath = sanitizeName(g.Name)
		}
	}

	if !isRoot {
		stats.Groups++
	}

	for i := range g.Entries {
		importEntry(v, &g.Entries[i], groupPath, stats)
	}

	for i := range g.Groups {
		walkGroups(v, &g.Groups[i], groupPath, recycleBinUUID, false, stats)
	}
}

func importEntry(v *vault.Vault, e *Entry, groupPath string, stats *ImportStats) {
	title := ""
	var totpFields map[string]string
	var totpSeed string
	var totpSettings string
	var otpValue string
	otherFields := make(map[string]string)

	for _, vd := range e.Strings {
		switch {
		case vd.Key == "Title":
			title = vd.Value
		case vd.Key == "Password":
			otherFields["PASSWD"] = vd.Value
		case vd.Key == "UserName":
			otherFields["username"] = vd.Value
		case vd.Key == "URL":
			otherFields["url"] = vd.Value
		case vd.Key == "Notes":
			otherFields["notes"] = vd.Value
		case strings.HasPrefix(vd.Key, "TimeOtp-"):
			if totpFields == nil {
				totpFields = make(map[string]string)
			}
			totpFields[vd.Key] = vd.Value
		case vd.Key == "TOTP Seed":
			totpSeed = vd.Value
		case vd.Key == "TOTP Settings":
			totpSettings = vd.Value
		case vd.Key == "otp":
			otpValue = vd.Value
		default:
			otherFields[vd.Key] = vd.Value
		}
	}

	if title == "" {
		title = strings.TrimSpace(e.UUID)
		if title == "" {
			stats.Skipped++
			return
		}
	}

	entryPath := groupPath + "/" + sanitizeName(title)
	if groupPath == "" {
		entryPath = sanitizeName(title)
	}

	ts := parseKeePassTimestamp(e.Times.LastModificationTime)

	for k, val := range otherFields {
		if val == "" {
			continue
		}
		key := "/" + entryPath + "/" + k
		v.AddEntry(vault.NewEntryWithTimestamp(vault.TypeText, key, []byte(val), ts))
	}

	totpUri := ""
	if totpFields != nil {
		totpUri = buildTimeOtpUri(title, totpFields)
	}
	if totpUri == "" && totpSeed != "" {
		totpUri = buildPluginTotpUri(title, totpSeed, totpSettings)
	}
	if totpUri == "" && otpValue != "" {
		totpUri = resolveOtpValue(otpValue)
	}

	if totpUri != "" {
		key := "/" + entryPath + "/TOTP"
		v.AddEntry(vault.NewEntryWithTimestamp(vault.TypeText, key, []byte(totpUri), ts))
		stats.TOTP++
	}

	stats.Entries++
}

func buildTimeOtpUri(title string, fields map[string]string) string {
	secret := fields["TimeOtp-Secret-Base32"]
	if secret == "" {
		secret = fields["TimeOtp-Secret"]
	}
	if secret == "" {
		return ""
	}

	period := "30"
	if v, ok := fields["TimeOtp-Period"]; ok && v != "" {
		period = v
	}
	digits := "6"
	if v, ok := fields["TimeOtp-Length"]; ok && v != "" {
		digits = v
	}
	algorithm := "SHA1"
	if v, ok := fields["TimeOtp-Algorithm"]; ok && v != "" {
		algorithm = v
	}

	return fmt.Sprintf("otpauth://totp/%s?secret=%s&digits=%s&period=%s&algorithm=%s",
		urlEncode(title), urlEncode(secret), digits, period, algorithm)
}

func buildPluginTotpUri(title, seed, settings string) string {
	if seed == "" {
		return ""
	}

	period := "30"
	digits := "6"

	if settings != "" {
		parts := strings.Split(settings, ";")
		if len(parts) >= 1 && parts[0] != "" {
			if _, err := strconv.Atoi(parts[0]); err == nil {
				period = parts[0]
			}
		}
		if len(parts) >= 2 && parts[1] != "" {
			if _, err := strconv.Atoi(parts[1]); err == nil {
				digits = parts[1]
			}
		}
	}

	return fmt.Sprintf("otpauth://totp/%s?secret=%s&digits=%s&period=%s&algorithm=SHA1",
		urlEncode(title), urlEncode(seed), digits, period)
}

func resolveOtpValue(val string) string {
	if strings.HasPrefix(val, "otpauth://") {
		return val
	}

	params, err := url.ParseQuery(val)
	if err != nil {
		return ""
	}

	secret := params.Get("key")
	if secret == "" {
		return ""
	}

	digits := params.Get("size")
	if digits == "" {
		digits = "6"
	}
	period := params.Get("step")
	if period == "" {
		period = "30"
	}
	algorithm := params.Get("otpHashMode")
	if algorithm == "" {
		algorithm = "SHA1"
	}

	return fmt.Sprintf("otpauth://totp/?secret=%s&digits=%s&period=%s&algorithm=%s",
		urlEncode(secret), digits, period, algorithm)
}

func parseKeePassTimestamp(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return uint64(time.Now().UnixMilli())
	}

	if data, err := base64.StdEncoding.DecodeString(s); err == nil && len(data) >= 1 {
		if len(data) < 8 {
			padded := make([]byte, 8)
			copy(padded, data)
			data = padded
		}
		secs := int64(binary.LittleEndian.Uint64(data[:8]))
		const dotNetEpochToUnixSecs int64 = 62135596800
		unixSecs := secs - dotNetEpochToUnixSecs
		if unixSecs > 0 {
			return uint64(unixSecs * 1000)
		}
	}

	t, err := time.Parse(time.RFC3339Nano, s)
	if err == nil {
		return uint64(t.UnixMilli())
	}

	t, err = time.Parse("2006-01-02T15:04:05Z", s)
	if err == nil {
		return uint64(t.UnixMilli())
	}

	t, err = time.Parse("2006-01-02T15:04:05", s)
	if err == nil {
		return uint64(t.UnixMilli())
	}

	return uint64(time.Now().UnixMilli())
}

func urlEncode(s string) string {
	return url.PathEscape(s)
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, "\n", " ")
	return name
}
