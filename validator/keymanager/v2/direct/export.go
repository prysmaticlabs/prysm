package direct

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

// ExportKeystores retrieves the secret keys for specified public keys
// in the function input, encrypts them using the specified password,
// and returns their respective EIP-2335 keystores.
func (dr *Keymanager) ExportKeystores(
	ctx context.Context, publicKeys []bls.PublicKey, exportsPassword string,
) ([]*v2keymanager.Keystore, error) {
	encryptor := keystorev4.New()
	keystores := make([]*v2keymanager.Keystore, len(publicKeys))
	for i, pk := range publicKeys {
		pubKeyBytes := pk.Marshal()
		secretKey, ok := dr.keysCache[bytesutil.ToBytes48(pubKeyBytes)]
		if !ok {
			return nil, fmt.Errorf(
				"secret key for public key %#x not found in cache",
				pubKeyBytes,
			)
		}
		cryptoFields, err := encryptor.Encrypt(secretKey.Marshal(), exportsPassword)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"could not encrypt secret key for public key %#x",
				pubKeyBytes,
			)
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		keystores[i] = &v2keymanager.Keystore{
			Crypto:  cryptoFields,
			ID:      id.String(),
			Pubkey:  fmt.Sprintf("%x", pubKeyBytes),
			Version: encryptor.Version(),
			Name:    encryptor.Name(),
		}
	}
	return keystores, nil
}
