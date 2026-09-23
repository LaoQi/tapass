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

	// 当前 SubKeys 的派生上下文（单一真相源）。
	// 头部参数/salt 与此不一致时，SubKeys 无法解开头部所声明的库，
	// MarshalBinary 会拒绝写出（见 ErrKDFParamsChanged）。
	derivedSalt   [SaltSize]byte
	derivedParams Argon2Params
}

// Params 返回头部声明的 KDF 参数（只读副本）。
func (v *Vault) Params() Argon2Params {
	return v.Hdr.Argon2
}

// DerivedParams 返回当前密钥实际的派生参数。
func (v *Vault) DerivedParams() Argon2Params {
	return v.derivedParams
}

// setDerivedContext 记录当前密钥的派生上下文。
func (v *Vault) setDerivedContext(salt [SaltSize]byte, params Argon2Params) {
	v.derivedSalt = salt
	v.derivedParams = params
}

// paramsConsistent 判断头部声明与密钥派生上下文是否一致。
func (v *Vault) paramsConsistent() bool {
	return v.derivedSalt == v.Hdr.Salt && v.derivedParams == v.Hdr.Argon2
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
		return nil, ErrWrongPassword
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

	v := &Vault{
		Hdr:     hdr,
		SubKeys: subKeys,
		Entries: entries,
	}
	v.setDerivedContext(hdr.Salt, hdr.Argon2)
	return v, nil
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
	if !v.paramsConsistent() {
		return nil, fmt.Errorf("%w: header argon2=%+v, keys derived with %+v",
			ErrKDFParamsChanged, v.Hdr.Argon2, v.derivedParams)
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

// ChangePassword 更换主密码，KDF 参数保持不变（等价于参数不变的 Rekey）。
func (v *Vault) ChangePassword(oldPassword, newPassword string) ([]byte, error) {
	return v.Rekey(oldPassword, newPassword, v.Hdr.Argon2)
}

// Rekey 用新的 KDF 参数重写整个 vault：新 salt、新 nonce、按新参数重新派生密钥、
// 重新加密并自校验。这是修改 KDF 参数的唯一合法入口（也可以同时更换主密码）。
//
// 必须提供当前主密码：新密钥要从密码按新参数重新派生；仅改参数而不重新派生
// 会让 vault 无法再被打开（历史缺陷）。
func (v *Vault) Rekey(oldPassword, newPassword string, params Argon2Params) ([]byte, error) {
	if err := ValidateArgon2Params(params); err != nil {
		return nil, err
	}
	if err := CheckArgon2Resource(params); err != nil {
		return nil, err
	}

	oldSubKeys, err := v.Hdr.DeriveKeys(oldPassword)
	if err != nil {
		return nil, fmt.Errorf("derive old keys: %w", err)
	}
	if !v.Hdr.VerifyHMAC(oldSubKeys.HMACKey) {
		oldSubKeys.Zero()
		return nil, fmt.Errorf("wrong old password")
	}
	oldSubKeys.Zero()

	hdr, subKeys, err := NewHeader(newPassword, params, v.Hdr.CompressionID)
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
	v.setDerivedContext(hdr.Salt, params)
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
