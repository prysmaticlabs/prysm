package accounts

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/cmd"
	"github.com/prysmaticlabs/prysm/v5/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v5/validator/client"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager/local"
	"github.com/prysmaticlabs/prysm/v5/validator/node"
	"github.com/urfave/cli/v2"
)

func Exit(c *cli.Context, r io.Reader) error {
	var w *wallet.Wallet
	var km keymanager.IKeymanager
	var err error
	dialOpts := client.ConstructDialOptions(
		c.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name),
		c.String(flags.CertFlag.Name),
		c.Uint(flags.GRPCRetriesFlag.Name),
		c.Duration(flags.GRPCRetryDelayFlag.Name),
	)
	grpcHeaders := strings.Split(c.String(flags.GRPCHeadersFlag.Name), ",")
	beaconRPCProvider := c.String(flags.BeaconRPCProviderFlag.Name)
	if !c.IsSet(flags.Web3SignerURLFlag.Name) && !c.IsSet(flags.WalletDirFlag.Name) && !c.IsSet(flags.InteropNumValidators.Name) {
		return errors.Errorf("No validators found, please provide a prysm wallet directory via flag --%s "+
			"or a remote signer location with corresponding public keys via flags --%s and --%s ",
			flags.WalletDirFlag.Name,
			flags.Web3SignerURLFlag.Name,
			flags.Web3SignerPublicValidatorKeysFlag,
		)
	}
	if c.IsSet(flags.InteropNumValidators.Name) {
		km, err = local.NewInteropKeymanager(c.Context, c.Uint64(flags.InteropStartIndex.Name), c.Uint64(flags.InteropNumValidators.Name))
		if err != nil {
			return errors.Wrap(err, "could not generate interop keys for key manager")
		}
		w = &wallet.Wallet{}
	} else if c.IsSet(flags.Web3SignerURLFlag.Name) {
		config, err := node.Web3SignerConfig(c)
		if err != nil {
			return errors.Wrapf(err, "could not configure remote signer")
		}
		w, km, err = walletWithWeb3SignerKeymanager(c, config)
		if err != nil {
			return err
		}
	} else {
		w, km, err = walletWithKeymanager(c)
		if err != nil {
			return err
		}
	}

	opts := []accounts.Option{
		accounts.WithWallet(w),
		accounts.WithKeymanager(km),
		accounts.WithGRPCDialOpts(dialOpts),
		accounts.WithBeaconRPCProvider(beaconRPCProvider),
		accounts.WithBeaconRESTApiProvider(c.String(flags.BeaconRESTApiProviderFlag.Name)),
		accounts.WithGRPCHeaders(grpcHeaders),
		accounts.WithExitJSONOutputPath(c.String(flags.VoluntaryExitJSONOutputPathFlag.Name)),
	}
	// Get full set of public keys from the keymanager.
	validatingPublicKeys, err := km.FetchValidatingPublicKeys(c.Context)
	if err != nil {
		return err
	}
	if len(validatingPublicKeys) == 0 {
		return errors.New("wallet is empty, no accounts to delete")
	}
	// Filter keys either from CLI flag or from interactive session.
	rawPubKey, formattedPubKeys, err := accounts.FilterExitAccountsFromUserInput(c, r, validatingPublicKeys, c.Bool(flags.ForceExitFlag.Name))
	if err != nil {
		return errors.Wrap(err, "could not filter public keys for deletion")
	}
	opts = append(opts, accounts.WithRawPubKeys(rawPubKey))
	opts = append(opts, accounts.WithFormattedPubKeys(formattedPubKeys))
	acc, err := accounts.NewCLIManager(opts...)
	if err != nil {
		return err
	}
	return acc.Exit(c.Context)
}
