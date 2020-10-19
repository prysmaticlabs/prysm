package accounts

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts")

// CreateAccountConfig to run the create account function.
type CreateAccountConfig struct {
	Wallet      *wallet.Wallet
	NumAccounts int64
}

// CreateAccountCli creates a new validator account from user input by opening
// a wallet from the user's specified path. This uses the CLI to extract information
// to perform account creation.
func CreateAccountCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, CreateAndSaveWalletCli)
	if err != nil {
		return err
	}
	numAccounts := cliCtx.Int64(flags.NumAccountsFlag.Name)
	log.Info("Creating a new account...")
	return CreateAccount(cliCtx.Context, &CreateAccountConfig{
		Wallet:      w,
		NumAccounts: numAccounts,
	})
}

// CreateAccount creates a new validator account from user input by opening
// a wallet from the user's specified path.
func CreateAccount(ctx context.Context, cfg *CreateAccountConfig) error {
	km, err := cfg.Wallet.InitializeKeymanager(ctx, false /* skip mnemonic confirm */)
	if err != nil && strings.Contains(err.Error(), "invalid checksum") {
		return errors.New("wrong wallet password entered")
	}
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	switch cfg.Wallet.KeymanagerKind() {
	case keymanager.Remote:
		return errors.New("cannot create a new account for a remote keymanager")
	case keymanager.Imported:
		km, ok := km.(*imported.Keymanager)
		if !ok {
			return errors.New("not a imported keymanager")
		}
		// Create a new validator account using the specified keymanager.
		if _, _, err := km.CreateAccount(ctx); err != nil {
			return errors.Wrap(err, "could not create account in wallet")
		}
	case keymanager.Derived:
		km, ok := km.(*derived.Keymanager)
		if !ok {
			return errors.New("not a derived keymanager")
		}
		startNum := km.NextAccountNumber()
		if cfg.NumAccounts == 1 {
			if _, _, err := km.CreateAccount(ctx); err != nil {
				return errors.Wrap(err, "could not create account in wallet")
			}
		} else {
			for i := 0; i < int(cfg.NumAccounts); i++ {
				if _, _, err := km.CreateAccount(ctx); err != nil {
					return errors.Wrap(err, "could not create account in wallet")
				}
			}
			log.Infof(
				"Successfully created %d accounts. Please use accounts list to view details for accounts %d through %d",
				cfg.NumAccounts,
				startNum,
				startNum+uint64(cfg.NumAccounts)-1,
			)
		}
	default:
		return fmt.Errorf("keymanager kind %s not supported", cfg.Wallet.KeymanagerKind())
	}
	return nil
}

// DepositDataJSON creates a raw map to match the deposit_data.json file format
// from the official eth2.0-deposit-cli https://github.com/ethereum/eth2.0-deposit-cli.
// The reason we utilize this map is to ensure we match the format of
// the eth2 deposit cli, which utilizes snake case and hex strings to represent binary data.
// Our gRPC gateway instead uses camel case and base64, which is why we use this workaround.
func DepositDataJSON(depositData *ethpb.Deposit_Data) (map[string]string, error) {
	depositMessage := &pb.DepositMessage{
		Pubkey:                depositData.PublicKey,
		WithdrawalCredentials: depositData.WithdrawalCredentials,
		Amount:                depositData.Amount,
	}
	depositMessageRoot, err := depositMessage.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	depositDataRoot, err := depositData.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	data := make(map[string]string)
	data["pubkey"] = fmt.Sprintf("%x", depositData.PublicKey)
	data["withdrawal_credentials"] = fmt.Sprintf("%x", depositData.WithdrawalCredentials)
	data["amount"] = fmt.Sprintf("%d", depositData.Amount)
	data["signature"] = fmt.Sprintf("%x", depositData.Signature)
	data["deposit_message_root"] = fmt.Sprintf("%x", depositMessageRoot)
	data["deposit_data_root"] = fmt.Sprintf("%x", depositDataRoot)
	data["fork_version"] = fmt.Sprintf("%x", params.BeaconConfig().GenesisForkVersion)
	return data, nil
}
