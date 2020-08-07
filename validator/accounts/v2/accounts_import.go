package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/k0kubun/go-ansi"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var derivationPathRegex = regexp.MustCompile("m_12381_3600_([0-9]+)_([0-9]+)_([0-9]+)")

// byDerivationPath implements sort.Interface based on a
// derivation path present in a keystore filename, if any. This
// will allow us to sort filenames such as keystore-m_12381_3600_1_0_0.json
// in a directory and import them nicely in order of the derivation path.
type byDerivationPath []string

func (fileNames byDerivationPath) Len() int { return len(fileNames) }
func (fileNames byDerivationPath) Less(i, j int) bool {
	// We check if file name at index i has a derivation path
	// in the filename. If it does not, then it is not less than j, and
	// we should swap it towards the end of the sorted list.
	if !derivationPathRegex.MatchString(fileNames[i]) {
		return false
	}
	derivationPathA := derivationPathRegex.FindString(fileNames[i])
	derivationPathB := derivationPathRegex.FindString(fileNames[j])
	if derivationPathA == "" {
		return false
	}
	if derivationPathB == "" {
		return true
	}
	a, err := strconv.Atoi(accountIndexFromFileName(derivationPathA))
	if err != nil {
		return false
	}
	b, err := strconv.Atoi(accountIndexFromFileName(derivationPathB))
	if err != nil {
		return false
	}
	return a < b
}

func (fileNames byDerivationPath) Swap(i, j int) {
	fileNames[i], fileNames[j] = fileNames[j], fileNames[i]
}

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
	km, err := direct.NewKeymanager(cliCtx, wallet, directCfg)
	if err != nil {
		return err
	}
	keysDir, err := inputDirectory(cliCtx, importKeysDirPromptText, flags.KeysDirFlag)
	if err != nil {
		return errors.Wrap(err, "could not parse keys directory")
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
		keystoreFileNames := make([]string, 0)
		for i := 0; i < len(files); i++ {
			if files[i].IsDir() {
				continue
			}
			if !strings.HasPrefix(files[i].Name(), "keystore") {
				continue
			}
			keystoreFileNames = append(keystoreFileNames, files[i].Name())
		}
		// Sort the imported keystores by derivation path if they
		// specify this value in their filename.
		sort.Sort(byDerivationPath(keystoreFileNames))
		for _, name := range keystoreFileNames {
			keystore, err := wallet.readKeystoreFile(ctx, filepath.Join(keysDir, name))
			if err != nil {
				return errors.Wrapf(err, "could not import keystore at path: %s", name)
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

func (w *Wallet) enterPasswordForAllAccounts(cliCtx *cli.Context, accountNames []string, pubKeys [][]byte) error {
	au := aurora.NewAurora(true)
	var password string
	var err error
	if cliCtx.IsSet(flags.AccountPasswordFileFlag.Name) {
		passwordFilePath := cliCtx.String(flags.AccountPasswordFileFlag.Name)
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return err
		}
		password = string(data)
		for i := 0; i < len(accountNames); i++ {
			err = w.checkPasswordForAccount(accountNames[i], password)
			if err != nil && strings.Contains(err.Error(), "invalid checksum") {
				return fmt.Errorf("invalid password for account with public key %#x", pubKeys[i])
			}
			if err != nil {
				return err
			}
		}
	} else {
		password, err = inputWeakPassword(
			cliCtx,
			flags.AccountPasswordFileFlag,
			"Enter the password for your imported accounts",
		)
		fmt.Println("Importing accounts, this may take a while...")
		bar := progressbar.NewOptions(
			len(accountNames),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
			progressbar.OptionOnCompletion(func() { fmt.Println() }),
			progressbar.OptionSetDescription("Importing accounts"),
		)
		for i := 0; i < len(accountNames); i++ {
			// We check if the individual account unlocks with the global password.
			err = w.checkPasswordForAccount(accountNames[i], password)
			if err != nil && strings.Contains(err.Error(), "invalid checksum") {
				// If the password fails for an individual account, we ask the user to input
				// that individual account's password until it succeeds.
				_, err := w.askUntilPasswordConfirms(cliCtx, accountNames[i], pubKeys[i])
				if err != nil {
					return err
				}
				if err := bar.Add(1); err != nil {
					return errors.Wrap(err, "could not add to progress bar")
				}
				continue
			}
			if err != nil {
				return err
			}
			fmt.Printf("Finished importing %#x\n", au.BrightMagenta(bytesutil.Trunc(pubKeys[i])))
			if err := bar.Add(1); err != nil {
				return errors.Wrap(err, "could not add to progress bar")
			}
		}
	}
	return nil
}

func (w *Wallet) askUntilPasswordConfirms(cliCtx *cli.Context, accountName string, pubKey []byte) (string, error) {
	// Loop asking for the password until the user enters it correctly.
	var password string
	var err error
	for {
		password, err = inputWeakPassword(
			cliCtx,
			flags.AccountPasswordFileFlag,
			fmt.Sprintf(passwordForAccountPromptText, bytesutil.Trunc(pubKey)),
		)
		if err != nil {
			return "", errors.Wrap(err, "could not input password")
		}
		err = w.checkPasswordForAccount(accountName, password)
		if err != nil && strings.Contains(err.Error(), "invalid checksum") {
			fmt.Println(au.Red("Incorrect password entered, please try again"))
			continue
		}
		if err != nil {
			return "", err
		}
		break
	}
	return password, nil
}

func (w *Wallet) checkPasswordForAccount(accountName string, password string) error {
	encoded, err := w.ReadFileAtPath(context.Background(), accountName, direct.KeystoreFileName)
	if err != nil {
		return errors.Wrap(err, "could not read keystore file")
	}
	keystoreJSON := &v2keymanager.Keystore{}
	if err := json.Unmarshal(encoded, &keystoreJSON); err != nil {
		return errors.Wrap(err, "could not decode json")
	}
	decryptor := keystorev4.New()
	_, err = decryptor.Decrypt(keystoreJSON.Crypto, password)
	if err != nil {
		return errors.Wrap(err, "could not decrypt keystore")
	}
	return nil
}

// Extracts the account index, j, from a derivation path in a file name
// with the format m_12381_3600_j_0_0.
func accountIndexFromFileName(derivationPath string) string {
	derivationPath = derivationPath[13:]
	accIndexEnd := strings.Index(derivationPath, "_")
	return derivationPath[:accIndexEnd]
}
