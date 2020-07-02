package mock

import (
	"context"
	"errors"

	petname "github.com/dustinkirkland/golang-petname"
)

// MockWallet contains an in-memory, simulated wallet implementation.
type MockWallet struct {
	Files            map[string][]byte
	AccountPasswords map[string]string
}

// AccountNames --
func (m *MockWallet) AccountNames() ([]string, error) {
	names := make([]string, 0)
	for name := range m.AccountPasswords {
		names = append(names, name)
	}
	return names, nil
}

// AccountsDir --
func (m *MockWallet) AccountsDir() string {
	return ""
}

// WriteAccountToDisk --
func (m *MockWallet) WriteAccountToDisk(ctx context.Context, password string) (string, error) {
	accountName := petname.Generate(3, "-")
	m.AccountPasswords[accountName] = password
	return accountName, nil
}

// WriteFileForAccount --
func (m *MockWallet) WriteFileForAccount(
	ctx context.Context,
	accountName string,
	fileName string,
	data []byte,
) error {
	m.Files[fileName] = data
	return nil
}

// ReadPasswordForAccount --
func (m *MockWallet) ReadPasswordForAccount(accountName string) (string, error) {
	for name, password := range m.AccountPasswords {
		if name == accountName {
			return password, nil
		}
	}
	return "", errors.New("account not found")
}

// ReadFileForAccount --
func (m *MockWallet) ReadFileForAccount(accountName string, fileName string) ([]byte, error) {
	for f, v := range m.Files {
		if f == fileName {
			return v, nil
		}
	}
	return nil, errors.New("file not found")
}
