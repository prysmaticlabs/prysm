package iface

import (
	"context"
	"io"
)

// Wallet --
type Wallet interface {
	Path() string
	PasswordsPath() string
	WriteAccountToDisk(ctx context.Context, filename string, encoded []byte) error
	WriteKeymanagerConfigToDisk(ctx context.Context, encoded []byte) error
	ReadKeymanagerConfigFromDisk(ctx context.Context) (io.ReadCloser, error)
}
