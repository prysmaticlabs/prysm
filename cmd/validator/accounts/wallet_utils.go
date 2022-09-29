package accounts

import (
	"context"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	grpcutil "github.com/prysmaticlabs/prysm/v3/api/grpc"
	"github.com/prysmaticlabs/prysm/v3/cmd"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/client"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/node"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

func walletWithKeymanager(c *cli.Context) (*wallet.Wallet, keymanager.IKeymanager, error) {
	if c.IsSet(flags.Web3SignerURLFlag.Name) {
		config, err := node.Web3SignerConfig(c)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not configure web3signer")
		}
		w := wallet.NewWalletForWeb3Signer()

		grpcHeaders := strings.Split(c.String(flags.GrpcHeadersFlag.Name), ",")
		ctx := grpcutil.AppendHeaders(context.Background(), grpcHeaders)
		dialOpts := client.ConstructDialOptions(
			c.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name),
			c.String(flags.CertFlag.Name),
			c.Uint(flags.GrpcRetriesFlag.Name),
			c.Duration(flags.GrpcRetryDelayFlag.Name),
		)
		beaconRPCProvider := c.String(flags.BeaconRPCProviderFlag.Name)
		conn, err := grpc.DialContext(ctx, beaconRPCProvider, dialOpts...)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not dial endpoint %s", beaconRPCProvider)
		}
		nodeClient := ethpb.NewNodeClient(conn)
		resp, err := nodeClient.GetGenesis(ctx, &empty.Empty{})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get genesis info")
		}
		config.GenesisValidatorsRoot = resp.GenesisValidatorsRoot
		km, err := w.InitializeKeymanager(c.Context, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: config})
		if err != nil {
			return nil, nil, err
		}
		return w, km, nil
	} else {
		w, err := wallet.OpenWalletOrElseCli(c, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
			return nil, wallet.ErrNoWalletFound
		})
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not open wallet")
		}
		km, err := w.InitializeKeymanager(c.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
		if err != nil && strings.Contains(err.Error(), keymanager.IncorrectPasswordErrMsg) {
			return nil, nil, errors.New("wrong wallet password entered")
		}
		if err != nil {
			return nil, nil, errors.Wrap(err, accounts.ErrCouldNotInitializeKeymanager)
		}
		return w, km, nil
	}

}
