package v2

import (
	"io"

	"github.com/prysmaticlabs/prysm/shared/bls"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

// CreateDirectWallet leverages input from a user to create a new instance of
// a direct, derived, or remote wallet on disk.
func CreateDirectWallet(walletWriter io.Writer, passwordsWriter io.Writer, password string) error {
	key := bls.RandKey()
	encryptor := keystorev4.New()
	keystore, err := encryptor.Encrypt(key.Marshal(), []byte(password))
	if err != nil {
		log.Fatal(err)
	}
	log.Info(keystore)
	_ = keystore
	return nil
}
