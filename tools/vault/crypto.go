package vault

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"runtime"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	KeySize        = 32
	NonceSize      = 24
	SaltSize       = 32
	AuthTagSize    = 16
	HKDFOutputSize = 64
)

var (
	HKDFInfoHMAC = []byte("tapass-v1-hmac")
	HKDFInfoEnc  = []byte("tapass-v1-enc")
)

type SubKeys struct {
	HMACKey    []byte
	EncryptKey []byte
}

func (s *SubKeys) Zero() {
	zeroBytes(s.HMACKey)
	zeroBytes(s.EncryptKey)
	runtime.KeepAlive(s)
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(&b[0])
}

func DeriveMasterKey(password string, salt []byte, timeCost, memoryCost, parallelism uint32) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		timeCost,
		memoryCost,
		uint8(parallelism),
		KeySize,
	)
}

func DeriveSubKeys(masterKey, salt []byte) (*SubKeys, error) {
	hkdfReader := hkdf.New(sha256.New, masterKey, salt, nil)
	output := make([]byte, HKDFOutputSize)
	if _, err := hkdfReader.Read(output); err != nil {
		return nil, fmt.Errorf("hkdf expand: %w", err)
	}

	hmacKey, err := hkdfExpandWithInfo(output, HKDFInfoHMAC, KeySize)
	if err != nil {
		zeroBytes(output)
		return nil, err
	}
	encKey, err := hkdfExpandWithInfo(output, HKDFInfoEnc, KeySize)
	zeroBytes(output)
	if err != nil {
		zeroBytes(hmacKey)
		return nil, err
	}

	return &SubKeys{
		HMACKey:    hmacKey,
		EncryptKey: encKey,
	}, nil
}

func hkdfExpandWithInfo(prk, info []byte, length int) ([]byte, error) {
	h := hkdf.New(sha256.New, prk, nil, info)
	out := make([]byte, length)
	if _, err := h.Read(out); err != nil {
		return nil, fmt.Errorf("hkdf expand with info: %w", err)
	}
	return out, nil
}

func Encrypt(key, nonce, plaintext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(key))
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: %d", len(nonce))
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("create aead: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nil
}

func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(key))
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: %d", len(nonce))
	}
	if len(ciphertext) < AuthTagSize {
		return nil, fmt.Errorf("ciphertext too short: %d", len(ciphertext))
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("create aead: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

func ComputeHMAC(key, message []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(message)
	return h.Sum(nil)
}

func VerifyHMAC(key, message, expectedMAC []byte) bool {
	mac := ComputeHMAC(key, message)
	return hmac.Equal(mac, expectedMAC)
}

func ComputeSHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
