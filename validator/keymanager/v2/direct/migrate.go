package direct

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/k0kubun/go-ansi"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/schollz/progressbar/v3"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

// Migrates the old format for validator direct-keymanaged accounts into a new, more
// efficient format which stores only a single keystore all accounts, encrypted using
// a high-entropy password. This allows for incredibly fast startup-time, requiring only
// a single decryption operation to obtain all validator accounts. This migration process
// is meant to happen only once, ensuring all future restarts of the validator client utilize
// the fast, efficient format.
//
// Old format:
//  wallet/
//    direct/
//      perfectly-intense-mosquito/
//        keystore-2909299.json
//      personally-conscious-echidna/
//        keystore-20390922.json
//  passwords/
//    perfectly-intense-mosquito.pass
//    personally-conscious-echidna.pass
//
// New format:
//  wallet/
//    direct/
//      accounts/
//        all-accounts.keystore-2983823.json
func (dr *Keymanager) migrateToSingleKeystore(ctx context.Context) error {
	accountNames, err := dr.wallet.ListDirs()
	if err != nil {
		return err
	}
	if len(accountNames) == 0 {
		return nil
	}
	for _, name := range accountNames {
		// If the user is already using the single keystore format,
		// we have no need to migrate and we exit normally.
		if strings.Contains(name, AccountsPath) {
			return nil
		}
	}
	au := aurora.NewAurora(true)
	log.Infof(
		"Now migrating accounts to a more efficient format, this is a %s setup\n",
		au.BrightRed("one-time"),
	)
	bar := initializeProgressBar(len(accountNames))
	decryptor := keystorev4.New()
	privKeys := make([][]byte, len(accountNames))
	pubKeys := make([][]byte, len(accountNames))
	// Next up, we retrieve every single keystore for each
	// account and attempt to unlock.
	for i, name := range accountNames {
		password, err := dr.wallet.ReadPasswordFromDisk(ctx, name+PasswordFileSuffix)
		if err != nil {
			return errors.Wrapf(err, "could not read password for account %s", name)
		}
		encoded, err := dr.wallet.ReadFileAtPath(ctx, name, KeystoreFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read keystore file for account %s", name)
		}
		keystoreFile := &v2keymanager.Keystore{}
		if err := json.Unmarshal(encoded, keystoreFile); err != nil {
			return errors.Wrapf(err, "could not decode keystore file for account %s", name)
		}
		// We extract the validator signing private key from the keystore
		// by utilizing the password.
		privKeyBytes, err := decryptor.Decrypt(keystoreFile.Crypto, password)
		if err != nil {
			return errors.Wrapf(err, "could not decrypt signing key for account %s", name)
		}
		publicKeyBytes, err := hex.DecodeString(keystoreFile.Pubkey)
		if err != nil {
			return err
		}
		privKeys[i] = privKeyBytes
		pubKeys[i] = publicKeyBytes
		if err := bar.Add(1); err != nil {
			return err
		}
	}
	accountsKeystore, err := dr.createAccountsKeystore(ctx, privKeys, pubKeys)
	if err != nil {
		return err
	}
	encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
	if err != nil {
		return err
	}
	fileName := fmt.Sprintf(accountsKeystoreFileNameFormat, roughtime.Now().Unix())
	return dr.wallet.WriteFileAtPath(ctx, AccountsPath, fileName, encodedAccounts)
}

func initializeProgressBar(numItems int) *progressbar.ProgressBar {
	return progressbar.NewOptions(
		numItems,
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
		progressbar.OptionSetDescription("Migrating accounts..."),
	)
}
