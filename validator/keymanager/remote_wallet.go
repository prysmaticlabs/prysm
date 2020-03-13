package keymanager

import (
	"github.com/prysmaticlabs/prysm/shared/bls"
	e2wtypes "github.com/wealdtech/go-eth2-wallet-types"
)

// TODO
var RemotewalletOptsHelp = `The wallet key manager stores keys in a local encrypted store.  The options are:
  - location This is the location to look for wallets.  If not supplied it will
    use the standard (operating system-dependent) path.
  - accounts This is a list of account specifiers.  An account specifier is of
    the form <wallet name>/[account name],  where the account name can be a
    regular expression.  If the account specifier is just <wallet name> all
    accounts in that wallet will be used.  Multiple account specifiers can be
    supplied if required.
  - passphrase This is the passphrase used to encrypt the accounts when they
    were created.  Multiple passphrases can be supplied if required.

An sample keymanager options file (with annotations; these should be removed if
using this as a template) is:

  {
    "location":    "/wallets",               // Look for wallets in the directory '/wallets'
    "accounts":    ["Validators/Account.*"], // Use all accounts in the 'Validators' wallet starting with 'Account'
    "passphrases": ["secret1","secret2"]     // Use the passphrases 'secret1' and 'secret2' to decrypt accounts
  }`

func NewRemoteWallet(input string) (KeyManager, string, error) {
	km := &RemoteWallet{
		accounts: make(map[[48]byte]e2wtypes.Account),
	}

	return km, RemotewalletOptsHelp, nil
}

// Wallet is a key manager that loads keys from a local Ethereum 2 wallet.
type RemoteWallet struct {
	accounts map[[48]byte]e2wtypes.Account
}

// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
func (km *RemoteWallet) FetchValidatingKeys() ([][48]byte, error) {
	res := make([][48]byte, 0, len(km.accounts))
	for pubKey := range km.accounts {
		res = append(res, pubKey)
	}
	return res, nil
}

// Sign signs a message for the validator to broadcast.
func (km *RemoteWallet) Sign(pubKey [48]byte, root [32]byte, domain uint64) (*bls.Signature, error) {
	account, exists := km.accounts[pubKey]
	if !exists {
		return nil, ErrNoSuchKey
	}
	sig, err := account.Sign(root[:], domain)
	if err != nil {
		return nil, err
	}
	return bls.SignatureFromBytes(sig.Marshal())
}
