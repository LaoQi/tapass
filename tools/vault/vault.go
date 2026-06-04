package vault

import (
	"crypto/rand"
	"fmt"
	"sort"
)

type Vault struct {
	Hdr     *Header
	SubKeys *SubKeys
	Entries []Entry
}

func Create(password string) ([]byte, error) {
	hdr, subKeys, err := NewHeader(password, DefaultArgon2Params, CompressionDEFLATE)
	if err != nil {
		return nil, fmt.Errorf("create header: %w", err)
	}

	hdrBytes, err := hdr.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal header: %w", err)
	}

	emptyData := []byte{}
	compressed, err := Compress(emptyData)
	if err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}

	ciphertext, err := Encrypt(subKeys.EncryptKey, hdr.Nonce[:], compressed)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	fileData := append(hdrBytes, ciphertext...)
	return fileData, nil
}

func Open(data []byte, password string) (*Vault, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("data too short: %d", len(data))
	}

	hdr, err := UnmarshalHeader(data[:HeaderSize])
	if err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	if !hdr.VerifyMAC() {
		return nil, fmt.Errorf("header MAC verification failed")
	}

	subKeys, err := hdr.DeriveKeys(password)
	if err != nil {
		return nil, fmt.Errorf("derive keys: %w", err)
	}

	if !hdr.VerifyHMAC(subKeys.HMACKey) {
		return nil, fmt.Errorf("wrong password")
	}

	ciphertext := data[HeaderSize:]
	plaintext, err := Decrypt(subKeys.EncryptKey, hdr.Nonce[:], ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var entryData []byte
	if hdr.CompressionID == CompressionDEFLATE {
		entryData, err = Decompress(plaintext)
		if err != nil {
			return nil, fmt.Errorf("decompress: %w", err)
		}
	} else {
		entryData = plaintext
	}

	entries, err := ParseAll(entryData)
	if err != nil {
		return nil, fmt.Errorf("parse entries: %w", err)
	}

	return &Vault{
		Hdr:     hdr,
		SubKeys: subKeys,
		Entries: entries,
	}, nil
}

func (v *Vault) Set(key string, value []byte) {
	e := NewEntry(TypeText, key, value)
	v.Entries = append(v.Entries, e)
}

func (v *Vault) SetWithTimestamp(key string, value []byte, ts uint64) {
	e := NewEntryWithTimestamp(TypeText, key, value, ts)
	v.Entries = append(v.Entries, e)
}

func (v *Vault) AddEntry(e Entry) {
	v.Entries = append(v.Entries, e)
}

func (v *Vault) Sort() {
	sort.Slice(v.Entries, func(i, j int) bool {
		return v.Entries[i].Timestamp < v.Entries[j].Timestamp
	})
}

func (v *Vault) SetBlob(key string, value []byte) {
	e := NewEntry(TypeBlob, key, value)
	v.Entries = append(v.Entries, e)
}

func (v *Vault) Get(key string) ([]byte, bool) {
	resolved := ResolveLatest(v.Entries)
	e, ok := resolved[key]
	if !ok {
		return nil, false
	}
	return e.Value, true
}

func (v *Vault) Delete(key string) {
	e := NewEntry(TypeClear, key, nil)
	v.Entries = append(v.Entries, e)
}

func (v *Vault) List() map[string]Entry {
	return ResolveLatest(v.Entries)
}

func (v *Vault) MarshalBinary() ([]byte, error) {
	var data []byte
	for _, e := range v.Entries {
		b, err := e.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("marshal entry: %w", err)
		}
		data = append(data, b...)
	}

	var compressed []byte
	var err error
	if v.Hdr.CompressionID == CompressionDEFLATE {
		compressed, err = Compress(data)
		if err != nil {
			return nil, fmt.Errorf("compress: %w", err)
		}
	} else {
		compressed = data
	}

	if _, err := rand.Read(v.Hdr.Nonce[:]); err != nil {
		return nil, fmt.Errorf("regenerate nonce: %w", err)
	}
	v.Hdr.computeAndSetMAC()
	v.Hdr.computeAndSetHMAC(v.SubKeys.HMACKey)

	ciphertext, err := Encrypt(v.SubKeys.EncryptKey, v.Hdr.Nonce[:], compressed)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	hdrBytes, err := v.Hdr.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal header: %w", err)
	}

	fileData := append(hdrBytes, ciphertext...)
	return fileData, nil
}

func (v *Vault) ChangePassword(oldPassword, newPassword string) ([]byte, error) {
	oldSubKeys, err := v.Hdr.DeriveKeys(oldPassword)
	if err != nil {
		return nil, fmt.Errorf("derive old keys: %w", err)
	}
	if !v.Hdr.VerifyHMAC(oldSubKeys.HMACKey) {
		oldSubKeys.Zero()
		return nil, fmt.Errorf("wrong old password")
	}
	oldSubKeys.Zero()

	hdr, subKeys, err := NewHeader(newPassword, v.Hdr.Argon2, v.Hdr.CompressionID)
	if err != nil {
		return nil, fmt.Errorf("create new header: %w", err)
	}

	var data []byte
	for _, e := range v.Entries {
		b, err := e.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("marshal entry: %w", err)
		}
		data = append(data, b...)
	}

	var compressed []byte
	if hdr.CompressionID == CompressionDEFLATE {
		compressed, err = Compress(data)
		if err != nil {
			return nil, fmt.Errorf("compress: %w", err)
		}
	} else {
		compressed = data
	}

	ciphertext, err := Encrypt(subKeys.EncryptKey, hdr.Nonce[:], compressed)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	hdrBytes, err := hdr.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal header: %w", err)
	}

	fileData := append(hdrBytes, ciphertext...)

	if _, err := Open(fileData, newPassword); err != nil {
		return nil, fmt.Errorf("verify new vault: %w", err)
	}

	v.Hdr = hdr
	v.SubKeys = subKeys
	return fileData, nil
}

func (v *Vault) Compact() {
	resolved := v.List()

	var entries []Entry
	for _, e := range resolved {
		entries = append(entries, e)
	}
	v.Entries = entries
}
