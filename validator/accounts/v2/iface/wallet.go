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
	AccountsDir() string
	ListDirs() ([]string, error)
	// Read methods for important wallet and accounts-related files.
	ReadEncryptedSeedFromDisk(ctx context.Context) (io.ReadCloser, error)
	ReadFileAtPath(ctx context.Context, filePath string, fileName string) ([]byte, error)
	ReadPasswordFromDisk(ctx context.Context, passwordFileName string) (string, error)
	// Write methods to persist important wallet and accounts-related files to disk.
	WriteFileAtPath(ctx context.Context, pathName string, fileName string, data []byte) error
	WritePasswordToDisk(ctx context.Context, passwordFileName string, password string) error
	WriteEncryptedSeedToDisk(ctx context.Context, encoded []byte) error
}
