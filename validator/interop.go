package main

import (
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/keystore"
)

func loadUnencryptedKeys(path string) (map[string]*keystore.Key, error) {
	log.Warn("Loading encrypted keys from disk. Do not do this in production!")

	pth, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	r, err := os.Open(pth)
	if err != nil {
		return nil, err
	}
	validatorKeysUnecrypted, _, err := parseUnencryptedKeysFile(r)
	if err != nil {
		return nil, err
	}
	validatorKeys := make(map[string]*keystore.Key)
	for _, item := range validatorKeysUnecrypted {
		priv, err := bls.SecretKeyFromBytes(item)
		if err != nil {
			return nil, err
		}
		k, err := keystore.NewKeyFromBLS(priv)
		if err != nil {
			return nil, err
		}
		validatorKeys[hex.EncodeToString(priv.PublicKey().Marshal())] = k
	}

	return validatorKeys, nil
}

func interopValidatorKeys(idx, count uint64) (map[string]*keystore.Key, error) {
	log.Warn("Using interop deterministic generated validator keys.")
	sks, _, err := interop.DeterministicallyGenerateKeys(idx, count)
	if err != nil {
		return nil, err
	}

	validatorKeys := make(map[string]*keystore.Key)
	for _, priv := range sks {
		k, err := keystore.NewKeyFromBLS(priv)
		if err != nil {
			return nil, err
		}
		validatorKeys[hex.EncodeToString(priv.PublicKey().Marshal())] = k
	}

	return validatorKeys, nil
}
