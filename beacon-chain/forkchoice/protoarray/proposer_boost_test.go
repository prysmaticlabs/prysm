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
	ctx := context.Background()
	zeroHash := params.BeaconConfig().ZeroHash
	graffiti := [32]byte{}
	balances := make([]uint64, 64) // 64 active validators.
	for i := 0; i < len(balances); i++ {
		balances[i] = 10
	}
	jEpoch, fEpoch := types.Epoch(0), types.Epoch(0)
	t.Run("back-propagates boost score to ancestors after proposer boosting", func(t *testing.T) {
		f := setup(jEpoch, fEpoch)

		// The head should always start at the finalized block.
		headRoot, err := f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, zeroHash, headRoot, "Incorrect head with genesis")

		// Insert block at slot 1 into the tree and verify head is at that block:
		//         0
		//         |
		//         1 <- HEAD
		slot := types.Slot(1)
		newRoot := indexToHash(1)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				slot,
				newRoot,
				headRoot,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)
		f.ProcessAttestation(ctx, []uint64{0}, newRoot, fEpoch)
		headRoot, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, newRoot, headRoot, "Incorrect head for justified epoch at slot 1")

		// Insert block at slot 2 into the tree and verify head is at that block:
		//         0
		//         |
		//         1
		//         |
		//         2 <- HEAD
		slot = types.Slot(2)
		newRoot = indexToHash(2)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				slot,
				newRoot,
				headRoot,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)
		f.ProcessAttestation(ctx, []uint64{1}, newRoot, fEpoch)
		headRoot, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, newRoot, headRoot, "Incorrect head for justified epoch at slot 2")

		// Insert block at slot 3 into the tree and verify head is at that block:
		//         0
		//         |
		//         1
		//         |
		//         2
		//         |
		//         3 <- HEAD
		slot = types.Slot(2)
		newRoot = indexToHash(2)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				slot,
				newRoot,
				headRoot,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)
		f.ProcessAttestation(ctx, []uint64{2}, newRoot, fEpoch)
		headRoot, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, newRoot, headRoot, "Incorrect head for justified epoch at slot 3")

		// Insert a second block at slot 3 into the tree and boost its score.
		//         0
		//         |
		//         1
		//         |
		//         2
		//        / \
		//       3   4 <- HEAD
		slot = types.Slot(3)
		newRoot = indexToHash(4)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				slot,
				newRoot,
				headRoot,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)
		f.ProcessAttestation(ctx, []uint64{3}, newRoot, fEpoch)
		threeSlots := 3 * params.BeaconConfig().SecondsPerSlot
		genesisTime := time.Now().Add(-time.Second * time.Duration(threeSlots))
		require.NoError(t, f.BoostProposerRoot(ctx, slot, newRoot, genesisTime))
		headRoot, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, newRoot, headRoot, "Incorrect head for justified epoch at slot 3")

		// Check the ancestor scores from the store.
		require.Equal(t, 4, len(f.store.nodes))
		require.Equal(t, f.store.nodes[0].weight, uint64(0)) // Genesis has 0 weight here.

		// Expect nodes to have a boosted, back-propagated score.
		require.Equal(t, f.store.nodes[1].weight, uint64(47))
		require.Equal(t, f.store.nodes[2].weight, uint64(37))
		require.Equal(t, f.store.nodes[3].weight, uint64(17))
	})
	t.Run("vanilla ex ante attack", func(t *testing.T) {
		f := setup(jEpoch, fEpoch)

		// The head should always start at the finalized block.
		r, err := f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, zeroHash, r, "Incorrect head with genesis")

		// Proposer from slot 1 does not reveal their block, B, at slot 1.
		// Proposer at slot 2 does reveal their block, C, and it becomes the head.
		// C builds on A, as proposer at slot 1 did not reveal B.
		//         A       -> Slot 0
		//        / \
		//	    (B?) \     -> Slot 1, B supposed to happen but does not occur.
		//            \
		//             C   -> Slot 2 HEAD
		honestBlockSlot := types.Slot(2)
		honestBlock := indexToHash(2)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				honestBlockSlot,
				honestBlock,
				zeroHash,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")

		maliciouslyWithheldBlockSlot := types.Slot(1)
		maliciouslyWithheldBlock := indexToHash(1)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				maliciouslyWithheldBlockSlot,
				maliciouslyWithheldBlock,
				zeroHash,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)

		// Ensure the head is C, the honest block.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")

		// We boost the honest proposal at slot 2.
		secondsPerSlot := time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot)
		genesis := time.Now().Add(-2 * secondsPerSlot)
		require.NoError(t, f.BoostProposerRoot(ctx, honestBlockSlot /* slot */, honestBlock, genesis))

		// The maliciously withheld block has one vote.
		votes := []uint64{1}
		f.ProcessAttestation(ctx, votes, maliciouslyWithheldBlock, fEpoch)

		// Ensure the head is STILL C, the honest block, as the honest block had proposer boost.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")
	})
	t.Run("adversarial attestations > proposer boosting", func(t *testing.T) {
		f := setup(jEpoch, fEpoch)

		// The head should always start at the finalized block.
		r, err := f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, zeroHash, r, "Incorrect head with genesis")

		// Proposer from slot 1 does not reveal their block, B, at slot 1.
		// Proposer at slot 2 does reveal their block, C, and it becomes the head.
		// C builds on A, as proposer at slot 1 did not reveal B.
		//         A       -> Slot 0
		//        / \
		//	    (B?) \     -> Slot 1, B supposed to happen but does not occur.
		//            \
		//             C   -> Slot 2 HEAD
		honestBlockSlot := types.Slot(2)
		honestBlock := indexToHash(2)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				honestBlockSlot,
				honestBlock,
				zeroHash,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)

		// Ensure C is the head.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")

		maliciouslyWithheldBlockSlot := types.Slot(1)
		maliciouslyWithheldBlock := indexToHash(1)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				maliciouslyWithheldBlockSlot,
				maliciouslyWithheldBlock,
				zeroHash,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)

		// Ensure C is still the head after the malicious proposer reveals their block.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")

		// We boost the honest proposal at slot 2.
		secondsPerSlot := time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot)
		genesis := time.Now().Add(-2 * secondsPerSlot)
		require.NoError(t, f.BoostProposerRoot(ctx, honestBlockSlot /* slot */, honestBlock, genesis))

		// An attestation is received for B that has more voting power than C with the proposer boost,
		// allowing B to then become the head if their attestation has enough adversarial votes.
		votes := []uint64{1, 2}
		f.ProcessAttestation(ctx, votes, maliciouslyWithheldBlock, fEpoch)

		// Expect the head to have switched to B.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, maliciouslyWithheldBlock, r, "Expected B to become the head")
	})
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
