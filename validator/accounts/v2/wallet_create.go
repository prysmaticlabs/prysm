package v2

import (
	"github.com/urfave/cli/v2"

	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

func CreateWallet(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil {
		log.Fatalf("Could not parse wallet directory: %v", err)
	}
	// Check if the user has a wallet at the specified path.
	// If a user does not have a wallet, we instantiate one
	// based on specified options.
	walletExists, err := hasDir(walletDir)
	if err != nil {
		log.Fatal(err)
	}
	if walletExists {
		log.Fatal(
			"You already have a wallet at the specified path. You can " +
				"edit your wallet configuration by running prysm validator wallet-v2 edit",
		)
	}
	// Determine the desired keymanager kind for the wallet from user input.
	keymanagerKind, err := inputKeymanagerKind(cliCtx)
	if err != nil {
		log.Fatalf("Could not select keymanager kind: %v", err)
	}
	switch keymanagerKind {
	case v2keymanager.Direct:
		log.Fatal("Direct keymanager is not yet supported")
	case v2keymanager.Derived:
		log.Fatal("Derived keymanager is not yet supported")
	case v2keymanager.Remote:
		log.Fatal("Derived keymanager is not yet supported")
	default:
		log.Fatalf("Keymanager type %s is not supported", keymanagerKind.String())
	}
	return nil
}
