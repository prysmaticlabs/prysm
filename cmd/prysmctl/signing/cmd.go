package signing

import (
	"os"

	"github.com/prysmaticlabs/prysm/v3/cmd"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/runtime/tos"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var Commands = []*cli.Command{
	{
		Name:  "sign",
		Usage: "signs a message and broadcasts it to the network through the beacon node",
		Subcommands: []*cli.Command{
			{
				Name:        "voluntary-exit",
				Description: "Performs a voluntary exit on selected accounts",
				Flags: cmd.WrapFlags([]cli.Flag{
					flags.WalletDirFlag,
					flags.WalletPasswordFileFlag,
					flags.AccountPasswordFileFlag,
					flags.VoluntaryExitPublicKeysFlag,
					flags.BeaconRPCProviderFlag,
					flags.Web3SignerURLFlag,
					flags.Web3SignerPublicValidatorKeysFlag,
					flags.InteropNumValidators,
					flags.InteropStartIndex,
					cmd.GrpcMaxCallRecvMsgSizeFlag,
					flags.CertFlag,
					flags.GrpcHeadersFlag,
					flags.GrpcRetriesFlag,
					flags.GrpcRetryDelayFlag,
					flags.ExitAllFlag,
					flags.ForceExitFlag,
					features.Mainnet,
					features.PraterTestnet,
					features.RopstenTestnet,
					features.SepoliaTestnet,
					cmd.AcceptTosFlag,
				}),
				Before: func(cliCtx *cli.Context) error {
					if err := cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags); err != nil {
						return err
					}
					if err := tos.VerifyTosAcceptedOrPrompt(cliCtx); err != nil {
						return err
					}
					return features.ConfigureValidator(cliCtx)
				},
				Action: func(cliCtx *cli.Context) error {
					if err := accounts.AccountsExit(cliCtx, os.Stdin); err != nil {
						log.WithError(err).Fatal("Could not perform voluntary exit")
					}
					return nil
				},
			},
		},
	},
}
