package protoarray

import (
	"context"
	"fmt"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

// Simple, ex-ante attack reorg.
// In a nutshell, an adversarial block proposer in slot n keeps its proposal hidden.
// The honest block proposer in slot n+1 will then propose a competing block. The
// adversary can now use its committee members’ votes from both slots n and n+ 1
// to vote for the withheld block of slot n in an attempt to outnumber honest votes
// on the proposal of slot n + 1. As a result, blocks proposed by honest validators
// may end up orphaned, i.e., they are displaced out of the chain chosen by LMD
// GHOST. In [19] this reorg strategy is part of a bigger scheme to delay consensus
func TestForkChoice_BoostProposerRoot_PreventsExAnteAttack(t *testing.T) {
	// TODO: Add more robust setup with actual threshold of votes at which proposer boost wins vs.
	// not having a threshold of votes.
	// test case matrix: proposed blocks A and B
	// 1. A boosted, B no, same votes => A wins
	// 2. A boosted, B no, but B a lot of votes => B wins depending on vote threshold
	// 3. A boosted, B boosted => comes down to votes
	// 4. Neither boosted => comes down to votes

	balances := make([]uint64, 32)
	for i := 0; i < len(balances); i++ {
		balances[i] = 10
	}
	jEpoch, fEpoch := types.Epoch(0), types.Epoch(0)
	f := setup(jEpoch, fEpoch)

	// The head should always start at the finalized block.
	r, err := f.Head(context.Background(), jEpoch, params.BeaconConfig().ZeroHash, balances, fEpoch)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().ZeroHash, r, "Incorrect head with genesis")

	// The proposer at slot 1 does not reveal their block.

	// Insert block at slot 2 into the tree and verify head is at that block:
	//         0
	//        /
	//	(1? no block?)
	//      /
	//     2 <- HEAD
	proposalSlot := types.Slot(2)
	require.NoError(t,
		f.ProcessBlock(
			context.Background(),
			proposalSlot,
			indexToHash(2),
			params.BeaconConfig().ZeroHash,
			[32]byte{},
			jEpoch,
			fEpoch,
		),
	)

	r, err = f.Head(context.Background(), jEpoch, params.BeaconConfig().ZeroHash, balances, fEpoch)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(2), r, "Incorrect head for justified epoch at slot 2")

	// Attacker comes out with block from slot 1, very late (after slot 2 has started).
	// with the recently proposer block. The attacker has all the withheld votes from slot n in the
	// proposer
	//         0
	//        /
	//	(1? no block?)
	//      /     \
	//     2       1 (block from slot 1 released late)

	//// An attacker prepared block n, but does not broadcast it until time n+1, to compete
	////            0
	////           / \
	////  head -> 2  1
	//require.NoError(t, f.ProcessBlock(context.Background(), 0, indexToHash(1), params.BeaconConfig().ZeroHash, [32]byte{}, 1, 1))
	//
	//r, err = f.Head(context.Background(), 1, params.BeaconConfig().ZeroHash, balances, 1)
	//require.NoError(t, err)
	//assert.Equal(t, indexToHash(2), r, "Incorrect head for with justified epoch at 1")
	//
	//// Add a vote to block 1 of the tree, but BOOST block 2 and ensure the head is still at 2.
	////            0
	////           / \
	////  head -> 2  1 (1 did not win, EVEN with a vote, as 2 got a proposer boost)
	//err = f.BoostProposerRoot(context.Background(), 0, indexToHash(1), time.Now())
	//require.NoError(t, err)
	//f.ProcessAttestation(context.Background(), []uint64{0}, indexToHash(1), 2)
	//r, err = f.Head(context.Background(), 1, params.BeaconConfig().ZeroHash, balances, 1)
	//require.NoError(t, err)
	//assert.Equal(t, indexToHash(1), r, "Incorrect head for with justified epoch at 1")
}

func TestForkChoice_BoostProposerRoot_CanFindHead(t *testing.T) {
	balances := make([]uint64, 32)
	for i := 0; i < len(balances); i++ {
		balances[i] = 10
	}
	f := setup(1, 1)

	// The head should always start at the finalized block.
	r, err := f.Head(context.Background(), 1, params.BeaconConfig().ZeroHash, balances, 1)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().ZeroHash, r, "Incorrect head with genesis")

	// Insert block 2 into the tree and verify head is at 2:
	//         0
	//        /
	//       2 <- head
	require.NoError(t, f.ProcessBlock(context.Background(), 0, indexToHash(2), params.BeaconConfig().ZeroHash, [32]byte{}, 1, 1))

	r, err = f.Head(context.Background(), 1, params.BeaconConfig().ZeroHash, balances, 1)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(2), r, "Incorrect head for justified epoch at 1")

	// Insert block 1 into the tree and verify head is still at 2:
	//            0
	//           / \
	//  head -> 2  1
	require.NoError(t, f.ProcessBlock(context.Background(), 0, indexToHash(1), params.BeaconConfig().ZeroHash, [32]byte{}, 1, 1))

	r, err = f.Head(context.Background(), 1, params.BeaconConfig().ZeroHash, balances, 1)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(2), r, "Incorrect head for with justified epoch at 1")

	// Add a vote to block 1 of the tree, but BOOST block 2 and ensure the head is still at 2.
	//            0
	//           / \
	//  head -> 2  1 (1 did not win, EVEN with a vote, as 2 got a proposer boost)
	err = f.BoostProposerRoot(context.Background(), 0, indexToHash(1), time.Now())
	require.NoError(t, err)
	f.ProcessAttestation(context.Background(), []uint64{0}, indexToHash(1), 2)
	r, err = f.Head(context.Background(), 1, params.BeaconConfig().ZeroHash, balances, 1)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(1), r, "Incorrect head for with justified epoch at 1")
}

func TestForkChoice_BoostProposerRoot(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.SecondsPerSlot = 6
	cfg.IntervalsPerSlot = 3
	params.OverrideBeaconConfig(cfg)
	ctx := context.Background()

	t.Run("does not boost block from different slot", func(t *testing.T) {
		f := &ForkChoice{
			store: &Store{},
		}
		// Genesis set to 1 slot ago.
		genesis := time.Now().Add(-time.Duration(cfg.SecondsPerSlot) * time.Second)
		blockRoot := [32]byte{'A'}

		// Trying to boost a block from slot 0 should not work.
		err := f.BoostProposerRoot(ctx, 0 /* slot */, blockRoot, genesis)
		require.NoError(t, err)
		require.DeepEqual(t, [32]byte{}, f.store.proposerBoostRoot)
	})
	t.Run("does not boost untimely block from same slot", func(t *testing.T) {
		f := &ForkChoice{
			store: &Store{},
		}
		// Genesis set to 1 slot ago + X where X > attesting interval.
		genesis := time.Now().Add(-time.Duration(cfg.SecondsPerSlot) * time.Second)
		attestingInterval := time.Duration(cfg.SecondsPerSlot / cfg.IntervalsPerSlot)
		greaterThanAttestingInterval := attestingInterval + 100*time.Millisecond
		genesis = genesis.Add(-greaterThanAttestingInterval * time.Second)
		blockRoot := [32]byte{'A'}

		// Trying to boost a block from slot 1 that is untimely should not work.
		err := f.BoostProposerRoot(ctx, 1 /* slot */, blockRoot, genesis)
		require.NoError(t, err)
		require.DeepEqual(t, [32]byte{}, f.store.proposerBoostRoot)
	})
	t.Run("boosts perfectly timely block from same slot", func(t *testing.T) {
		f := &ForkChoice{
			store: &Store{},
		}
		// Genesis set to 1 slot ago + 0 seconds into the attesting interval.
		genesis := time.Now().Add(-time.Duration(cfg.SecondsPerSlot) * time.Second)
		fmt.Println(genesis)
		blockRoot := [32]byte{'A'}

		err := f.BoostProposerRoot(ctx, 1 /* slot */, blockRoot, genesis)
		require.NoError(t, err)
		require.DeepEqual(t, [32]byte{'A'}, f.store.proposerBoostRoot)
	})
	t.Run("boosts timely block from same slot", func(t *testing.T) {
		f := &ForkChoice{
			store: &Store{},
		}
		// Genesis set to 1 slot ago + (attesting interval / 2).
		genesis := time.Now().Add(-time.Duration(cfg.SecondsPerSlot) * time.Second)
		blockRoot := [32]byte{'A'}
		halfAttestingInterval := time.Second
		genesis = genesis.Add(-halfAttestingInterval)

		err := f.BoostProposerRoot(ctx, 1 /* slot */, blockRoot, genesis)
		require.NoError(t, err)
		require.DeepEqual(t, [32]byte{'A'}, f.store.proposerBoostRoot)
	})
}

func TestForkChoice_computeProposerBoostScore(t *testing.T) {
	t.Run("nil justified balances throws error", func(t *testing.T) {
		_, err := computeProposerBoostScore(nil)
		require.ErrorContains(t, "no active validators", err)
	})
	t.Run("normal active balances computes score", func(t *testing.T) {
		validatorBalances := make([]uint64, 32)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = 10
		}
		// Avg balance is 10, and the number of validators is 32.
		// With a committee size of num validators (32) / slots per epoch (32) == 1,
		// we then have a committee weight of avg balance * committee size = 10 * 1 = 10.
		// The score then becomes 10 * PROPOSER_SCORE_BOOST // 100, which is
		// 10 * 70 / 100 = 7.
		score, err := computeProposerBoostScore(validatorBalances)
		require.NoError(t, err)
		require.Equal(t, uint64(7), score)
	})
}
