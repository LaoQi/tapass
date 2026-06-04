package vault

import (
	"crypto/rand"
	"fmt"
	"os"
	"sort"
)

type Vault struct {
	Path    string
	Hdr     *Header
	SubKeys *SubKeys
	Entries []Entry
}

func Create(path, password string) error {
	hdr, subKeys, err := NewHeader(password, DefaultArgon2Params, CompressionDEFLATE)
	if err != nil {
		return fmt.Errorf("create header: %w", err)
	}

	hdrBytes, err := hdr.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal header: %w", err)
	}

	emptyData := []byte{}
	compressed, err := Compress(emptyData)
	if err != nil {
		return fmt.Errorf("compress: %w", err)
	}

	ciphertext, err := Encrypt(subKeys.EncryptKey, hdr.Nonce[:], compressed)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	fileData := append(hdrBytes, ciphertext...)
	if err := os.WriteFile(path, fileData, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func Open(path, password string) (*Vault, error) {
	fileData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if len(fileData) < HeaderSize {
		return nil, fmt.Errorf("file too short: %d", len(fileData))
	}

	hdr, err := UnmarshalHeader(fileData[:HeaderSize])
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

	ciphertext := fileData[HeaderSize:]
	plaintext, err := Decrypt(subKeys.EncryptKey, hdr.Nonce[:], ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var data []byte
	if hdr.CompressionID == CompressionDEFLATE {
		data, err = Decompress(plaintext)
		if err != nil {
			return nil, fmt.Errorf("decompress: %w", err)
		}
	} else {
		data = plaintext
	}

	entries, err := ParseAll(data)
	if err != nil {
		return nil, fmt.Errorf("parse entries: %w", err)
	}

	return &Vault{
		Path:    path,
		Hdr:     hdr,
		SubKeys: subKeys,
		Entries: entries,
	}, nil
}

func (v *Vault) Set(key string, value []byte) error {
	e := NewEntry(TypeText, key, value)
	v.Entries = append(v.Entries, e)
	return v.write()
}

func (v *Vault) SetWithTimestamp(key string, value []byte, ts uint64) error {
	e := NewEntryWithTimestamp(TypeText, key, value, ts)
	v.Entries = append(v.Entries, e)
	return v.write()
}

func (v *Vault) AddEntry(e Entry) {
	v.Entries = append(v.Entries, e)
}

func (v *Vault) SortAndWrite() error {
	sort.Slice(v.Entries, func(i, j int) bool {
		return v.Entries[i].Timestamp < v.Entries[j].Timestamp
	})
	return v.write()
}

func (v *Vault) SetBlob(key string, value []byte) error {
	e := NewEntry(TypeBlob, key, value)
	v.Entries = append(v.Entries, e)
	return v.write()
}

func (v *Vault) Get(key string) ([]byte, bool) {
	resolved := ResolveLatest(v.Entries)
	e, ok := resolved[key]
	if !ok {
		return nil, false
	}
	return e.Value, true
}

func (v *Vault) Delete(key string) error {
	e := NewEntry(TypeClear, key, nil)
	v.Entries = append(v.Entries, e)
	return v.write()
}

func (v *Vault) List() map[string]Entry {
	return ResolveLatest(v.Entries)
}

func (v *Vault) Write() error {
	return v.write()
}

func (v *Vault) write() error {
	var data []byte
	for _, e := range v.Entries {
		b, err := e.MarshalBinary()
		if err != nil {
			return fmt.Errorf("marshal entry: %w", err)
		}
		data = append(data, b...)
	}

	var compressed []byte
	var err error
	if v.Hdr.CompressionID == CompressionDEFLATE {
		compressed, err = Compress(data)
		if err != nil {
			return fmt.Errorf("compress: %w", err)
		}
	} else {
		compressed = data
	}

	if _, err := rand.Read(v.Hdr.Nonce[:]); err != nil {
		return fmt.Errorf("regenerate nonce: %w", err)
	}
	v.Hdr.computeAndSetMAC()
	v.Hdr.computeAndSetHMAC(v.SubKeys.HMACKey)

	ciphertext, err := Encrypt(v.SubKeys.EncryptKey, v.Hdr.Nonce[:], compressed)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	hdrBytes, err := v.Hdr.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal header: %w", err)
	}

	tmpPath := v.Path + ".tmp"
	fileData := append(hdrBytes, ciphertext...)
	if err := os.WriteFile(tmpPath, fileData, 0600); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}

	if err := os.Rename(tmpPath, v.Path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp file: %w", err)
	}

	return nil
}

func (v *Vault) ChangePassword(oldPassword, newPassword string) error {
	oldSubKeys, err := v.Hdr.DeriveKeys(oldPassword)
	if err != nil {
		return fmt.Errorf("derive old keys: %w", err)
	}
	if !v.Hdr.VerifyHMAC(oldSubKeys.HMACKey) {
		oldSubKeys.Zero()
		return fmt.Errorf("wrong old password")
	}
	oldSubKeys.Zero()

	hdr, subKeys, err := NewHeader(newPassword, v.Hdr.Argon2, v.Hdr.CompressionID)
	if err != nil {
		return fmt.Errorf("create new header: %w", err)
	}

	var data []byte
	for _, e := range v.Entries {
		b, err := e.MarshalBinary()
		if err != nil {
			return fmt.Errorf("marshal entry: %w", err)
		}
		data = append(data, b...)
	}

	var compressed []byte
	if hdr.CompressionID == CompressionDEFLATE {
		compressed, err = Compress(data)
		if err != nil {
			return fmt.Errorf("compress: %w", err)
		}
	} else {
		compressed = data
	}

	ciphertext, err := Encrypt(subKeys.EncryptKey, hdr.Nonce[:], compressed)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	hdrBytes, err := hdr.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal header: %w", err)
	}

	tmpPath := v.Path + ".tmp"
	fileData := append(hdrBytes, ciphertext...)
	if err := os.WriteFile(tmpPath, fileData, 0600); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}

	newVault, err := Open(tmpPath, newPassword)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("verify new vault: %w", err)
	}
	_ = newVault

	if err := os.Rename(tmpPath, v.Path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp file: %w", err)
	}

	v.Hdr = hdr
	v.SubKeys = subKeys
	return nil
}

func (v *Vault) Compact() error {
	resolved := v.List()

	var entries []Entry
	for _, e := range resolved {
		entries = append(entries, e)
	}
	v.Entries = entries

	return v.write()
}
