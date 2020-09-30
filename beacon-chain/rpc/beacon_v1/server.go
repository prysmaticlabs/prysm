// Package beacon defines a gRPC beacon service implementation, providing
// useful endpoints for checking fetching chain-specific data such as
// blocks, committees, validators, assignments, and more.
package beacon_v1

import (
	"context"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type Server struct {
	BeaconDB                    db.ReadOnlyDatabase
	Ctx                         context.Context
	ChainStartFetcher           powchain.ChainStartFetcher
	HeadFetcher                 blockchain.HeadFetcher
	FinalizationFetcher         blockchain.FinalizationFetcher
	DepositFetcher              depositcache.DepositFetcher
	BlockFetcher                powchain.POWBlockFetcher
	GenesisTimeFetcher          blockchain.TimeFetcher
	GenesisFetcher              blockchain.GenesisFetcher
	StateNotifier               statefeed.Notifier
	BlockNotifier               blockfeed.Notifier
	AttestationNotifier         operation.Notifier
	Broadcaster                 p2p.Broadcaster
	AttestationsPool            attestations.Pool
	SlashingsPool               *slashings.Pool
	CanonicalStateChan          chan *pbp2p.BeaconState
	ChainStartChan              chan time.Time
	ReceivedAttestationsBuffer  chan *ethpb.Attestation
	CollectedAttestationsBuffer chan []*ethpb.Attestation
	StateGen                    *stategen.State
	SyncChecker                 sync.Checker
}

// GetGenesis fetches genesis chain information of Ethereum 2.0. Returns unix timestamp 0
// if a genesis time has yet to be determined.
func (ns *Server) GetGenesis(ctx context.Context, _ *ptypes.Empty) (*ethpb.GenesisResponse, error) {
	genesisTime := ns.GenesisTimeFetcher.GenesisTime()
	var defaultGenesisTime time.Time
	var gt *ptypes.Timestamp
	var err error
	if genesisTime == defaultGenesisTime {
		gt, err = ptypes.TimestampProto(time.Unix(0, 0))
	} else {
		gt, err = ptypes.TimestampProto(genesisTime)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert genesis time to proto: %v", err)
	}

	genValRoot := ns.GenesisFetcher.GenesisValidatorRoot()
	return &ethpb.GenesisResponse{
		GenesisTime:           uint64(gt.GetSeconds()),
		GenesisForkVersion:    params.BeaconConfig().GenesisForkVersion,
		GenesisValidatorsRoot: genValRoot[:],
	}, nil
}
