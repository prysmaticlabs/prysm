package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
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
	cfg, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	if err != nil {
		return err
	}
	directCfg, err := direct.UnmarshalConfigFile(cfg)
	if err != nil {
		return err
	}
	km, err := direct.NewKeymanager(ctx, wallet, directCfg)
	if err != nil {
		return err
	}
	keysDir, err := inputDirectory(cliCtx, importKeysDirPromptText, flags.KeysDirFlag)
	if err != nil {
		return errors.Wrap(err, "could not parse keys directory")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet")
	}
	isDir, err := hasDir(keysDir)
	if err != nil {
		return errors.Wrap(err, "could not determine if path is a directory")
	}

	keystoresImported := make([]*v2keymanager.Keystore, 0)

	// Consider that the keysDir might be a path to a specific file and handle accordingly.
	if isDir {
		files, err := ioutil.ReadDir(keysDir)
		if err != nil {
			return errors.Wrap(err, "could not read dir")
		}
		if len(files) == 0 {
			return fmt.Errorf("directory %s has no files, cannot import from it", keysDir)
		}
		for i := 0; i < len(files); i++ {
			if files[i].IsDir() {
				continue
			}
			if !strings.HasPrefix(files[i].Name(), "keystore") {
				continue
			}
			keystore, err := wallet.readKeystoreFile(ctx, filepath.Join(keysDir, files[i].Name()))
			if err != nil {
				return errors.Wrap(err, "could not import keystore")
			}
			keystoresImported = append(keystoresImported, keystore)
		}
	} else {
		keystore, err := wallet.readKeystoreFile(ctx, keysDir)
		if err != nil {
			return errors.Wrap(err, "could not import keystore")
		}
		keystoresImported = append(keystoresImported, keystore)
	}

	au := aurora.NewAurora(true)
	if err := km.ImportKeystores(cliCtx, keystoresImported); err != nil {
		return errors.Wrap(err, "could not import all keystores")
	}
	fmt.Printf(
		"Successfully imported %s accounts, view all of them by running accounts-v2 list\n",
		au.BrightMagenta(strconv.Itoa(len(keystoresImported))),
	)
	return nil
}

func (w *Wallet) readKeystoreFile(ctx context.Context, keystoreFilePath string) (*v2keymanager.Keystore, error) {
	keystoreBytes, err := ioutil.ReadFile(keystoreFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keystore file")
	}
	keystoreFile := &v2keymanager.Keystore{}
	if err := json.Unmarshal(keystoreBytes, keystoreFile); err != nil {
		return nil, errors.Wrap(err, "could not decode keystore json")
	}
	return keystoreFile, nil
}
