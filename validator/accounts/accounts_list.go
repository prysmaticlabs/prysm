package accounts

import (
	"context"
	"fmt"
	"math"
	"path/filepath"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	grpcutil "github.com/prysmaticlabs/prysm/api/grpc"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/validator/accounts/petnames"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote"
	"google.golang.org/grpc"
)

// List allows for listing all accounts in the wallet and pretty-printing them.
func (acc *AccountsCLIManager) List(ctx context.Context) error {
	if acc.listValidatorIndices {
		client, _, err := acc.prepareBeaconClients(ctx)
		if err != nil {
			return err
		}
		return listValidatorIndices(ctx, acc.keymanager, *client)
	}

	switch acc.wallet.KeymanagerKind() {
	case keymanager.Imported:
		km, ok := acc.keymanager.(*imported.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		if err := listImportedKeymanagerAccounts(ctx, acc.showDepositData, acc.showPrivateKeys, km); err != nil {
			return errors.Wrap(err, "could not list validator accounts with imported keymanager")
		}
	case keymanager.Derived:
		km, ok := acc.keymanager.(*derived.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		if err := listDerivedKeymanagerAccounts(ctx, acc.showPrivateKeys, km); err != nil {
			return errors.Wrap(err, "could not list validator accounts with derived keymanager")
		}
	case keymanager.Remote:
		km, ok := acc.keymanager.(*remote.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		if err := listRemoteKeymanagerAccounts(ctx, acc.wallet, km, km.KeymanagerOpts()); err != nil {
			return errors.Wrap(err, "could not list validator accounts with remote keymanager")
		}
	default:
		return fmt.Errorf(errKeymanagerNotSupported, acc.wallet.KeymanagerKind().String())
	}
	return nil
}

func (acc *AccountsCLIManager) prepareBeaconClients(ctx context.Context) (*ethpb.BeaconNodeValidatorClient, *ethpb.NodeClient, error) {
	if acc.dialOpts == nil {
		return nil, nil, errors.New("failed to construct dial options")
	}

	ctx = grpcutil.AppendHeaders(ctx, acc.grpcHeaders)

	conn, err := grpc.DialContext(ctx, acc.beaconRPCProvider, acc.dialOpts...)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not dial endpoint %s", acc.beaconRPCProvider)
	}
	validatorClient := ethpb.NewBeaconNodeValidatorClient(conn)
	nodeClient := ethpb.NewNodeClient(conn)
	return &validatorClient, &nodeClient, nil
}

func listImportedKeymanagerAccounts(
	ctx context.Context,
	showDepositData,
	showPrivateKeys bool,
	keymanager *imported.Keymanager,
) error {
	// We initialize the wallet's keymanager.
	accountNames, err := keymanager.ValidatingAccountNames()
	if err != nil {
		return errors.Wrap(err, "could not fetch account names")
	}
	numAccounts := au.BrightYellow(len(accountNames))
	fmt.Printf("(keymanager kind) %s\n", au.BrightGreen("imported wallet").Bold())
	fmt.Println("")
	if len(accountNames) == 1 {
		fmt.Printf("Showing %d validator account\n", numAccounts)
	} else {
		fmt.Printf("Showing %d validator accounts\n", numAccounts)
	}
	fmt.Println(
		au.BrightRed("View the eth1 deposit transaction data for your accounts " +
			"by running `validator accounts list --show-deposit-data`"),
	)

	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	var privateKeys [][32]byte
	if showPrivateKeys {
		privateKeys, err = keymanager.FetchValidatingPrivateKeys(ctx)
		if err != nil {
			return errors.Wrap(err, "could not fetch private keys")
		}
	}
	for i := 0; i < len(accountNames); i++ {
		fmt.Println("")
		fmt.Printf("%s | %s\n", au.BrightBlue(fmt.Sprintf("Account %d", i)).Bold(), au.BrightGreen(accountNames[i]).Bold())
		fmt.Printf("%s %#x\n", au.BrightMagenta("[validating public key]").Bold(), pubKeys[i])
		if showPrivateKeys {
			if len(privateKeys) > i {
				fmt.Printf("%s %#x\n", au.BrightRed("[validating private key]").Bold(), privateKeys[i])
			}
		}
		if !showDepositData {
			continue
		}
		fmt.Printf(
			"%s\n",
			au.BrightRed("If you imported your account coming from the eth2 launchpad, you will find your "+
				"deposit_data.json in the eth2.0-deposit-cli's validator_keys folder"),
		)
		fmt.Println("")
	}
	fmt.Println("")
	return nil
}

func listDerivedKeymanagerAccounts(
	ctx context.Context,
	showPrivateKeys bool,
	keymanager *derived.Keymanager,
) error {
	au := aurora.NewAurora(true)
	fmt.Printf("(keymanager kind) %s\n", au.BrightGreen("derived, (HD) hierarchical-deterministic").Bold())
	fmt.Printf("(derivation format) %s\n", au.BrightGreen(derived.DerivationPathFormat).Bold())
	validatingPubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	var validatingPrivateKeys [][32]byte
	if showPrivateKeys {
		validatingPrivateKeys, err = keymanager.FetchValidatingPrivateKeys(ctx)
		if err != nil {
			return errors.Wrap(err, "could not fetch validating private keys")
		}
	}
	accountNames, err := keymanager.ValidatingAccountNames(ctx)
	if err != nil {
		return err
	}
	if len(accountNames) == 1 {
		fmt.Print("Showing 1 validator account\n")
	} else if len(accountNames) == 0 {
		fmt.Print("No accounts found\n")
		return nil
	} else {
		fmt.Printf("Showing %d validator accounts\n", len(accountNames))
	}
	for i := 0; i < len(accountNames); i++ {
		fmt.Println("")
		validatingKeyPath := fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i)

		// Retrieve the withdrawal key account metadata.
		fmt.Printf("%s | %s\n", au.BrightBlue(fmt.Sprintf("Account %d", i)).Bold(), au.BrightGreen(accountNames[i]).Bold())
		// Retrieve the validating key account metadata.
		fmt.Printf("%s %#x\n", au.BrightCyan("[validating public key]").Bold(), validatingPubKeys[i])
		if showPrivateKeys && validatingPrivateKeys != nil {
			fmt.Printf("%s %#x\n", au.BrightRed("[validating private key]").Bold(), validatingPrivateKeys[i])
		}
		fmt.Printf("%s %s\n", au.BrightCyan("[derivation path]").Bold(), validatingKeyPath)
		fmt.Println(" ")
	}
	return nil
}

func listRemoteKeymanagerAccounts(
	ctx context.Context,
	w *wallet.Wallet,
	keymanager keymanager.IKeymanager,
	opts *remote.KeymanagerOpts,
) error {
	au := aurora.NewAurora(true)
	fmt.Printf("(keymanager kind) %s\n", au.BrightGreen("remote signer").Bold())
	fmt.Printf(
		"(configuration file path) %s\n",
		au.BrightGreen(filepath.Join(w.AccountsDir(), wallet.KeymanagerConfigFileName)).Bold(),
	)
	fmt.Println(" ")
	fmt.Printf("%s\n", au.BrightGreen("Configuration options").Bold())
	fmt.Println(opts)
	validatingPubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	if len(validatingPubKeys) == 1 {
		fmt.Print("Showing 1 validator account\n")
	} else if len(validatingPubKeys) == 0 {
		fmt.Print("No accounts found\n")
		return nil
	} else {
		fmt.Printf("Showing %d validator accounts\n", len(validatingPubKeys))
	}
	for i := 0; i < len(validatingPubKeys); i++ {
		fmt.Println("")
		fmt.Printf(
			"%s\n", au.BrightGreen(petnames.DeterministicName(validatingPubKeys[i][:], "-")).Bold(),
		)
		// Retrieve the validating key account metadata.
		fmt.Printf("%s %#x\n", au.BrightCyan("[validating public key]").Bold(), validatingPubKeys[i])
		fmt.Println(" ")
	}
	return nil
}

func listValidatorIndices(ctx context.Context, km keymanager.IKeymanager, client ethpb.BeaconNodeValidatorClient) error {
	pubKeys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get validating public keys")
	}
	var pks [][]byte
	for i := range pubKeys {
		pks = append(pks, pubKeys[i][:])
	}
	req := &ethpb.MultipleValidatorStatusRequest{PublicKeys: pks}
	resp, err := client.MultipleValidatorStatus(ctx, req)
	if err != nil {
		return errors.Wrap(err, "could not request validator indices")
	}
	fmt.Println(au.BrightGreen("Validator indices:").Bold())
	for i, idx := range resp.Indices {
		if idx != math.MaxUint64 {
			fmt.Printf("%#x: %d\n", pubKeys[i][0:4], idx)
		}
	}
	return nil
}
