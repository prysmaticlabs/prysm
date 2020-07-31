package v2

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/urfave/cli/v2"
)

// ImportAccount uses the archived account made from ExportAccount to import an account and
// asks the users for account passwords.
func ImportAccount(cliCtx *cli.Context) error {
	ctx := context.Background()
	wallet, err := createOrOpenWallet(cliCtx, func(cliCtx *cli.Context) (*Wallet, error) {
		w, err := NewWallet(cliCtx, v2keymanager.Direct)
		if err != nil && !errors.Is(err, ErrWalletExists) {
			return nil, errors.Wrap(err, "could not create new wallet")
		}
		if err = createDirectKeymanagerWallet(cliCtx, w); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet")
		}
		log.WithField("wallet-path", w.walletDir).Info(
			"Successfully created new wallet",
		)
		return w, err
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize wallet")
	}
	if wallet.KeymanagerKind() != v2keymanager.Direct {
		return errors.New(
			"only non-HD wallets can import accounts, try creating a new wallet with wallet-v2 create",
		)
	}
	keysDir, err := inputDirectory(cliCtx, importKeysDirPromptText, flags.KeysDirFlag)
	if err != nil {
		return errors.Wrap(err, "could not parse keys directory")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet")
	}
	accountsImported := make([]string, 0)
	pubKeysImported := make([][]byte, 0)
	isDir, err := hasDir(keysDir)
	if err != nil {
		return errors.Wrap(err, "could not determine if path is a directory")
	}

	// Consider that the keysDir might be a path to a specific file and handle accordingly.
	if isDir {
		files, err := ioutil.ReadDir(keysDir)
		if err != nil {
			return errors.Wrap(err, "could not read dir")
		}
		for i := 0; i < len(files); i++ {
			if files[i].IsDir() {
				continue
			}
			if !strings.HasPrefix(files[i].Name(), "keystore") {
				continue
			}
			accountName, pubKey, err := wallet.importKeystore(ctx, filepath.Join(keysDir, files[i].Name()))
			if err != nil {
				return errors.Wrap(err, "could not import keystore")
			}
			accountsImported = append(accountsImported, accountName)
			pubKeysImported = append(pubKeysImported, pubKey)
		}
	} else {
		accountName, pubKey, err := wallet.importKeystore(ctx, keysDir)
		if err != nil {
			return errors.Wrap(err, "could not import keystore")
		}
		accountsImported = append(accountsImported, accountName)
		pubKeysImported = append(pubKeysImported, pubKey)
	}

	au := aurora.NewAurora(true)
	formattedPubkeys := make([]string, len(pubKeysImported))
	for i, pk := range pubKeysImported {
		formattedPubkeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pk))
	}
	fmt.Printf("Importing accounts: %s\n", au.BrightGreen(strings.Join(formattedPubkeys, ", ")))
	if err := wallet.enterPasswordForAllAccounts(cliCtx, accountsImported, pubKeysImported); err != nil {
		return errors.Wrap(err, "could not verify password for keystore")
	}
	fmt.Printf(
		"Successfully imported %s accounts, view all of them by running accounts-v2 list\n",
		au.BrightMagenta(strconv.Itoa(len(pubKeysImported))),
	)
	return nil
}

func (w *Wallet) importKeystore(ctx context.Context, keystoreFilePath string) (string, []byte, error) {
	keystoreBytes, err := ioutil.ReadFile(keystoreFilePath)
	if err != nil {
		return "", nil, errors.Wrap(err, "could not read keystore file")
	}
	keystoreFile := &v2keymanager.Keystore{}
	if err := json.Unmarshal(keystoreBytes, keystoreFile); err != nil {
		return "", nil, errors.Wrap(err, "could not decode keystore json")
	}
	pubKeyBytes, err := hex.DecodeString(keystoreFile.Pubkey)
	if err != nil {
		return "", nil, errors.Wrap(err, "could not decode public key string in keystore")
	}
	accountName := petnames.DeterministicName(pubKeyBytes, "-")
	keystoreFileName := filepath.Base(keystoreFilePath)
	if err := w.WriteFileAtPath(ctx, accountName, keystoreFileName, keystoreBytes); err != nil {
		return "", nil, errors.Wrap(err, "could not write keystore to account dir")
	}
	return accountName, pubKeyBytes, nil
}

func logAccountsImported(ctx context.Context, wallet *Wallet, keymanager *direct.Keymanager, accountNames []string) error {
	au := aurora.NewAurora(true)

	numAccounts := au.BrightYellow(len(accountNames))
	fmt.Println("")
	if len(accountNames) == 1 {
		fmt.Printf("Imported %d validator account\n", numAccounts)
	} else {
		fmt.Printf("Imported %d validator accounts\n", numAccounts)
	}
	for _, accountName := range accountNames {
		fmt.Println("")
		// Retrieve the account creation timestamp.
		keystoreFileName, err := wallet.FileNameAtPath(ctx, accountName, direct.KeystoreFileName)
		if err != nil {
			return errors.Wrapf(err, "could not get keystore file name for account: %s", accountName)
		}
		unixTimestamp, err := AccountTimestamp(keystoreFileName)
		if err != nil {
			return errors.Wrap(err, "could not get timestamp from keystore file name")
		}
		fmt.Printf("%s | Created %s\n", au.BrightGreen(accountName).Bold(), humanize.Time(unixTimestamp))

		publicKey, err := keymanager.PublicKeyForAccount(accountName)
		if err != nil {
			return errors.Wrap(err, "could not get public key")
		}
		fmt.Printf("%s %#x\n", au.BrightMagenta("[validating public key]").Bold(), publicKey)

		dirPath := au.BrightCyan("(wallet dir)")
		fmt.Printf("%s %s\n", dirPath, filepath.Join(wallet.AccountsDir(), accountName))
	}
	return nil
}
