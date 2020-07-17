package mock

import (
	"context"
	"errors"
	"sync"

	petname "github.com/dustinkirkland/golang-petname"
)

// Wallet contains an in-memory, simulated wallet implementation.
type Wallet struct {
	Files            map[string]map[string][]byte
	AccountPasswords map[string]string
	UnlockAccounts   bool
	lock             sync.RWMutex
}

// AccountNames --
func (m *Wallet) AccountNames() ([]string, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	names := make([]string, 0)
	for name := range m.AccountPasswords {
		names = append(names, name)
	}
	return names, nil
}

// AccountsDir --
func (m *Wallet) AccountsDir() string {
	return ""
}

// CanUnlockAccounts --
func (m *Wallet) CanUnlockAccounts() bool {
	return m.UnlockAccounts
}

// WriteAccountToDisk --
func (m *Wallet) WriteAccountToDisk(ctx context.Context, password string) (string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	accountName := petname.Generate(3, "-")
	m.AccountPasswords[accountName] = password
	return accountName, nil
}

// WriteFileForAccount --
func (m *Wallet) WriteFileForAccount(
	ctx context.Context,
	accountName string,
	fileName string,
	data []byte,
) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.Files[accountName] == nil {
		m.Files[accountName] = make(map[string][]byte)
	}
	m.Files[accountName][fileName] = data
	return nil
}

// ReadPasswordForAccount --
func (m *Wallet) ReadPasswordForAccount(accountName string) (string, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for name, password := range m.AccountPasswords {
		if name == accountName {
			return password, nil
		}
	}
	return "", errors.New("account not found")
}

// ReadFileForAccount --
func (m *Wallet) ReadFileForAccount(accountName string, fileName string) ([]byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for f, v := range m.Files[accountName] {
		if f == fileName {
			return v, nil
		}
	}
	return nil, errors.New("file not found")
}
