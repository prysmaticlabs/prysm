package imported

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	util "github.com/wealdtech/go-eth2-util"
)

// RecoverAccountsFromMnemonic given a mnemonic phrase, is able to regenerate N accounts
// from a derived seed, encrypt them according to the EIP-2334 JSON standard, and write them
// to disk. Then, the mnemonic is never stored nor used by the validator.
func (km *Keymanager) RecoverAccountsFromMnemonic(
	ctx context.Context, mnemonic, mnemonicPassphrase string, numAccounts int,
) error {
	seed, err := seedFromMnemonic(mnemonic, mnemonicPassphrase)
	if err != nil {
		return errors.Wrap(err, "could not initialize new wallet seed file")
	}
	privKeys := make([][]byte, numAccounts)
	pubKeys := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		privKey, err := util.PrivateKeyFromSeedAndPath(
			seed, fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i),
		)
		if err != nil {
			return err
		}
		privKeys[i] = privKey.Marshal()
		pubKeys[i] = privKey.PublicKey().Marshal()
	}
	return nil
}
