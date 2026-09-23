package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"golang.org/x/crypto/hkdf"
)

type formatVectors struct {
	Inputs struct {
		Password string `json:"password_utf8"`
		SaltHex  string `json:"salt_hex"`
		NonceHex string `json:"nonce_hex"`
		Argon2   struct {
			TimeCost      uint32 `json:"time_cost"`
			MemoryCostKiB uint32 `json:"memory_cost_kib"`
			Parallelism   uint32 `json:"parallelism"`
		} `json:"argon2"`
	} `json:"inputs"`
	Expected struct {
		MasterKeyHex      string `json:"master_key_hex"`
		HKDFOKM64Hex      string `json:"hkdf_okm64_hex"`
		HKDFPRK2Hex       string `json:"hkdf_prk2_hex"`
		HMACKeyHex        string `json:"hmac_key_hex"`
		EncryptKeyHex     string `json:"encrypt_key_hex"`
		HeaderHex         string `json:"header_hex"`
		DataSegmentHex    string `json:"data_segment_hex"`
		AEADCiphertextHex string `json:"aead_ciphertext_hex"`
	} `json:"expected"`
}

func loadFormatVectors(t *testing.T) formatVectors {
	t.Helper()

	data, err := os.ReadFile("testdata/format_v1_vectors.json")
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	var v formatVectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}
	return v
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	return b
}

// TestFormatV1Vectors 用固定向量校验密钥派生、头部字节与密文体字节。
// 任何平台的实现只要通过这组向量，即可与本参考实现互通。
func TestFormatV1Vectors(t *testing.T) {
	v := loadFormatVectors(t)
	salt := mustHex(t, v.Inputs.SaltHex)
	nonce := mustHex(t, v.Inputs.NonceHex)
	if len(salt) != SaltSize || len(nonce) != NonceSize {
		t.Fatalf("unexpected salt/nonce size: %d/%d", len(salt), len(nonce))
	}
	params := Argon2Params{
		TimeCost:    v.Inputs.Argon2.TimeCost,
		MemoryCost:  v.Inputs.Argon2.MemoryCostKiB,
		Parallelism: v.Inputs.Argon2.Parallelism,
	}

	// 1) Argon2id 主密钥
	masterKey, err := DeriveMasterKey(v.Inputs.Password, salt, params)
	if err != nil {
		t.Fatalf("DeriveMasterKey: %v", err)
	}
	if got := hex.EncodeToString(masterKey); got != v.Expected.MasterKeyHex {
		t.Errorf("master key mismatch:\n got %s\nwant %s", got, v.Expected.MasterKeyHex)
	}

	// 2) HKDF 两步：okm64 → prk2 → HMAC Key / Encrypt Key
	okm64 := make([]byte, 64)
	if _, err := hkdf.New(sha256.New, masterKey, salt, nil).Read(okm64); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(okm64); got != v.Expected.HKDFOKM64Hex {
		t.Errorf("hkdf okm64 mismatch:\n got %s\nwant %s", got, v.Expected.HKDFOKM64Hex)
	}
	prk2 := hkdf.Extract(sha256.New, okm64, nil)
	if got := hex.EncodeToString(prk2); got != v.Expected.HKDFPRK2Hex {
		t.Errorf("hkdf prk2 mismatch:\n got %s\nwant %s", got, v.Expected.HKDFPRK2Hex)
	}

	subKeys, err := DeriveSubKeys(masterKey, salt)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(subKeys.HMACKey); got != v.Expected.HMACKeyHex {
		t.Errorf("hmac key mismatch:\n got %s\nwant %s", got, v.Expected.HMACKeyHex)
	}
	if got := hex.EncodeToString(subKeys.EncryptKey); got != v.Expected.EncryptKeyHex {
		t.Errorf("encrypt key mismatch:\n got %s\nwant %s", got, v.Expected.EncryptKeyHex)
	}

	// 3) 固定 salt/nonce 的头部字节（含 MAC 规则）
	hdr := &Header{Version: Version, Argon2: params, CompressionID: CompressionDEFLATE}
	copy(hdr.Magic[:], Magic)
	copy(hdr.Salt[:], salt)
	copy(hdr.Nonce[:], nonce)
	hdr.computeAndSetMAC()
	if !hdr.VerifyMAC() {
		t.Error("header MAC 规则自校验失败（应为 SHA256(header[0:80])）")
	}
	hdr.computeAndSetHMAC(subKeys.HMACKey)
	hdrBytes, err := hdr.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(hdrBytes); got != v.Expected.HeaderHex {
		t.Errorf("header bytes mismatch:\n got %s\nwant %s", got, v.Expected.HeaderHex)
	}
	if !hdr.VerifyHMAC(subKeys.HMACKey) {
		t.Error("header HMAC 校验失败")
	}

	// 4) 数据段字节（三种记录类型）
	entries := []Entry{
		NewEntryWithTimestamp(TypeText, "/vault/entry/PASSWD", []byte("secret-password"), 1700000000000),
		NewEntryWithTimestamp(TypeBlob, "/vault/entry/ssh_key", []byte{0x01, 0x02, 0x03, 0x04}, 1700000001000),
		NewEntryWithTimestamp(TypeClear, "/vault/deleted", nil, 1700000002000),
	}
	var dataSegment []byte
	for _, e := range entries {
		b, err := e.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		dataSegment = append(dataSegment, b...)
	}
	if got := hex.EncodeToString(dataSegment); got != v.Expected.DataSegmentHex {
		t.Errorf("data segment mismatch:\n got %s\nwant %s", got, v.Expected.DataSegmentHex)
	}

	// 5) AEAD 密文体字节（跳过压缩，AAD 为空）
	ciphertext, err := Encrypt(subKeys.EncryptKey, nonce, dataSegment)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(ciphertext); got != v.Expected.AEADCiphertextHex {
		t.Errorf("aead ciphertext mismatch:\n got %s\nwant %s", got, v.Expected.AEADCiphertextHex)
	}

	// 6) 端到端：固定 salt/nonce 组装完整文件（含压缩）→ 解密并解析条目
	compressed, err := Compress(dataSegment)
	if err != nil {
		t.Fatal(err)
	}
	body, err := Encrypt(subKeys.EncryptKey, nonce, compressed)
	if err != nil {
		t.Fatal(err)
	}
	file := append(append([]byte{}, hdrBytes...), body...)

	vault, err := Open(file, v.Inputs.Password)
	if err != nil {
		t.Fatalf("open vector file: %v", err)
	}
	if got, ok := vault.Get("/vault/entry/PASSWD"); !ok || string(got) != "secret-password" {
		t.Errorf("text entry not resolved: ok=%v value=%q", ok, got)
	}
	if got, ok := vault.Get("/vault/entry/ssh_key"); !ok || hex.EncodeToString(got) != "01020304" {
		t.Errorf("blob entry not resolved: ok=%v value=%s", ok, hex.EncodeToString(got))
	}
	if _, ok := vault.Get("/vault/deleted"); ok {
		t.Error("type=0 entry must be treated as deleted")
	}
}
