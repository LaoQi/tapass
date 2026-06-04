package local

import (
	"fmt"
	"os"

	"github.com/tapass/tapass-tools/vault"
)

type LocalStore struct{}

func New() *LocalStore {
	return &LocalStore{}
}

func (s *LocalStore) Open(path string, password string) (*vault.Vault, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read vault file: %w", err)
	}
	return vault.Open(data, password)
}

func (s *LocalStore) Create(path string, password string) error {
	data, err := vault.Create(password)
	if err != nil {
		return err
	}
	return atomicWriteFile(path, data)
}

func (s *LocalStore) Save(v *vault.Vault, path string) error {
	data, err := v.MarshalBinary()
	if err != nil {
		return err
	}
	return atomicWriteFile(path, data)
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
