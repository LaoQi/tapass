package vault

import (
	"errors"
	"testing"
)

// buildVaultFile 按指定压缩算法手工组装一个 vault 文件（用于覆盖非默认路径）。
func buildVaultFile(t *testing.T, password string, compressionID uint8, entries []Entry) []byte {
	t.Helper()

	hdr, subKeys, err := NewHeader(password, DefaultArgon2Params, compressionID)
	if err != nil {
		t.Fatalf("NewHeader failed: %v", err)
	}

	var payload []byte
	for _, e := range entries {
		b, err := e.MarshalBinary()
		if err != nil {
			t.Fatalf("marshal entry: %v", err)
		}
		payload = append(payload, b...)
	}

	var body []byte
	switch compressionID {
	case CompressionNone:
		body = payload
	case CompressionDEFLATE:
		body, err = Compress(payload)
		if err != nil {
			t.Fatalf("compress: %v", err)
		}
	default:
		body = payload // 非法 ID：仅用于构造损坏文件
	}

	ciphertext, err := Encrypt(subKeys.EncryptKey, hdr.Nonce[:], body)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	hdrBytes, err := hdr.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	return append(hdrBytes, ciphertext...)
}

// Compression ID = 0（无压缩）的读写路径。
func TestCompressionNoneRoundTrip(t *testing.T) {
	file := buildVaultFile(t, "pw", CompressionNone, []Entry{
		NewEntry(TypeText, "/grp/entry/PASSWD", []byte("secret")),
	})

	v, err := Open(file, "pw")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if got := v.Params(); got != DefaultArgon2Params {
		t.Errorf("unexpected params: %+v", got)
	}
	if val, ok := v.Get("/grp/entry/PASSWD"); !ok || string(val) != "secret" {
		t.Fatalf("entry not readable: ok=%v value=%q", ok, val)
	}

	// 重新序列化后仍保持无压缩且可再次打开
	v.Set("/grp/entry/username", []byte("alice"))
	out, err := v.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}
	reopened, err := Open(out, "pw")
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	if reopened.Hdr.CompressionID != CompressionNone {
		t.Errorf("expected CompressionNone preserved, got %d", reopened.Hdr.CompressionID)
	}
	if val, ok := reopened.Get("/grp/entry/username"); !ok || string(val) != "alice" {
		t.Errorf("entry lost after round trip: ok=%v value=%q", ok, val)
	}
}

// 未知 Compression ID 必须显式报错，不能被静默当成"无压缩"（否则会解析出错乱数据）。
func TestUnknownCompressionRejected(t *testing.T) {
	data, err := Create("pw")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	h, err := UnmarshalHeader(data[:HeaderSize])
	if err != nil {
		t.Fatal(err)
	}
	h.CompressionID = 2
	h.computeAndSetMAC()
	hdrBytes, err := h.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	file := append(hdrBytes, data[HeaderSize:]...)

	if _, err := UnmarshalHeader(file[:HeaderSize]); !errors.Is(err, ErrUnsupportedCompression) {
		t.Errorf("expected ErrUnsupportedCompression from UnmarshalHeader, got %v", err)
	}
	if _, err := Open(file, "pw"); !errors.Is(err, ErrUnsupportedCompression) {
		t.Errorf("expected ErrUnsupportedCompression from Open, got %v", err)
	}
}

func TestNewHeaderRejectsUnknownCompression(t *testing.T) {
	if _, _, err := NewHeader("pw", DefaultArgon2Params, 2); !errors.Is(err, ErrUnsupportedCompression) {
		t.Errorf("expected ErrUnsupportedCompression, got %v", err)
	}
}

// 内存中篡改压缩算法后不得写出（否则产出的是无法解读的文件）。
func TestMarshalBinaryRejectsUnknownCompression(t *testing.T) {
	data, err := Create("pw")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	v, err := Open(data, "pw")
	if err != nil {
		t.Fatal(err)
	}
	v.Hdr.CompressionID = 3
	if _, err := v.MarshalBinary(); !errors.Is(err, ErrUnsupportedCompression) {
		t.Errorf("expected ErrUnsupportedCompression, got %v", err)
	}
}
