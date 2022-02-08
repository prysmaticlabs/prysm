package forkchoice

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	coreTime "github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func startChainService(t *testing.T, st state.BeaconState, block block.SignedBeaconBlock, engineMock *ExecutionEngineMock) *blockchain.Service {
	db := testDB.SetupDB(t)
	ctx := context.Background()
	require.NoError(t, db.SaveBlock(ctx, block))
	r, err := block.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, r))
	require.NoError(t, db.SaveState(ctx, st, r))
	cp := &ethpb.Checkpoint{
		Epoch: coreTime.CurrentEpoch(st),
		Root:  r[:],
	}
	require.NoError(t, db.SaveJustifiedCheckpoint(ctx, cp))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))
	attPool, err := attestations.NewService(ctx, &attestations.Config{
		Pool: attestations.NewPool(),
	})
	require.NoError(t, err)

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	opts := append([]blockchain.Option{},
		blockchain.WithExecutionEngineCaller(engineMock),
		blockchain.WithFinalizedStateAtStartUp(st),
		blockchain.WithDatabase(db),
		blockchain.WithAttestationService(attPool),
		blockchain.WithForkChoiceStore(protoarray.New(0, 0, params.BeaconConfig().ZeroHash)),
		blockchain.WithStateGen(stategen.New(db)),
		blockchain.WithStateNotifier(&mock.MockStateNotifier{}),
		blockchain.WithAttestationPool(attestations.NewPool()),
		blockchain.WithDepositCache(depositCache),
	)
	service, err := blockchain.NewService(context.Background(), opts...)
	require.NoError(t, err)
	service.Start()
	return service
}
