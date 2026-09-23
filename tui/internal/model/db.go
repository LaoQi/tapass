package model

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"charm.land/bubbletea/v2"
	"github.com/LaoQi/tapass/tools/vault"
)

type EventType int

const (
	EventAttrSet EventType = iota
	EventAttrDeleted
	EventVaultChanged
	EventConfigChanged
)

type Event struct {
	Type EventType
	Key  string
}

type Listener func(evt Event) []tea.Cmd

type Argon2Params struct {
	TimeCost    uint32
	MemoryCost  uint32
	Parallelism uint32
}

type Config struct {
	Argon2 Argon2Params
}

type Entry struct {
	Timestamp uint64
	Type      uint8
	Key       string
	Value     []byte
}

type QueryResult struct {
	Key   string
	Entry Entry
}

// changeListener 带 id 的监听器：Go 无法比较函数值，退订需按 id 定位
// （历史缺陷：用 `&l == &fn` 比较循环变量副本地址，退订永不生效）。
type changeListener struct {
	id int
	fn Listener
}

type DB struct {
	vault          *vault.Vault
	listeners      []changeListener
	nextListenerID int
	dbPath         string
	dirty          bool
}

func newDB(v *vault.Vault, dbPath string) *DB {
	return &DB{vault: v, dbPath: dbPath}
}

func OpenDB(path, password string) (*DB, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read vault file: %w", err)
	}
	v, err := vault.Open(data, password)
	if err != nil {
		return nil, err
	}
	return newDB(v, path), nil
}

func CreateDB(path, password string) (*DB, error) {
	data, err := vault.Create(password)
	if err != nil {
		return nil, err
	}
	v, err := vault.Open(data, password)
	if err != nil {
		return nil, err
	}
	return &DB{vault: v, dbPath: path, dirty: true}, nil
}

func (db *DB) Path() string {
	return db.dbPath
}

func (db *DB) Dirty() bool {
	return db.dirty
}

func (db *DB) Save() error {
	data, err := db.vault.MarshalBinary()
	if err != nil {
		return err
	}
	if err := atomicWriteFile(db.dbPath, data); err != nil {
		return err
	}
	db.dirty = false
	return nil
}

func (db *DB) Query(prefix string) []QueryResult {
	prefix = normalizePathPrefix(prefix)
	all := db.vault.List()
	result := make([]QueryResult, 0)
	for key, entry := range all {
		if !strings.HasPrefix(key, prefix+"/") {
			continue
		}
		result = append(result, QueryResult{Key: key, Entry: copyEntry(entry)})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	return result
}

func (db *DB) QueryKeys(prefix string) []string {
	prefix = normalizePathPrefix(prefix)
	all := db.vault.List()
	result := make([]string, 0, len(all))
	for key := range all {
		if !strings.HasPrefix(key, prefix+"/") {
			continue
		}
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func (db *DB) Get(key string) (Entry, bool) {
	all := db.vault.List()
	e, ok := all[key]
	if !ok {
		return Entry{}, false
	}
	return copyEntry(e), true
}

func (db *DB) Set(key string, value []byte) []tea.Cmd {
	db.vault.Set(key, value)
	db.dirty = true
	return db.emit(Event{Type: EventAttrSet, Key: key})
}

func (db *DB) Delete(key string) []tea.Cmd {
	db.vault.Delete(key)
	db.dirty = true
	return db.emit(Event{Type: EventAttrDeleted, Key: key})
}

func (db *DB) ChangePassword(old, new string) ([]tea.Cmd, error) {
	_, err := db.vault.ChangePassword(old, new)
	if err != nil {
		return nil, err
	}
	db.dirty = true
	return db.emit(Event{Type: EventVaultChanged}), nil
}

func (db *DB) Config() Config {
	a := db.vault.Params()
	return Config{Argon2: Argon2Params{
		TimeCost:    a.TimeCost,
		MemoryCost:  a.MemoryCost,
		Parallelism: a.Parallelism,
	}}
}

// Rekey 按新的 KDF 参数重写 vault（新 salt/nonce、重新派生、重新加密）。
// 必须提供当前主密码：新密钥需由密码按新参数重新派生。
// 参数变更只能走这里；直接改头部参数会让库无法再被打开。
func (db *DB) Rekey(oldPassword, newPassword string, a Argon2Params) ([]tea.Cmd, error) {
	if _, err := db.vault.Rekey(oldPassword, newPassword, vault.Argon2Params{
		TimeCost:    a.TimeCost,
		MemoryCost:  a.MemoryCost,
		Parallelism: a.Parallelism,
	}); err != nil {
		return nil, err
	}
	db.dirty = true
	return db.emit(Event{Type: EventConfigChanged}), nil
}

// OnChange 注册变更监听器，返回退订函数。退订幂等：重复调用无副作用。
func (db *DB) OnChange(fn Listener) func() {
	id := db.nextListenerID
	db.nextListenerID++
	db.listeners = append(db.listeners, changeListener{id: id, fn: fn})

	return func() {
		for i, l := range db.listeners {
			if l.id == id {
				db.listeners = append(db.listeners[:i], db.listeners[i+1:]...)
				return
			}
		}
	}
}

func (db *DB) emit(evt Event) []tea.Cmd {
	var cmds []tea.Cmd
	for _, l := range db.listeners {
		cmds = append(cmds, l.fn(evt)...)
	}
	return cmds
}

func copyEntry(e vault.Entry) Entry {
	valueCopy := make([]byte, len(e.Value))
	copy(valueCopy, e.Value)
	return Entry{
		Timestamp: e.Timestamp,
		Type:      e.Type,
		Key:       e.Key,
		Value:     valueCopy,
	}
}

func atomicWriteFile(path string, data []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp file: %w", err)
	}
	return nil
}
