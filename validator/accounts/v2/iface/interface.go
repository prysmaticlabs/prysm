// Package iface defines important interfaces for accounts functionality
// in Prysm's validator implementation, such as a wallet.
package iface

import (
	"context"
	"io"
)

// Wallet defines a struct which has capabilities and knowledge of how
// to read and write important accounts-related files to the filesystem.
// It defines an on-disk store of accounts.
type Wallet interface {
	Path() string
	PasswordsPath() string
	WriteAccountToDisk(ctx context.Context, filename string, encoded []byte) error
	WriteKeymanagerConfigToDisk(ctx context.Context, encoded []byte) error
	ReadKeymanagerConfigFromDisk(ctx context.Context) (io.ReadCloser, error)
}
