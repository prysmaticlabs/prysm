package core

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/builder"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache/depositcache"
	blockfeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/block"
	opfeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/execution"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/blstoexec"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/sync"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type Proposer interface {
	ProposeBeaconBlock(ctx context.Context, req *ethpb.GenericSignedBeaconBlock) ([]byte, *RpcError)
}

type Service struct {
	HeadFetcher            blockchain.HeadFetcher
	TimeFetcher            blockchain.TimeFetcher
	SyncChecker            sync.Checker
	Broadcaster            p2p.Broadcaster
	SyncCommitteePool      synccommittee.Pool
	OperationNotifier      opfeed.Notifier
	AttestationCache       *cache.AttestationCache
	StateGen               stategen.StateManager
	P2P                    p2p.Broadcaster
	ForkchoiceFetcher      blockchain.ForkchoiceFetcher
	OptimisticModeFetcher  blockchain.OptimisticModeFetcher
	Eth1BlockFetcher       execution.POWBlockFetcher
	Eth1InfoFetcher        execution.ChainInfoFetcher
	DepositFetcher         cache.DepositFetcher
	ChainStartFetcher      execution.ChainStartFetcher
	PendingDepositsFetcher depositcache.PendingDepositsFetcher
	SlashingsPool          slashings.PoolManager
	ExitPool               voluntaryexits.PoolManager
	AttPool                attestations.Pool
	BLSChangesPool         blstoexec.PoolManager
	ProposerSlotIndexCache *cache.ProposerPayloadIDsCache
	BeaconDB               db.HeadAccessDatabase
	FinalizationFetcher    blockchain.FinalizationFetcher
	ExecutionEngineCaller  execution.EngineCaller
	BlockBuilder           builder.BlockBuilder
	BlockReceiver          blockchain.BlockReceiver
	BlockNotifier          blockfeed.Notifier
	ForkFetcher            blockchain.ForkFetcher
	BlockFetcher           execution.POWBlockFetcher
	MockEth1Votes          bool
}
