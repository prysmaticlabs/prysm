package accounts

import (
	"context"

	"github.com/pkg/errors"
	grpcutil "github.com/prysmaticlabs/prysm/v3/api/grpc"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote"
	"google.golang.org/grpc"
)

// NewCLIManager allows for managing validator accounts via CLI commands.
func NewCLIManager(opts ...Option) (*AccountsCLIManager, error) {
	acc := &AccountsCLIManager{}
	for _, opt := range opts {
		if err := opt(acc); err != nil {
			return nil, err
		}
	}
	return acc, nil
}

// AccountsCLIManager defines a struct capable of performing various validator
// wallet & account operations via the command line.
type AccountsCLIManager struct {
	keymanager           keymanager.IKeymanager
	keymanagerOpts       *remote.KeymanagerOpts
	wallet               *wallet.Wallet
	walletDir            string
	mnemonic             string
	walletPassword       string
	beaconRPCProvider    string
	keysDir              string
	backupsDir           string
	passwordFilePath     string
	backupsPassword      string
	privateKeyFile       string
	mnemonic25thWord     string
	rawPubKeys           [][]byte
	grpcHeaders          []string
	dialOpts             []grpc.DialOption
	filteredPubKeys      []bls.PublicKey
	formattedPubKeys     []string
	walletKeyCount       int
	numAccounts          int
	keymanagerKind       keymanager.Kind
	skipMnemonicConfirm  bool
	readPasswordFile     bool
	importPrivateKeys    bool
	deletePublicKeys     bool
	listValidatorIndices bool
	showPrivateKeys      bool
	showDepositData      bool
}

func (acm *AccountsCLIManager) prepareBeaconClients(ctx context.Context) (*ethpb.BeaconNodeValidatorClient, *ethpb.NodeClient, error) {
	if acm.dialOpts == nil {
		return nil, nil, errors.New("failed to construct dial options for beacon clients")
	}

	ctx = grpcutil.AppendHeaders(ctx, acm.grpcHeaders)
	conn, err := grpc.DialContext(ctx, acm.beaconRPCProvider, acm.dialOpts...)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not dial endpoint %s", acm.beaconRPCProvider)
	}
	validatorClient := ethpb.NewBeaconNodeValidatorClient(conn)
	nodeClient := ethpb.NewNodeClient(conn)
	return &validatorClient, &nodeClient, nil
}
