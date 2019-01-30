package accounts

import "github.com/prysmaticlabs/prysm/shared/keystore"

// NewValidatorAccount sets up a validator client's secrets and generates the necessary deposit data
// parameters needed to deposit into the deposit contract on the ETH1.0 chain. Specifically, this
// generates a BLS private and public key, and then logs the serialized deposit input hex string
// to be used in an ETH1.0 transaction by the validator.
func NewValidatorAccount() error {
	shardWithdrawalKey, err := keystore.NewKey()
	if err != nil {
		return err
	}
	validatorKey, err := keystore.NewKey()
	if err != nil {
		return err
	}
	return nil
}
