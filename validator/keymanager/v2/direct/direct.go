package direct

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/validator/accounts/v2/iface"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "keymanager-v2")

// Config for a direct keymanager.
type Config struct{}

// Keymanager implementation for direct keystores.
type Keymanager struct {
	wallet iface.Wallet
}

// DefaultConfig for a direct keymanager implementation.
func DefaultConfig() *Config {
	return &Config{}
}

// NewKeymanager instantiates a new direct keymanager from configuration options.
func NewKeymanager(ctx context.Context, wallet iface.Wallet, cfg *Config) *Keymanager {
	return &Keymanager{
		wallet: wallet,
	}
}

// NewKeymanagerFromConfigFile instantiates a direct keymanager instance
// from a configuration file accesed via a wallet.
// TODO(#6220): Implement.
func NewKeymanagerFromConfigFile(ctx context.Context, wallet iface.Wallet) (*Keymanager, error) {
	f, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	if err != nil {
		log.Fatalf("Could not read keymanager config file from wallet: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("Could not close keymanager config opts file: %v", err)
		}
	}()
	_ = f
	return &Keymanager{
		wallet: wallet,
	}, nil
}

// CreateAccount for a direct keymanager implementation.
// TODO(#6220): Implement.
func (dr *Keymanager) CreateAccount(ctx context.Context, password string) error {
	return errors.New("unimplemented")
}

// MarshalConfigFile returns a marshaled configuration file for a direct keymanager.
// TODO(#6220): Implement.
func (dr *Keymanager) MarshalConfigFile(ctx context.Context) ([]byte, error) {
	return nil, nil
}
