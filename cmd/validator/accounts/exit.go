package accounts

import (
	"context"
	"io"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	grpcutil "github.com/prysmaticlabs/prysm/v3/api/grpc"
	"github.com/prysmaticlabs/prysm/v3/cmd"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/client"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

func AccountsExit(c *cli.Context, r io.Reader) error {

	dialOpts := client.ConstructDialOptions(
		c.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name),
		c.String(flags.CertFlag.Name),
		c.Uint(flags.GrpcRetriesFlag.Name),
		c.Duration(flags.GrpcRetryDelayFlag.Name),
	)
	grpcHeaders := strings.Split(c.String(flags.GrpcHeadersFlag.Name), ",")
	beaconRPCProvider := c.String(flags.BeaconRPCProviderFlag.Name)
	ctx := grpcutil.AppendHeaders(context.Background(), grpcHeaders)
	conn, err := grpc.DialContext(ctx, beaconRPCProvider, dialOpts...)
	if err != nil {
		return errors.Wrapf(err, "could not dial endpoint %s", beaconRPCProvider)
	}
	nodeClient := ethpb.NewNodeClient(conn)
	resp, err := nodeClient.GetGenesis(c.Context, &empty.Empty{})
	if err != nil {
		return errors.Wrapf(err, "failed to get genesis info")
	}
	if err := conn.Close(); err != nil {
		log.WithError(err).Error("Failed to close connection")
	}
	w, km, err := walletWithKeymanager(c, resp.GenesisValidatorsRoot)
	if err != nil {
		return err
	}
	opts := []accounts.Option{
		accounts.WithWallet(w),
		accounts.WithKeymanager(km),
		accounts.WithGRPCDialOpts(dialOpts),
		accounts.WithBeaconRPCProvider(beaconRPCProvider),
		accounts.WithGRPCHeaders(grpcHeaders),
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
	rawPubKey, formattedPubKeys, err := accounts.FilterExitAccountsFromUserInput(c, r, validatingPublicKeys)
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
