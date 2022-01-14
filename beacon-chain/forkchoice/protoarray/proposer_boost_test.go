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
	balances := make([]uint64, 32)
	for i := 0; i < len(balances); i++ {
		balances[i] = 10
	}
	jEpoch, fEpoch := types.Epoch(0), types.Epoch(0)
	t.Run("attacker succeeds without proposer score boosting", func(t *testing.T) {
		f := setup(jEpoch, fEpoch)

		// The head should always start at the finalized block.
		r, err := f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, zeroHash, r, "Incorrect head with genesis")

		// The proposer at slot 1 does not reveal their block.

		// Insert block at slot 2 into the tree and verify head is at that block:
		//         0
		//        /
		//	(1? no block?)
		//      /
		//     2 <- HEAD
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
		// One validator from the assigned committee for slot 2 has voted for the honest block so far.
		votes := []uint64{0}
		f.ProcessAttestation(ctx, votes, honestBlock, fEpoch)

		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")

		// Attacker comes out with block from slot 1 late, at the same time as the honest, slot 2 proposal.
		//         0
		//        /
		//	(1? no block?)
		//      /     \
		//     2       1 (block from slot 1 released late)
		maliciousBlockSlot := types.Slot(1)
		maliciouslyWithheldBlock := indexToHash(1)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				maliciousBlockSlot,
				maliciouslyWithheldBlock,
				zeroHash,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)
		// The attacker takes advantage of getting some of the honest, voting validators
		// from slot 2 to vote on his malicious block, as it was published at the same time as
		// honest proposal.
		votesFromSlot2 := []uint64{1}
		f.ProcessAttestation(ctx, votesFromSlot2, maliciouslyWithheldBlock, fEpoch)
		// The attacker also had a vote it withheld from slot 1.
		// The attacker has the more votes than the honest block now.
		votesFromSlot1 := []uint64{2}
		f.ProcessAttestation(ctx, votesFromSlot1, maliciouslyWithheldBlock, fEpoch)

		// The head should change because the attacker published their block right after the honest proposer
		// published their own. We should see the head change to the malicious block.
		//                  0
		//                 /
		//	         (1? no block?)
		//               /     \
		//              2       1 <- HEAD, attacker wins
		r, err = f.Head(
			ctx,
			jEpoch,
			zeroHash,
			balances,
			fEpoch,
		)
		require.NoError(t, err)
		assert.Equal(t, maliciouslyWithheldBlock, r, "Incorrect head for with justified epoch")
	})
	t.Run("attacker fails when honest, timely proposals are boosted", func(t *testing.T) {
		f := setup(jEpoch, fEpoch)

		// The head should always start at the finalized block.
		r, err := f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, zeroHash, r, "Incorrect head with genesis")

		// The proposer at slot 1 does not reveal their block.

		// Insert block at slot 2 into the tree and verify head is at that block:
		//         0
		//        /
		//	(1? no block?)
		//      /
		//     2 <- HEAD
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
		// One validator from the assigned committee for slot 2 has voted for the honest block so far.
		votes := []uint64{0}
		f.ProcessAttestation(ctx, votes, honestBlock, fEpoch)

		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")

		// Attacker comes out with block from slot 1, very late (after slot 2 has started).
		// with the recently proposer block. The attacker has all the withheld votes from slot 1 in their block.
		//         0
		//        /
		//	(1? no block?)
		//      /     \
		//     2       1 (block from slot 1 released late)
		maliciousBlockSlot := types.Slot(1)
		maliciouslyWithheldBlock := indexToHash(1)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				maliciousBlockSlot,
				maliciouslyWithheldBlock,
				zeroHash,
				graffiti,
				jEpoch,
				fEpoch,
			),
		)
		// The attacker takes advantage of getting some of the honest, voting validators
		// from slot 2 to vote on his malicious block, as it was published at the same time as
		// honest proposal.
		votesFromSlot2 := []uint64{1}
		f.ProcessAttestation(ctx, votesFromSlot2, maliciouslyWithheldBlock, fEpoch)
		// The attacker also had a vote it withheld from slot 1.
		// The attacker has the more votes than the honest block now.
		votesFromSlot1 := []uint64{2}
		f.ProcessAttestation(ctx, votesFromSlot1, maliciouslyWithheldBlock, fEpoch)

		// The honest block at slot 2, assuming it received in a timely manner, will get a boost
		// during the chain head calculations by fork choice. We give it that boost.
		// We set the genesis time as current_time - 2 slots worth of seconds to place us in slot 2.
		secondsPerSlot := time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
		genesisTime := time.Now().Add(-2 * secondsPerSlot)
		err = f.BoostProposerRoot(ctx, honestBlockSlot, honestBlock, genesisTime)
		require.NoError(t, err)

		// Proposer boost helps the honestly-proposed block win (received in a timely manner) even in
		// this attack scenario from a maliciously-withheld block. We should see the head should not change.
		//                  0
		//                 /
		//	         (1? no block?)
		//               /     \
		// remains HEAD -> 2    1 (1 should not win, EVEN with more votes than 2, as 2 got a proposer boost)
		r, err = f.Head(
			ctx,
			jEpoch,
			zeroHash,
			balances,
			fEpoch,
		)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for with justified epoch")
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
