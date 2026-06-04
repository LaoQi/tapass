package local

import "github.com/tapass/tapass-tools/vault"

type LocalStore struct{}

func New() *LocalStore {
	return &LocalStore{}
}

func (s *LocalStore) Open(path string, password string) (*vault.Vault, error) {
	return vault.Open(path, password)
}

func (s *LocalStore) Create(path string, password string) error {
	return vault.Create(path, password)
}

func (s *LocalStore) Save(v *vault.Vault) error {
	return v.Write()
}
