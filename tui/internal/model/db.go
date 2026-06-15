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

type DB struct {
	vault     *vault.Vault
	listeners []Listener
	dbPath    string
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
	if err := atomicWriteFile(path, data); err != nil {
		return nil, err
	}
	v, err := vault.Open(data, password)
	if err != nil {
		return nil, err
	}
	return newDB(v, path), nil
}

func (db *DB) Path() string {
	return db.dbPath
}

func (db *DB) Save() error {
	data, err := db.vault.MarshalBinary()
	if err != nil {
		return err
	}
	return atomicWriteFile(db.dbPath, data)
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
	return db.emit(Event{Type: EventAttrSet, Key: key})
}

func (db *DB) Delete(key string) []tea.Cmd {
	db.vault.Delete(key)
	return db.emit(Event{Type: EventAttrDeleted, Key: key})
}

func (db *DB) ChangePassword(old, new string) ([]tea.Cmd, error) {
	_, err := db.vault.ChangePassword(old, new)
	if err != nil {
		return nil, err
	}
	return db.emit(Event{Type: EventVaultChanged}), nil
}

func (db *DB) Config() Config {
	a := db.vault.Hdr.Argon2
	return Config{Argon2: Argon2Params{
		TimeCost:    a.TimeCost,
		MemoryCost:  a.MemoryCost,
		Parallelism: a.Parallelism,
	}}
}

func (db *DB) SetConfig(c Config) []tea.Cmd {
	db.vault.Hdr.Argon2 = vault.Argon2Params{
		TimeCost:    c.Argon2.TimeCost,
		MemoryCost:  c.Argon2.MemoryCost,
		Parallelism: c.Argon2.Parallelism,
	}
	return db.emit(Event{Type: EventConfigChanged})
}

func (db *DB) OnChange(fn Listener) func() {
	db.listeners = append(db.listeners, fn)
	return func() {
		for i, l := range db.listeners {
			if &l == &fn {
				db.listeners = append(db.listeners[:i], db.listeners[i+1:]...)
				break
			}
		}
	}
}

func (db *DB) emit(evt Event) []tea.Cmd {
	var cmds []tea.Cmd
	for _, fn := range db.listeners {
		cmds = append(cmds, fn(evt)...)
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
