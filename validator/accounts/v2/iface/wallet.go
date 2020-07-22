package iface

import (
	"context"
)

// Wallet defines a struct which has capabilities and knowledge of how
// to read and write important accounts-related files to the filesystem.
// Useful for keymanagers to have persistent capabilities for accounts on-disk.
type Wallet interface {
	// Methods to retrieve wallet and accounts metadata.
	AccountsDir() string
	CanUnlockAccounts() bool
	// Read methods for important wallet and accounts-related files.
	ReadFileAtPath(ctx context.Context, filePath string, fileName string) ([]byte, error)
	ReadPasswordFromDisk(ctx context.Context, passwordFileName string) (string, error)
	// Write methods to persist important wallet and accounts-related files to disk.
	WriteFileAtPath(ctx context.Context, pathName string, fileName string, data []byte) error
	WritePasswordToDisk(ctx context.Context, passwordFileName string, password string) error
}
