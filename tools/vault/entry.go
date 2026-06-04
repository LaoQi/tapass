package vault

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

const (
	TypeClear = 0
	TypeText  = 1
	TypeBlob  = 2
)

type Entry struct {
	Timestamp uint64
	Type      uint8
	Key       string
	Value     []byte
}

func NewEntry(typ uint8, key string, value []byte) Entry {
	return Entry{
		Timestamp: uint64(time.Now().UnixMilli()),
		Type:      typ,
		Key:       key,
		Value:     value,
	}
}

func NewEntryWithTimestamp(typ uint8, key string, value []byte, ts uint64) Entry {
	return Entry{
		Timestamp: ts,
		Type:      typ,
		Key:       key,
		Value:     value,
	}
}

func (e Entry) MarshalBinary() ([]byte, error) {
	keyBytes := []byte(e.Key)
	if len(keyBytes) > 65535 {
		return nil, fmt.Errorf("key too long: %d", len(keyBytes))
	}

	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.LittleEndian, e.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, e.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(keyBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(keyBytes); err != nil {
		return nil, err
	}

	valLen := uint32(0)
	if e.Type != TypeClear {
		valLen = uint32(len(e.Value))
	}
	if err := binary.Write(buf, binary.LittleEndian, valLen); err != nil {
		return nil, err
	}
	if e.Type != TypeClear && valLen > 0 {
		if _, err := buf.Write(e.Value); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func ParseAll(data []byte) ([]Entry, error) {
	var entries []Entry
	r := bytes.NewReader(data)

	for r.Len() > 0 {
		e, err := parseOne(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse entry at offset %d: %w", len(data)-r.Len(), err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}

func parseOne(r *bytes.Reader) (Entry, error) {
	var e Entry

	if err := binary.Read(r, binary.LittleEndian, &e.Timestamp); err != nil {
		return Entry{}, err
	}
	if err := binary.Read(r, binary.LittleEndian, &e.Type); err != nil {
		return Entry{}, err
	}

	var keyLen uint16
	if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
		return Entry{}, err
	}

	keyBytes := make([]byte, keyLen)
	if _, err := io.ReadFull(r, keyBytes); err != nil {
		return Entry{}, fmt.Errorf("read key: %w", err)
	}
	e.Key = string(keyBytes)

	var valLen uint32
	if err := binary.Read(r, binary.LittleEndian, &valLen); err != nil {
		return Entry{}, err
	}

	if e.Type != TypeClear && valLen > 0 {
		e.Value = make([]byte, valLen)
		if _, err := io.ReadFull(r, e.Value); err != nil {
			return Entry{}, fmt.Errorf("read value: %w", err)
		}
	}

	return e, nil
}

func ResolveLatest(entries []Entry) map[string]Entry {
	latest := make(map[string]Entry)
	for _, e := range entries {
		existing, ok := latest[e.Key]
		if !ok || e.Timestamp >= existing.Timestamp {
			latest[e.Key] = e
		}
	}

	result := make(map[string]Entry)
	for k, e := range latest {
		if e.Type != TypeClear {
			result[k] = e
		}
	}

	return result
}
