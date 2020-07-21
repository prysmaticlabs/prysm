package iface

import (
	"context"
	"io"
)

// Wallet defines a struct which has capabilities and knowledge of how
// to read and write important accounts-related files to the filesystem.
// Useful for keymanagers to have persistent capabilities for accounts on-disk.
type Wallet interface {
	// Methods to retrieve wallet and accounts metadata.
	AccountNames() ([]string, error)
	AccountsDir() string
	CanUnlockAccounts() bool
	// Read methods for important wallet and accounts-related files.
	ReadEncryptedSeedFromDisk(ctx context.Context) (io.ReadCloser, error)
	ReadPasswordForAccount(accountName string) (string, error)
	ReadFileForAccount(accountName string, fileName string) ([]byte, error)
	ReadFileAtPath(ctx context.Context, filePath string, fileName string) ([]byte, error)
	// Write methods to persist important wallet and accounts-related files to disk.
	WriteFileAtPath(ctx context.Context, pathName string, fileName string, data []byte) error
	WriteEncryptedSeedToDisk(ctx context.Context, encoded []byte) error
	WriteAccountToDisk(ctx context.Context, password string) (string, error)
	WriteFileForAccount(ctx context.Context, accountName string, fileName string, data []byte) error
}
