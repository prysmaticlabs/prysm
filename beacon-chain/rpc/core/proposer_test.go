package core

import (
	"context"
	"math/big"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	coretime "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	dbutil "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v4/beacon-chain/execution/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/container/trie"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/attestation"
	attaggregation "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/attestation/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"google.golang.org/protobuf/proto"
)

func TestProposer_ComputeStateRoot_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, parentRoot, privKeys := util.DeterministicGenesisStateWithGenesisBlock(t, ctx, db, 100)

	s := &Service{
		ChainStartFetcher: &mockExecution.Chain{},
		Eth1InfoFetcher:   &mockExecution.Chain{},
		Eth1BlockFetcher:  &mockExecution.Chain{},
		StateGen:          stategen.New(db, doublylinkedtree.New()),
	}
	req := util.NewBeaconBlock()
	req.Block.ProposerIndex = 84
	req.Block.ParentRoot = parentRoot[:]
	req.Block.Slot = 1
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(ctx, beaconState)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()-1))
	req.Block.Body.RandaoReveal = randaoReveal
	currentEpoch := coretime.CurrentEpoch(beaconState)
	req.Signature, err = signing.ComputeDomainAndSign(beaconState, currentEpoch, req.Block, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerIdx])
	require.NoError(t, err)

	wsb, err := blocks.NewSignedBeaconBlock(req)
	require.NoError(t, err)
	_, err = s.computeStateRoot(context.Background(), wsb)
	require.NoError(t, err)
}

func TestProposer_PendingDeposits_Eth1DataVoteOK(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	newHeight := big.NewInt(height.Int64() + 11000)
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()):    []byte("0x0"),
			int(newHeight.Int64()): []byte("0x1"),
		},
	}

	var votes []*ethpb.Eth1Data

	blockHash := make([]byte, 32)
	copy(blockHash, "0x1")
	vote := &ethpb.Eth1Data{
		DepositRoot:  make([]byte, 32),
		BlockHash:    blockHash,
		DepositCount: 3,
	}
	period := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod)))
	for i := 0; i <= int(period/2); i++ {
		votes = append(votes, vote)
	}

	blockHash = make([]byte, 32)
	copy(blockHash, "0x0")
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetEth1DepositIndex(2))
	require.NoError(t, beaconState.SetEth1Data(&ethpb.Eth1Data{
		DepositRoot:  make([]byte, 32),
		BlockHash:    blockHash,
		DepositCount: 2,
	}))
	require.NoError(t, beaconState.SetEth1DataVotes(votes))

	blk := util.NewBeaconBlock()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	s := &Service{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockReceiver:     &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:       &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	_, eth1Height, err := s.canonicalEth1Data(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)

	assert.Equal(t, 0, eth1Height.Cmp(height))

	newState, err := b.ProcessEth1DataInBlock(ctx, beaconState, blk.Block.Body.Eth1Data)
	require.NoError(t, err)

	if proto.Equal(newState.Eth1Data(), vote) {
		t.Errorf("eth1data in the state equal to vote, when not expected to"+
			"have majority: Got %v", vote)
	}

	blk.Block.Body.Eth1Data = vote

	_, eth1Height, err = s.canonicalEth1Data(ctx, beaconState, vote)
	require.NoError(t, err)
	assert.Equal(t, 0, eth1Height.Cmp(newHeight))

	newState, err = b.ProcessEth1DataInBlock(ctx, beaconState, blk.Block.Body.Eth1Data)
	require.NoError(t, err)

	if !proto.Equal(newState.Eth1Data(), vote) {
		t.Errorf("eth1data in the state not of the expected kind: Got %v but wanted %v", newState.Eth1Data(), vote)
	}
}

func TestProposer_PendingDeposits_OutsideEth1FollowWindow(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:   bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot: make([]byte, 32),
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 2,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 8,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*ethpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 400,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("c"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 600,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("d"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, root))
	}
	for _, dp := range recentDeposits {
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, root)
	}

	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()

	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)

	s := &Service{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	deposits, err := s.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected list of deposits")

	// It should not return the recent deposits after their follow window.
	// as latest block number makes no difference in retrieval of deposits
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err = s.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_PendingDeposits_FollowsCorrectEth1Block(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	newHeight := big.NewInt(height.Int64() + 11000)
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()):    []byte("0x0"),
			int(newHeight.Int64()): []byte("0x1"),
		},
	}

	var votes []*ethpb.Eth1Data

	vote := &ethpb.Eth1Data{
		BlockHash:    bytesutil.PadTo([]byte("0x1"), 32),
		DepositRoot:  make([]byte, 32),
		DepositCount: 7,
	}
	period := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod)))
	for i := 0; i <= int(period/2); i++ {
		votes = append(votes, vote)
	}

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositRoot:  make([]byte, 32),
			DepositCount: 5,
		},
		Eth1DepositIndex: 1,
		Eth1DataVotes:    votes,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()

	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 8,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 14,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*ethpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 5000,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("c"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 6000,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("d"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, root))
	}
	for _, dp := range recentDeposits {
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, root)
	}

	s := &Service{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	deposits, err := s.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected list of deposits")

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	// we should get our pending deposits once this vote pushes the vote tally to include
	// the updated eth1 data.
	deposits, err = s.deposits(ctx, beaconState, vote)
	require.NoError(t, err)
	assert.Equal(t, len(recentDeposits), len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_PendingDeposits_CantReturnBelowStateEth1DepositIndex(t *testing.T) {
	ctx := context.Background()
	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetEth1Data(&ethpb.Eth1Data{
		BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
		DepositRoot:  make([]byte, 32),
		DepositCount: 100,
	}))
	require.NoError(t, beaconState.SetEth1DepositIndex(10))
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()
	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*ethpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*ethpb.DepositContainer
	for i := int64(2); i < 16; i++ {
		recentDeposits = append(recentDeposits, &ethpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{byte(i)}, 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, root))
	}
	for _, dp := range recentDeposits {
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, root)
	}

	s := &Service{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err := s.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)

	expectedDeposits := 6
	assert.Equal(t, expectedDeposits, len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_PendingDeposits_CantReturnMoreThanMax(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot:  make([]byte, 32),
			DepositCount: 100,
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()
	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*ethpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*ethpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &ethpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{byte(i)}, 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, height.Uint64(), dp.Index, root))
	}
	for _, dp := range recentDeposits {
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, height.Uint64(), dp.Index, root)
	}

	s := &Service{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err := s.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().MaxDeposits, uint64(len(deposits)), "Received unexpected number of pending deposits")
}

func TestProposer_PendingDeposits_CantReturnMoreThanDepositCount(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot:  make([]byte, 32),
			DepositCount: 5,
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()
	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*ethpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*ethpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &ethpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{byte(i)}, 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, root))
	}
	for _, dp := range recentDeposits {
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, root)
	}

	s := &Service{
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err := s.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 3, len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_DepositTrie_UtilizesCachedFinalizedDeposits(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot:  make([]byte, 32),
			DepositCount: 4,
		},
		Eth1DepositIndex: 1,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()

	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*ethpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("c"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("d"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(finalizedDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, root))
	}
	for _, dp := range recentDeposits {
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, root)
	}

	s := &Service{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	dt, err := s.depositTrie(ctx, &ethpb.Eth1Data{}, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))
	require.NoError(t, err)

	actualRoot, err := dt.HashTreeRoot()
	require.NoError(t, err)
	expectedRoot, err := depositTrie.HashTreeRoot()
	require.NoError(t, err)
	assert.Equal(t, expectedRoot, actualRoot, "Incorrect deposit trie root")
}

func TestProposer_DepositTrie_RebuildTrie(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot:  make([]byte, 32),
			DepositCount: 4,
		},
		Eth1DepositIndex: 1,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()

	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*ethpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("c"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("d"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(finalizedDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, root))
	}
	for _, dp := range recentDeposits {
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, root)
	}
	d := depositCache.AllDepositContainers(ctx)
	origDeposit, ok := proto.Clone(d[0].Deposit).(*ethpb.Deposit)
	assert.Equal(t, true, ok)
	junkCreds := mockCreds
	copy(junkCreds[:1], []byte{'A'})
	// Mutate it since its a pointer
	d[0].Deposit.Data.WithdrawalCredentials = junkCreds[:]
	// Insert junk to corrupt trie.
	err = depositCache.InsertFinalizedDeposits(ctx, 2, [32]byte{}, 0)
	require.NoError(t, err)

	// Add original back
	d[0].Deposit = origDeposit

	s := &Service{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	dt, err := s.depositTrie(ctx, &ethpb.Eth1Data{}, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))
	require.NoError(t, err)

	expectedRoot, err := depositTrie.HashTreeRoot()
	require.NoError(t, err)
	actualRoot, err := dt.HashTreeRoot()
	require.NoError(t, err)
	assert.Equal(t, expectedRoot, actualRoot, "Incorrect deposit trie root")

}

func TestProposer_ValidateDepositTrie(t *testing.T) {
	tt := []struct {
		name            string
		eth1dataCreator func() *ethpb.Eth1Data
		trieCreator     func() *trie.SparseMerkleTrie
		success         bool
	}{
		{
			name: "invalid trie items",
			eth1dataCreator: func() *ethpb.Eth1Data {
				return &ethpb.Eth1Data{DepositRoot: []byte{}, DepositCount: 10, BlockHash: []byte{}}
			},
			trieCreator: func() *trie.SparseMerkleTrie {
				newTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				return newTrie
			},
			success: false,
		},
		{
			name: "invalid deposit root",
			eth1dataCreator: func() *ethpb.Eth1Data {
				newTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				assert.NoError(t, newTrie.Insert([]byte{'a'}, 0))
				assert.NoError(t, newTrie.Insert([]byte{'b'}, 1))
				assert.NoError(t, newTrie.Insert([]byte{'c'}, 2))
				return &ethpb.Eth1Data{DepositRoot: []byte{'B'}, DepositCount: 3, BlockHash: []byte{}}
			},
			trieCreator: func() *trie.SparseMerkleTrie {
				newTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				assert.NoError(t, newTrie.Insert([]byte{'a'}, 0))
				assert.NoError(t, newTrie.Insert([]byte{'b'}, 1))
				assert.NoError(t, newTrie.Insert([]byte{'c'}, 2))
				return newTrie
			},
			success: false,
		},
		{
			name: "valid deposit trie",
			eth1dataCreator: func() *ethpb.Eth1Data {
				newTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				assert.NoError(t, newTrie.Insert([]byte{'a'}, 0))
				assert.NoError(t, newTrie.Insert([]byte{'b'}, 1))
				assert.NoError(t, newTrie.Insert([]byte{'c'}, 2))
				rt, err := newTrie.HashTreeRoot()
				require.NoError(t, err)
				return &ethpb.Eth1Data{DepositRoot: rt[:], DepositCount: 3, BlockHash: []byte{}}
			},
			trieCreator: func() *trie.SparseMerkleTrie {
				newTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				assert.NoError(t, newTrie.Insert([]byte{'a'}, 0))
				assert.NoError(t, newTrie.Insert([]byte{'b'}, 1))
				assert.NoError(t, newTrie.Insert([]byte{'c'}, 2))
				return newTrie
			},
			success: true,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			valid, err := validateDepositTrie(test.trieCreator(), test.eth1dataCreator())
			assert.Equal(t, test.success, valid)
			if valid {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProposer_Eth1Data_MajorityVote_SpansGenesis(t *testing.T) {
	ctx := context.Background()
	// Voting period will span genesis, causing the special case for pre-mined genesis to kick in.
	// In other words some part of the valid time range is before genesis, so querying the block cache would fail
	// without the special case added to allow this for testnets.
	slot := primitives.Slot(0)
	earliestValidTime, latestValidTime := majorityVoteBoundaryTime(slot)

	p := mockExecution.New().
		InsertBlock(50, earliestValidTime, []byte("earliest")).
		InsertBlock(100, latestValidTime, []byte("latest"))

	headBlockHash := []byte("headb")
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	s := &Service{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{BlockHash: headBlockHash, DepositCount: 0}},
	}

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Slot: slot,
		Eth1DataVotes: []*ethpb.Eth1Data{
			{BlockHash: []byte("earliest"), DepositCount: 1},
		},
	})
	require.NoError(t, err)
	majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
	require.NoError(t, err)
	assert.DeepEqual(t, headBlockHash, majorityVoteEth1Data.BlockHash)
}

func TestProposer_Eth1Data_MajorityVote(t *testing.T) {
	followDistanceSecs := params.BeaconConfig().Eth1FollowDistance * params.BeaconConfig().SecondsPerETH1Block
	followSlots := followDistanceSecs / params.BeaconConfig().SecondsPerSlot
	slot := primitives.Slot(64 + followSlots)
	earliestValidTime, latestValidTime := majorityVoteBoundaryTime(slot)

	dc := ethpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             bytesutil.PadTo([]byte("a"), 48),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err)
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	root, err := depositTrie.HashTreeRoot()
	require.NoError(t, err)
	assert.NoError(t, depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, root))

	t.Run("choose highest count", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("second"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count at earliest valid time - choose highest count", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("earliest"), DepositCount: 1},
				{BlockHash: []byte("earliest"), DepositCount: 1},
				{BlockHash: []byte("second"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("earliest")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count at latest valid time - choose highest count", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("latest"), DepositCount: 1},
				{BlockHash: []byte("latest"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("latest")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count before range - choose highest count within range", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().
			InsertBlock(49, earliestValidTime-1, []byte("before_range")).
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("before_range"), DepositCount: 1},
				{BlockHash: []byte("before_range"), DepositCount: 1},
				{BlockHash: []byte("first"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count after range - choose highest count within range", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(100, latestValidTime, []byte("latest")).
			InsertBlock(101, latestValidTime+1, []byte("after_range"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("after_range"), DepositCount: 1},
				{BlockHash: []byte("after_range"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count on unknown block - choose known block with highest count", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("unknown"), DepositCount: 1},
				{BlockHash: []byte("unknown"), DepositCount: 1},
				{BlockHash: []byte("first"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no blocks in range - choose current eth1data", func(t *testing.T) {
		p := mockExecution.New().
			InsertBlock(49, earliestValidTime-1, []byte("before_range")).
			InsertBlock(101, latestValidTime+1, []byte("after_range"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
		})
		require.NoError(t, err)

		currentEth1Data := &ethpb.Eth1Data{DepositCount: 1, BlockHash: []byte("current")}
		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: currentEth1Data},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("current")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no votes in range - choose most recent block", func(t *testing.T) {
		p := mockExecution.New().
			InsertBlock(49, earliestValidTime-1, []byte("before_range")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(101, latestValidTime+1, []byte("after_range"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("before_range"), DepositCount: 1},
				{BlockHash: []byte("after_range"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := make([]byte, 32)
		copy(expectedHash, "second")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no votes - choose more recent block", func(t *testing.T) {
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot:          slot,
			Eth1DataVotes: []*ethpb.Eth1Data{}})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := make([]byte, 32)
		copy(expectedHash, "latest")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no votes and more recent block has less deposits - choose current eth1data", func(t *testing.T) {
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
		})
		require.NoError(t, err)

		// Set the deposit count in current eth1data to exceed the latest most recent block's deposit count.
		currentEth1Data := &ethpb.Eth1Data{DepositCount: 2, BlockHash: []byte("current")}
		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: currentEth1Data},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("current")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("same count - choose more recent block", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("second"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("second")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count on block with less deposits - choose another block", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("no_new_deposits"), DepositCount: 0},
				{BlockHash: []byte("no_new_deposits"), DepositCount: 0},
				{BlockHash: []byte("second"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("second")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("only one block at earliest valid time - choose this block", func(t *testing.T) {
		t.Skip()
		p := mockExecution.New().InsertBlock(50, earliestValidTime, []byte("earliest"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("earliest"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("earliest")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("vote on last block before range - choose next block", func(t *testing.T) {
		p := mockExecution.New().
			InsertBlock(49, earliestValidTime-1, []byte("before_range")).
			// It is important to have height `50` with time `earliestValidTime+1` and not `earliestValidTime`
			// because of earliest block increment in the algorithm.
			InsertBlock(50, earliestValidTime+1, []byte("first"))

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("before_range"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := make([]byte, 32)
		copy(expectedHash, "first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no deposits - choose chain start eth1data", func(t *testing.T) {
		p := mockExecution.New().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(100, latestValidTime, []byte("latest"))
		p.Eth1Data = &ethpb.Eth1Data{
			BlockHash: []byte("eth1data"),
		}

		depositCache, err := depositcache.New()
		require.NoError(t, err)

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("earliest"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		s := &Service{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 0}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := s.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("eth1data")
		assert.DeepEqual(t, expectedHash, hash)
	})
}

func TestProposer_FilterAttestation(t *testing.T) {
	genesis := util.NewBeaconBlock()

	numValidators := uint64(64)
	st, privKeys := util.DeterministicGenesisState(t, numValidators)
	require.NoError(t, st.SetGenesisValidatorsRoot(params.BeaconConfig().ZeroHash[:]))
	assert.NoError(t, st.SetSlot(1))

	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		name         string
		wantedErr    string
		inputAtts    func() []*ethpb.Attestation
		expectedAtts func(inputAtts []*ethpb.Attestation) []*ethpb.Attestation
	}{
		{
			name: "nil attestations",
			inputAtts: func() []*ethpb.Attestation {
				return nil
			},
			expectedAtts: func(inputAtts []*ethpb.Attestation) []*ethpb.Attestation {
				return []*ethpb.Attestation{}
			},
		},
		{
			name: "invalid attestations",
			inputAtts: func() []*ethpb.Attestation {
				atts := make([]*ethpb.Attestation, 10)
				for i := 0; i < len(atts); i++ {
					atts[i] = util.HydrateAttestation(&ethpb.Attestation{
						Data: &ethpb.AttestationData{
							CommitteeIndex: primitives.CommitteeIndex(i),
						},
					})
				}
				return atts
			},
			expectedAtts: func(inputAtts []*ethpb.Attestation) []*ethpb.Attestation {
				return []*ethpb.Attestation{}
			},
		},
		{
			name: "filter aggregates ok",
			inputAtts: func() []*ethpb.Attestation {
				atts := make([]*ethpb.Attestation, 10)
				for i := 0; i < len(atts); i++ {
					atts[i] = util.HydrateAttestation(&ethpb.Attestation{
						Data: &ethpb.AttestationData{
							CommitteeIndex: primitives.CommitteeIndex(i),
							Source:         &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
						},
						AggregationBits: bitfield.Bitlist{0b00010010},
					})
					committee, err := helpers.BeaconCommitteeFromState(context.Background(), st, atts[i].Data.Slot, atts[i].Data.CommitteeIndex)
					assert.NoError(t, err)
					attestingIndices, err := attestation.AttestingIndices(atts[i].AggregationBits, committee)
					require.NoError(t, err)
					assert.NoError(t, err)
					domain, err := signing.Domain(st.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, params.BeaconConfig().ZeroHash[:])
					require.NoError(t, err)
					sigs := make([]bls.Signature, len(attestingIndices))
					var zeroSig [96]byte
					atts[i].Signature = zeroSig[:]

					for i, indice := range attestingIndices {
						hashTreeRoot, err := signing.ComputeSigningRoot(atts[i].Data, domain)
						require.NoError(t, err)
						sig := privKeys[indice].Sign(hashTreeRoot[:])
						sigs[i] = sig
					}
					atts[i].Signature = bls.AggregateSignatures(sigs).Marshal()
				}
				return atts
			},
			expectedAtts: func(inputAtts []*ethpb.Attestation) []*ethpb.Attestation {
				return []*ethpb.Attestation{inputAtts[0], inputAtts[1]}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				AttPool:     attestations.NewPool(),
				HeadFetcher: &mock.ChainService{State: st, Root: genesisRoot[:]},
			}
			atts := tt.inputAtts()
			received, err := s.validateAndDeleteAttsInPool(context.Background(), st, atts)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
				assert.Equal(t, nil, received)
			} else {
				assert.NoError(t, err)
				assert.DeepEqual(t, tt.expectedAtts(atts), received)
			}
		})
	}
}

func TestProposer_Deposits_ReturnsEmptyList_IfLatestEth1DataEqGenesisEth1Block(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockExecution.Chain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
		GenesisEth1Block: height,
	}

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:   bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot: make([]byte, 32),
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*ethpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*ethpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &ethpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{byte(i)}, 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, root))
	}
	for _, dp := range recentDeposits {
		root, err := depositTrie.HashTreeRoot()
		require.NoError(t, err)
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, root)
	}

	s := &Service{
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err := s.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_DeleteAttsInPool_Aggregated(t *testing.T) {
	s := &Service{
		AttPool: attestations.NewPool(),
	}
	priv, err := bls.RandKey()
	require.NoError(t, err)
	sig := priv.Sign([]byte("foo")).Marshal()
	aggregatedAtts := []*ethpb.Attestation{
		util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b10101}, Signature: sig}),
		util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b11010}, Signature: sig})}
	unaggregatedAtts := []*ethpb.Attestation{
		util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b10010}, Signature: sig}),
		util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b10100}, Signature: sig})}

	require.NoError(t, s.AttPool.SaveAggregatedAttestations(aggregatedAtts))
	require.NoError(t, s.AttPool.SaveUnaggregatedAttestations(unaggregatedAtts))

	aa, err := attaggregation.Aggregate(aggregatedAtts)
	require.NoError(t, err)
	require.NoError(t, s.deleteAttsInPool(context.Background(), append(aa, unaggregatedAtts...)))
	assert.Equal(t, 0, len(s.AttPool.AggregatedAttestations()), "Did not delete aggregated attestation")
	atts, err := s.AttPool.UnaggregatedAttestations()
	require.NoError(t, err)
	assert.Equal(t, 0, len(atts), "Did not delete unaggregated attestation")
}

func TestProposer_GetSyncAggregate_OK(t *testing.T) {
	s := &Service{
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		SyncCommitteePool: synccommittee.NewStore(),
	}

	r := params.BeaconConfig().ZeroHash
	conts := []*ethpb.SyncCommitteeContribution{
		{Slot: 1, SubcommitteeIndex: 0, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b0001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 0, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 0, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1110}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 1, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b0001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 1, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 1, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1110}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 2, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b0001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 2, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 2, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1110}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 3, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b0001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 3, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 3, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1110}, BlockRoot: r[:]},
		{Slot: 2, SubcommitteeIndex: 0, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b10101010}, BlockRoot: r[:]},
		{Slot: 2, SubcommitteeIndex: 1, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b10101010}, BlockRoot: r[:]},
		{Slot: 2, SubcommitteeIndex: 2, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b10101010}, BlockRoot: r[:]},
		{Slot: 2, SubcommitteeIndex: 3, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b10101010}, BlockRoot: r[:]},
	}

	for _, cont := range conts {
		require.NoError(t, s.SyncCommitteePool.SaveSyncCommitteeContribution(cont))
	}

	aggregate, err := s.getSyncAggregate(context.Background(), 1, bytesutil.ToBytes32(conts[0].BlockRoot))
	require.NoError(t, err)
	require.DeepEqual(t, bitfield.Bitvector32{0xf, 0xf, 0xf, 0xf}, aggregate.SyncCommitteeBits)

	aggregate, err = s.getSyncAggregate(context.Background(), 2, bytesutil.ToBytes32(conts[0].BlockRoot))
	require.NoError(t, err)
	require.DeepEqual(t, bitfield.Bitvector32{0xaa, 0xaa, 0xaa, 0xaa}, aggregate.SyncCommitteeBits)

	aggregate, err = s.getSyncAggregate(context.Background(), 3, bytesutil.ToBytes32(conts[0].BlockRoot))
	require.NoError(t, err)
	require.DeepEqual(t, bitfield.NewBitvector32(), aggregate.SyncCommitteeBits)
}

func majorityVoteBoundaryTime(slot primitives.Slot) (uint64, uint64) {
	s := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))
	slotStartTime := uint64(mockExecution.GenesisTime) + uint64((slot - (slot % (s))).Mul(params.BeaconConfig().SecondsPerSlot))
	earliestValidTime := slotStartTime - 2*params.BeaconConfig().SecondsPerETH1Block*params.BeaconConfig().Eth1FollowDistance
	latestValidTime := slotStartTime - params.BeaconConfig().SecondsPerETH1Block*params.BeaconConfig().Eth1FollowDistance

	return earliestValidTime, latestValidTime
}

func Test_extractBlobs(t *testing.T) {
	blobs := []*ethpb.SignedBlobSidecar{
		{Message: &ethpb.BlobSidecar{Index: 0}}, {Message: &ethpb.BlobSidecar{Index: 1}},
		{Message: &ethpb.BlobSidecar{Index: 2}}, {Message: &ethpb.BlobSidecar{Index: 3}},
		{Message: &ethpb.BlobSidecar{Index: 4}}, {Message: &ethpb.BlobSidecar{Index: 5}}}
	req := &ethpb.GenericSignedBeaconBlock{Block: &ethpb.GenericSignedBeaconBlock_Deneb{
		Deneb: &ethpb.SignedBeaconBlockAndBlobsDeneb{
			Blobs: blobs,
		},
	},
	}
	bs, err := extraSidecars(req)
	require.NoError(t, err)
	require.DeepEqual(t, blobs, bs)
}
