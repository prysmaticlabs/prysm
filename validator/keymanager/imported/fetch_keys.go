package imported

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"go.opencensus.io/trace"
)

// FetchValidatingPublicKeys fetches the list of active public keys from the imported account keystores.
func (km *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	ctx, span := trace.StartSpan(ctx, "keymanager.FetchValidatingPublicKeys")
	defer span.End()

	lock.RLock()
	keys := orderedPublicKeys
	result := make([][48]byte, len(keys))
	copy(result, keys)
	lock.RUnlock()
	return result, nil
}

// FetchValidatingPrivateKeys fetches the list of private keys from the secret keys cache
func (km *Keymanager) FetchValidatingPrivateKeys(ctx context.Context) ([][32]byte, error) {
	lock.RLock()
	defer lock.RUnlock()
	privKeys := make([][32]byte, len(secretKeysCache))
	pubKeys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve public keys")
	}
	for i, pk := range pubKeys {
		seckey, ok := secretKeysCache[pk]
		if !ok {
			return nil, errors.New("Could not fetch private key")
		}
		privKeys[i] = bytesutil.ToBytes32(seckey.Marshal())
	}
	return privKeys, nil
}
