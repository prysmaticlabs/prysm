package keymanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	e2wallet "github.com/wealdtech/go-eth2-wallet"
	filesystem "github.com/wealdtech/go-eth2-wallet-store-filesystem"
	e2wtypes "github.com/wealdtech/go-eth2-wallet-types"
)

type walletOpts struct {
	Location    string   `json:"location"`
	Accounts    []string `json:"accounts"`
	Passphrases []string `json:"passphrases"`
}

var walletOptsHelp = `The wallet key manager stores keys in a local encrypted store.  The options are:
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

// NewWallet creates a key manager populated with the keys from a wallet at the given path.
func NewWallet(input string) (KeyManager, string, error) {
	opts := &walletOpts{}
	err := json.Unmarshal([]byte(input), opts)
	if err != nil {
		return nil, walletOptsHelp, err
	}

	if len(opts.Accounts) == 0 {
		return nil, walletOptsHelp, errors.New("at least one account specifier is required")
	}

	if len(opts.Passphrases) == 0 {
		return nil, walletOptsHelp, errors.New("at least one passphrase is required to decrypt accounts")
	}

	km := &Wallet{
		accounts: make(map[[48]byte]e2wtypes.Account),
	}

	var store e2wtypes.Store
	if opts.Location == "" {
		store = filesystem.New()
	} else {
		store = filesystem.New(filesystem.WithLocation(opts.Location))
	}
	for _, path := range opts.Accounts {
		parts := strings.Split(path, "/")
		if len(parts[0]) == 0 {
			return nil, walletOptsHelp, fmt.Errorf("did not understand account specifier %q", path)
		}
		wallet, err := e2wallet.OpenWallet(parts[0], e2wallet.WithStore(store))
		if err != nil {
			return nil, walletOptsHelp, err
		}
		accountSpecifier := "^.*$"
		if len(parts) > 1 && len(parts[1]) > 0 {
			accountSpecifier = fmt.Sprintf("^%s$", parts[1])
		}
		re := regexp.MustCompile(accountSpecifier)
		for account := range wallet.Accounts() {
			if re.Match([]byte(account.Name())) {
				pubKey := bytesutil.ToBytes48(account.PublicKey().Marshal())
				for _, passphrase := range opts.Passphrases {
					if err := account.Unlock([]byte(passphrase)); err != nil {
						log.WithError(err).WithField("pubKey", fmt.Sprintf("%#x", pubKey)).Warn("Failed to unlock account with supplied passphrases; cannot validate")
					} else {
						km.accounts[pubKey] = account
					}
				}
			}
		}
	}

	return km, walletOptsHelp, nil
}

// Wallet is a key manager that loads keys from a local Ethereum 2 wallet.
type Wallet struct {
	accounts map[[48]byte]e2wtypes.Account
}

// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
func (km *Wallet) FetchValidatingKeys() ([][48]byte, error) {
	res := make([][48]byte, 0, len(km.accounts))
	for pubKey := range km.accounts {
		res = append(res, pubKey)
	}
	return res, nil
}

// Sign signs a message for the validator to broadcast.
func (km *Wallet) Sign(pubKey [48]byte, root [32]byte, domain uint64) (*bls.Signature, error) {
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
