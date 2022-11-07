package helpers

import (
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/pkg/errors"
)

func KeyFromPath(path, pw string) (*keystore.Key, error) {
	jsonb, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read keystore file %s", path)
	}
	return keystore.DecryptKey(jsonb, pw)
}
