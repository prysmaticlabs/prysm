package direct

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/sirupsen/logrus"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var log = logrus.WithField("prefix", "keymanager-v2")

const (
	keystoreFileName = "keystore.json"
)

// Wallet defines a struct which has capabilities and knowledge of how
// to read and write important accounts-related files to the filesystem.
// Useful for keymanager to have persistent capabilities for accounts on-disk.
type Wallet interface {
	WriteAccountToDisk(ctx context.Context, password string) (string, error)
	WriteFileForAccount(ctx context.Context, accountName string, fileName string, data []byte) error
	WriteKeymanagerConfigToDisk(ctx context.Context, encoded []byte) error
	ReadKeymanagerConfigFromDisk(ctx context.Context) (io.ReadCloser, error)
}

// Config for a direct keymanager.
type Config struct{}

// Keymanager implementation for direct keystores.
type Keymanager struct {
	wallet Wallet
}

// DefaultConfig for a direct keymanager implementation.
func DefaultConfig() *Config {
	return &Config{}
}

// NewKeymanager instantiates a new direct keymanager from configuration options.
func NewKeymanager(ctx context.Context, wallet Wallet, cfg *Config) *Keymanager {
	return &Keymanager{
		wallet: wallet,
	}
}

// NewKeymanagerFromConfigFile instantiates a direct keymanager instance
// from a configuration file accesed via a wallet.
// TODO(#6220): Implement.
func NewKeymanagerFromConfigFile(ctx context.Context, wallet Wallet) (*Keymanager, error) {
	return &Keymanager{
		wallet: wallet,
	}, nil
}

// CreateAccount for a direct keymanager implementation.
// TODO(#6220): Implement.
func (dr *Keymanager) CreateAccount(ctx context.Context, password string) error {
	accountName, err := dr.wallet.WriteAccountToDisk(ctx, password)
	if err != nil {
		return err
	}
	encryptor := keystorev4.New()
	secretKey := bls.RandKey()
	keystoreFile, err := encryptor.Encrypt(secretKey.Marshal(), []byte(password))
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(keystoreFile, "", "\t")
	if err != nil {
		return err
	}
	if err := dr.wallet.WriteFileForAccount(ctx, accountName, keystoreFileName, encoded); err != nil {
		return err
	}
	// TODO: Generate a withdrawal key as well.
	return nil
}

// MarshalConfigFile returns a marshaled configuration file for a direct keymanager.
// TODO(#6220): Implement.
func (dr *Keymanager) MarshalConfigFile(ctx context.Context) ([]byte, error) {
	return nil, nil
}

// FetchValidatingPublicKeys fetches the list of public keys from the direct account keystores.
func (dr *Keymanager) FetchValidatingPublicKeys() ([][48]byte, error) {
	return nil, errors.New("unimplemented")
}

// Sign signs a message using a validator key.
func (dr *Keymanager) Sign(context.Context, interface{}) (bls.Signature, error) {
	return nil, errors.New("unimplemented")
}
