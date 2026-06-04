package store

import "github.com/tapass/tapass-tools/vault"

type Store interface {
	Open(path string, password string) (*vault.Vault, error)
	Create(path string, password string) error
	Save(v *vault.Vault, path string) error
}
