package iface

import (
	"context"
	"io"
)

// Wallet defines a struct which has capabilities and knowledge of how
// to read and write important accounts-related files to the filesystem.
// Useful for keymanager to have persistent capabilities for accounts on-disk.
type Wallet interface {
	AccountsDir() string
	CanUnlockAccounts() bool
	WriteFileAtPath(ctx context.Context, pathName string, fileName string, data []byte) error
	ReadEncryptedSeedFromDisk(ctx context.Context) (io.ReadCloser, error)
	WriteEncryptedSeedToDisk(ctx context.Context, encoded []byte) error
	AccountNames() ([]string, error)
	ReadPasswordForAccount(accountName string) (string, error)
	ReadFileForAccount(accountName string, fileName string) ([]byte, error)
	WriteAccountToDisk(ctx context.Context, password string) (string, error)
	WriteFileForAccount(ctx context.Context, accountName string, fileName string, data []byte) error
}
