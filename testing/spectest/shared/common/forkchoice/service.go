package forkchoice

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache/depositcache"
	coreTime "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	payloadattribute "github.com/prysmaticlabs/prysm/v3/consensus-types/payload-attribute"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func startChainService(t testing.TB,
	st state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	engineMock *engineMock,
) *blockchain.Service {
	db := testDB.SetupDB(t)
	ctx := context.Background()
	require.NoError(t, db.SaveBlock(ctx, block))
	r, err := block.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, r))

	cp := &ethpb.Checkpoint{
		Epoch: coreTime.CurrentEpoch(st),
		Root:  r[:],
	}
	require.NoError(t, db.SaveState(ctx, st, r))
	require.NoError(t, db.SaveJustifiedCheckpoint(ctx, cp))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))
	attPool, err := attestations.NewService(ctx, &attestations.Config{
		Pool: attestations.NewPool(),
	})
	require.NoError(t, err)

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	fc := doublylinkedtree.New()
	opts := append([]blockchain.Option{},
		blockchain.WithExecutionEngineCaller(engineMock),
		blockchain.WithFinalizedStateAtStartUp(st),
		blockchain.WithDatabase(db),
		blockchain.WithAttestationService(attPool),
		blockchain.WithForkChoiceStore(fc),
		blockchain.WithStateGen(stategen.New(db, fc)),
		blockchain.WithStateNotifier(&mock.MockStateNotifier{}),
		blockchain.WithAttestationPool(attestations.NewPool()),
		blockchain.WithDepositCache(depositCache),
		blockchain.WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	)
	service, err := blockchain.NewService(context.Background(), opts...)
	require.NoError(t, err)
	require.NoError(t, service.StartFromSavedState(st))
	return service
}

type engineMock struct {
	powBlocks       map[[32]byte]*ethpb.PowBlock
	latestValidHash []byte
	payloadStatus   error
}

func (m *engineMock) GetPayload(context.Context, [8]byte, primitives.Slot) (interfaces.ExecutionData, error) {
	return nil, nil
}

func (m *engineMock) ForkchoiceUpdated(context.Context, *pb.ForkchoiceState, payloadattribute.Attributer) (*pb.PayloadIDBytes, []byte, error) {
	return nil, m.latestValidHash, m.payloadStatus
}
func (m *engineMock) NewPayload(context.Context, interfaces.ExecutionData) ([]byte, error) {
	return m.latestValidHash, m.payloadStatus
}

func (m *engineMock) LatestExecutionBlock() (*pb.ExecutionBlock, error) {
	return nil, nil
}

func (m *engineMock) ExchangeTransitionConfiguration(context.Context, *pb.TransitionConfiguration) error {
	return nil
}

func (m *engineMock) ExecutionBlockByHash(_ context.Context, hash common.Hash, _ bool) (*pb.ExecutionBlock, error) {
	b, ok := m.powBlocks[bytesutil.ToBytes32(hash.Bytes())]
	if !ok {
		return nil, nil
	}

	td := bytesutil.LittleEndianBytesToBigInt(b.TotalDifficulty)
	tdHex := hexutil.EncodeBig(td)
	return &pb.ExecutionBlock{
		Header: gethtypes.Header{
			ParentHash: common.BytesToHash(b.ParentHash),
		},
		TotalDifficulty: tdHex,
		Hash:            common.BytesToHash(b.BlockHash),
	}, nil
}

func (m *engineMock) GetTerminalBlockHash(context.Context, uint64) ([]byte, bool, error) {
	return nil, false, nil
}
