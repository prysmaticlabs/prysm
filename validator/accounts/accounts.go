package accounts

import (
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

var (
	errKeymanagerNotSupported = "keymanager kind not supported: %s"
	// MsgCouldNotInitializeKeymanager informs about failed keymanager initialization
	ErrCouldNotInitializeKeymanager = "could not initialize keymanager"
)

// Config specifies parameters to run to delete, enable, disable accounts.
type Config struct {
	Wallet            *wallet.Wallet
	Keymanager        keymanager.IKeymanager
	DisablePublicKeys [][]byte
	EnablePublicKeys  [][]byte
	DeletePublicKeys  [][]byte
}
