package accounts

import (
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

// AccountConfig specifies parameters to run to delete, enable, disable accounts.
type AccountConfig struct {
	Wallet     *wallet.Wallet
	Keymanager keymanager.IKeymanager
	PublicKeys [][]byte
}
