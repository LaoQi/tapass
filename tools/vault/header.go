package vault

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

const (
	Magic             = "TAPASS"
	Version           = 1
	HeaderSize        = 144
	MACDataSize       = 80
	CompressionNone   = 0
	CompressionDEFLATE = 1
)

type Argon2Params struct {
	TimeCost    uint32
	MemoryCost  uint32
	Parallelism uint32
}

var DefaultArgon2Params = Argon2Params{
	TimeCost:    6,
	MemoryCost:  16384,
	Parallelism: 1,
}

type Header struct {
	Magic         [6]byte
	Version       uint16
	Salt          [SaltSize]byte
	Nonce         [NonceSize]byte
	Argon2        Argon2Params
	CompressionID uint8
	Reserved      [3]byte
	HeaderMAC     [32]byte
	HeaderHMAC    [32]byte
}

func NewHeader(password string, argon2Params Argon2Params, compressionID uint8) (*Header, *SubKeys, error) {
	h := &Header{
		Version:       Version,
		Argon2:        argon2Params,
		CompressionID: compressionID,
	}
	copy(h.Magic[:], Magic)

	if _, err := rand.Read(h.Salt[:]); err != nil {
		return nil, nil, fmt.Errorf("generate salt: %w", err)
	}
	if _, err := rand.Read(h.Nonce[:]); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	masterKey := DeriveMasterKey(password, h.Salt[:], h.Argon2.TimeCost, h.Argon2.MemoryCost, h.Argon2.Parallelism)
	subKeys, err := DeriveSubKeys(masterKey, h.Salt[:])
	zeroBytes(masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("derive sub keys: %w", err)
	}

	h.computeAndSetMAC()
	h.computeAndSetHMAC(subKeys.HMACKey)

	return h, subKeys, nil
}

func (h *Header) computeAndSetMAC() {
	data := h.marshalForMAC()
	mac := ComputeSHA256(data)
	copy(h.HeaderMAC[:], mac)
}

func (h *Header) computeAndSetHMAC(hmacKey []byte) {
	hmac := ComputeHMAC(hmacKey, h.HeaderMAC[:])
	copy(h.HeaderHMAC[:], hmac)
}

func (h *Header) marshalForMAC() []byte {
	buf := new(bytes.Buffer)
	buf.Write(h.Magic[:])
	binary.Write(buf, binary.LittleEndian, h.Version)
	buf.Write(h.Salt[:])
	buf.Write(h.Nonce[:])
	binary.Write(buf, binary.LittleEndian, h.Argon2.TimeCost)
	binary.Write(buf, binary.LittleEndian, h.Argon2.MemoryCost)
	binary.Write(buf, binary.LittleEndian, h.Argon2.Parallelism)
	binary.Write(buf, binary.LittleEndian, h.CompressionID)
	buf.Write(h.Reserved[:])
	return buf.Bytes()
}

func (h *Header) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Write(h.Magic[:])
	binary.Write(buf, binary.LittleEndian, h.Version)
	buf.Write(h.Salt[:])
	buf.Write(h.Nonce[:])
	binary.Write(buf, binary.LittleEndian, h.Argon2.TimeCost)
	binary.Write(buf, binary.LittleEndian, h.Argon2.MemoryCost)
	binary.Write(buf, binary.LittleEndian, h.Argon2.Parallelism)
	binary.Write(buf, binary.LittleEndian, h.CompressionID)
	buf.Write(h.Reserved[:])
	buf.Write(h.HeaderMAC[:])
	buf.Write(h.HeaderHMAC[:])
	return buf.Bytes(), nil
}

func UnmarshalHeader(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("data too short: %d < %d", len(data), HeaderSize)
	}

	h := &Header{}
	r := bytes.NewReader(data)

	if _, err := r.Read(h.Magic[:]); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if string(h.Magic[:]) != Magic {
		return nil, fmt.Errorf("invalid magic: %s", string(h.Magic[:]))
	}

	if err := binary.Read(r, binary.LittleEndian, &h.Version); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	if h.Version != Version {
		return nil, fmt.Errorf("unsupported version: %d", h.Version)
	}

	if _, err := r.Read(h.Salt[:]); err != nil {
		return nil, fmt.Errorf("read salt: %w", err)
	}
	if _, err := r.Read(h.Nonce[:]); err != nil {
		return nil, fmt.Errorf("read nonce: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &h.Argon2.TimeCost); err != nil {
		return nil, fmt.Errorf("read time cost: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &h.Argon2.MemoryCost); err != nil {
		return nil, fmt.Errorf("read memory cost: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &h.Argon2.Parallelism); err != nil {
		return nil, fmt.Errorf("read parallelism: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &h.CompressionID); err != nil {
		return nil, fmt.Errorf("read compression id: %w", err)
	}
	if _, err := r.Read(h.Reserved[:]); err != nil {
		return nil, fmt.Errorf("read reserved: %w", err)
	}

	if _, err := r.Read(h.HeaderMAC[:]); err != nil {
		return nil, fmt.Errorf("read header mac: %w", err)
	}
	if _, err := r.Read(h.HeaderHMAC[:]); err != nil {
		return nil, fmt.Errorf("read header hmac: %w", err)
	}

	return h, nil
}

func (h *Header) VerifyMAC() bool {
	data := h.marshalForMAC()
	expectedMAC := ComputeSHA256(data)
	return bytes.Equal(h.HeaderMAC[:], expectedMAC)
}

func (h *Header) VerifyHMAC(hmacKey []byte) bool {
	return VerifyHMAC(hmacKey, h.HeaderMAC[:], h.HeaderHMAC[:])
}

func (h *Header) DeriveKeys(password string) (*SubKeys, error) {
	masterKey := DeriveMasterKey(password, h.Salt[:], h.Argon2.TimeCost, h.Argon2.MemoryCost, h.Argon2.Parallelism)
	sk, err := DeriveSubKeys(masterKey, h.Salt[:])
	zeroBytes(masterKey)
	return sk, err
}
