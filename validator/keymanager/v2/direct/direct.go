package direct

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "keymanager-v2")

// Config --
type Config struct{}

// Keymanager implementation for direct keystores.
type Keymanager struct{}

// DefaultConfig for a direct keymanager implementation.
func DefaultConfig() *Config {
	return &Config{}
}

// NewKeymanager --
func NewKeymanager(ctx context.Context, cfg *Config) *Keymanager {
	return &Keymanager{}
}

// CreateAccount for a direct keymanager implementation.
func (dr *Keymanager) CreateAccount(ctx context.Context, password string) error {
	return errors.New("unimplemented")
}

// ConfigFile --
func (dr *Keymanager) ConfigFile(ctx context.Context) ([]byte, error) {
	return nil, nil
}
