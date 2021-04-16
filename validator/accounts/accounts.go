package accounts

import (
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

var (
	errKeymanagerNotSupported = "Keymanager kind not supported: %s"
	// MsgCouldNotInitializeKeymanager informs about failed Keymanager initialization
	ErrCouldNotInitializeKeymanager = "could not initialize Keymanager"
)

// Config specifies parameters for accounts commands.
type Config struct {
	Wallet           *wallet.Wallet
	Keymanager       keymanager.IKeymanager
	DeletePublicKeys [][]byte
}
