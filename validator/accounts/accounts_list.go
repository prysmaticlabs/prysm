package accounts

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/urfave/cli/v2"
)

// ListAccountsCli displays all available validator accounts in a Prysm wallet.
func ListAccountsCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	// TODO(#9883) - Remove this when we have a better way to handle this. this is fine.
	// genesis root is not set here which is used for sign function, but fetch keys should be fine.
	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil && strings.Contains(err.Error(), keymanager.IncorrectPasswordErrMsg) {
		return errors.New("wrong wallet password entered")
	}
	if err != nil {
		return errors.Wrap(err, ErrCouldNotInitializeKeymanager)
	}
	showDepositData := cliCtx.Bool(flags.ShowDepositDataFlag.Name)
	showPrivateKeys := cliCtx.Bool(flags.ShowPrivateKeysFlag.Name)
	listIndices := cliCtx.Bool(flags.ListValidatorIndices.Name)

	if listIndices {
		client, _, err := prepareClients(cliCtx)
		if err != nil {
			return err
		}
		return listValidatorIndices(cliCtx.Context, km, *client)
	}
	return km.ListKeymanagerAccounts(cliCtx.Context,
		keymanager.ListKeymanagerAccountConfig{
			ShowDepositData:          showDepositData,
			ShowPrivateKeys:          showPrivateKeys,
			WalletAccountsDir:        w.AccountsDir(),
			KeymanagerConfigFileName: wallet.KeymanagerConfigFileName,
		})
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
