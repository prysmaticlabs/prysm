package keymanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	e2wallet "github.com/wealdtech/go-eth2-wallet"
	filesystem "github.com/wealdtech/go-eth2-wallet-store-filesystem"
	e2wtypes "github.com/wealdtech/go-eth2-wallet-types/v2"
)

type walletOpts struct {
	Location    string   `json:"location"`
	Accounts    []string `json:"accounts"`
	Passphrases []string `json:"passphrases"`
}

// Wallet is a key manager that loads keys from a local Ethereum 2 wallet.
type Wallet struct {
	jsonWallet *jsonWallet
}

type jsonWallet struct {
	mu          sync.RWMutex
	opts     	*walletOpts
	watcher  	*fsnotify.Watcher
	scans    	map[string]*walletScan
	accounts 	map[[48]byte]e2wtypes.Account
}

type walletScan struct {
	wallet  e2wtypes.Wallet
	regexes []*regexp.Regexp
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

DYNAMICALLY ADDING ACCOUNTS:
if an account's file is added while the validator runs, its corresponding account will be added dynamically.

DYNAMICALLY DELETING ACCOUNTS:
if an account's file is deleted while the validator runs, its corresponding account will be deleted dynamically.

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

	if strings.Contains(opts.Location, "$") || strings.Contains(opts.Location, "~") || strings.Contains(opts.Location, "%") {
		log.WithField("path", opts.Location).Warn("Keystore path contains unexpanded shell expansion characters")
	}
	var store e2wtypes.Store
	if opts.Location == "" {
		store = filesystem.New()
	} else {
		store = filesystem.New(filesystem.WithLocation(opts.Location))
	}

	// generate json wallet with watcher for changes
	jsonwallet, error := newJsonWallet(opts, store)
	if error != nil {
		return nil, walletOptsHelp, error
	}

	km := &Wallet{
		jsonWallet: jsonwallet,
	}

	return km, walletOptsHelp, nil
}

// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
func (km *Wallet) FetchValidatingKeys() ([][48]byte, error) {
	accounts := km.jsonWallet.getAccounts()
	res := make([][48]byte, 0, len(accounts))
	for pubKey := range accounts {
		res = append(res, pubKey)
	}
	return res, nil
}

// Sign signs a message for the validator to broadcast.
func (km *Wallet) Sign(pubKey [48]byte, root [32]byte) (*bls.Signature, error) {
	accounts := km.jsonWallet.getAccounts()

	account, exists := accounts[pubKey]
	if !exists {
		return nil, ErrNoSuchKey
	}
	// TODO(#4817) Update with new library to remove domain here.
	sig, err := account.Sign(root[:])
	if err != nil {
		return nil, err
	}
	return bls.SignatureFromBytes(sig.Marshal())
}


// json file
func newJsonWallet(opts *walletOpts, store e2wtypes.Store) (*jsonWallet,error) {
	watcher, error := fsnotify.NewWatcher()
	if error != nil {
		return nil, error
	}

	jsonwallet := &jsonWallet{
		accounts: make(map[[48]byte]e2wtypes.Account),
		opts:    opts,
		watcher: watcher,
		scans:   make(map[string]*walletScan),
	}

	for _, path := range opts.Accounts {
		parts := strings.Split(path, "/")
		if len(parts[0]) == 0 {
			return nil, fmt.Errorf("did not understand account specifier %q", path)
		}
		wallet, err := e2wallet.OpenWallet(parts[0], e2wallet.WithStore(store))
		if err != nil {
			return nil, err
		}
		accountSpecifier := "^.*$"
		if len(parts) > 1 && len(parts[1]) > 0 {
			accountSpecifier = fmt.Sprintf("^%s$", parts[1])
		}
		re := regexp.MustCompile(accountSpecifier)
		for account := range wallet.Accounts() {
			log := log.WithField("account", fmt.Sprintf("%s/%s", wallet.Name(), account.Name()))
			if re.Match([]byte(account.Name())) {
				unlocked := false
				for _, passphrase := range opts.Passphrases { // important if several accounts have the same passphrase
					if err := account.Unlock([]byte(passphrase)); err != nil {
						log.WithError(err).Trace("Failed to unlock account with one of the supplied passphrases")
					} else {
						jsonwallet.addAccounts([]e2wtypes.Account{account})
						unlocked = true
						break
					}
				}
				if !unlocked {
					log.Warn("Failed to unlock account with any supplied passphrase; cannot validate with this key")
				}
			} else {
				// not in account specifier
			}
		}

		err = jsonwallet.addWalletWatcher(wallet, re)
		if err != nil {
			log.WithError(err).Warn("Failed to add wallet directory; dynamic updates to validator keys disabled")
		}
	}

	jsonwallet.listenToChanges()

	return jsonwallet, nil
}

func (jw *jsonWallet) getAccounts() map[[48]byte]e2wtypes.Account {
	jw.mu.RLock()
	defer jw.mu.RUnlock()

	ret := make(map[[48]byte]e2wtypes.Account)
	for k,v := range jw.accounts {
		ret[k] = v
	}

	return ret
}

func (jw *jsonWallet) addAccounts(accounts []e2wtypes.Account) {
	jw.mu.RLock()
	defer jw.mu.RUnlock()

	for i := range accounts {
		account := accounts[i]
		pubKey := bytesutil.ToBytes48(account.PublicKey().Marshal())
		jw.accounts[pubKey] = account
	}
}

func (jw *jsonWallet) addWalletWatcher(wallet e2wtypes.Wallet, specifier *regexp.Regexp) error {
	// add wallet and specifier for later identifying relevant accounts
	id := wallet.ID().String()
	scan, exists := jw.scans[id]
	if !exists {
		scan = &walletScan{
			wallet:  wallet,
			regexes: []*regexp.Regexp{},
		}

		jw.scans[id] = scan
	}
	scan.regexes = append(scan.regexes, specifier)


	// add wallet folder to watcher
	if err := jw.watcher.Add(filepath.Join(jw.opts.Location, wallet.ID().String())); err != nil {
		return err
	}
	return nil
}

func (jw *jsonWallet) handleRemoveAccountChange(accountID string) {
	for k,v := range jw.accounts {
		if v.ID().String() == accountID {
			delete(jw.accounts, k)
			log.WithField("pubKey", fmt.Sprintf("%#x", k)).Info("removed key from wallet, ")
		}
	}
}

func (jw *jsonWallet) handleAddAccountChange(walletScan *walletScan, accountID uuid.UUID) {
	account, err := walletScan.wallet.AccountByID(accountID)
	if err != nil {
		log.WithError(err).Warn("Failed to detect wallet change; account not found")
		return
	}

	// make sure keymanager file includes this account
	for _, re := range walletScan.regexes {
		if re.Match([]byte(account.Name())) {
			for _, passphrase := range jw.opts.Passphrases { // unlock
				if err := account.Unlock([]byte(passphrase)); err != nil {
					log.WithError(err).Trace("Failed to unlock account with one of the supplied passphrases")
				} else {
					jw.addAccounts([]e2wtypes.Account{account})

					pubKey := bytesutil.ToBytes48(account.PublicKey().Marshal())
					log.WithField("pubKey", fmt.Sprintf("%#x", pubKey)).Info("added new key to wallet, ")
					break
				}
			}
			break
		}
	}
}

// look for changes in wallet json files and handle those changes (delete, add, change)
func (jw *jsonWallet) listenToChanges () {
	go func() {
		for {
			select {
			case event, ok := <-jw.watcher.Events:
				if !ok {
					log.Warn("wallet stoped listening for changes in keys...")
					return
				}
				if event.Op&fsnotify.Chmod == fsnotify.Chmod {
					// nothing to do
					continue
				}

				// parse wallet and account ids
				path := strings.TrimPrefix(event.Name, jw.opts.Location)
				walletIDStr, accountIDStr := filepath.Split(path)
				walletIDStr = strings.Trim(walletIDStr, string(filepath.Separator))
				accountID, err := uuid.Parse(accountIDStr)
				if err != nil {
					// Commonly an index, sometimes a backup file; ignore.
					continue
				}
				walletScan, exists := jw.scans[walletIDStr]
				if !exists {
					log.Warn("Failed to detect wallet change; wallet not found")
					continue
				}

				if event.Op&fsnotify.Create == fsnotify.Create { // in case adding a new account
					// new account has been added
					jw.handleAddAccountChange(walletScan, accountID)
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename { // in case a specific account was deleted
					jw.handleRemoveAccountChange(accountIDStr)
				}

			case err, ok := <-jw.watcher.Errors:
				if !ok {
					log.WithError(err).Warn("watcher warn")
					continue
				}
				log.WithError(err).Debug("watcher error")
			}
		}
	}()
}

