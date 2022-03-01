package nodetree

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

// Simple, ex-ante attack mitigation using proposer boost.
// In a nutshell, an adversarial block proposer in slot n+1 keeps its proposal hidden.
// The honest block proposer in slot n+2 will then propose an honest block. The
// adversary can now use its committee members’ votes from both slots n+1 and n+2.
// and release their withheld block of slot n+2 in an attempt to win fork choice.
// If the honest proposal is boosted at slot n+2, it will win against this attacker.
func TestForkChoice_BoostProposerRoot_PreventsExAnteAttack(t *testing.T) {
	ctx := context.Background()
	zeroHash := params.BeaconConfig().ZeroHash
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
				jEpoch,
				fEpoch,
				true,
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
				jEpoch,
				fEpoch,
				true,
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
		slot = types.Slot(3)
		newRoot = indexToHash(3)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				slot,
				newRoot,
				headRoot,
				jEpoch,
				fEpoch,
				true,
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
				jEpoch,
				fEpoch,
				true,
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
		require.Equal(t, 5, len(f.store.nodeByRoot))

		// Expect nodes to have a boosted, back-propagated score.
		// Ancestors have the added weights of their children. Genesis is a special exception at 0 weight,
		require.Equal(t, f.store.treeRootNode.weight, uint64(0))

		// Otherwise, assuming a block, A, that is not-genesis:
		//
		// A -> B -> C
		//
		//Where each one has a weight of 10 individually, the final weights will look like
		//
		// (A: 30) -> (B: 20) -> (C: 10)
		//
		// The boost adds 14 to the weight, so if C is boosted, we would have
		//
		// (A: 44) -> (B: 34) -> (C: 24)
		//
		// In this case, we have a small fork:
		//
		// (A: 54) -> (B: 44) -> (C: 34)
		//                   \_->(D: 24)
		//
		// So B has its own weight, 10, and the sum of both C and D. That's why we see weight 54 in the
		// middle instead of the normal progression of (44 -> 34 -> 24).
		node1 := f.store.nodeByRoot[indexToHash(1)]
		require.Equal(t, node1.weight, uint64(54))
		node2 := f.store.nodeByRoot[indexToHash(2)]
		require.Equal(t, node2.weight, uint64(44))
		node3 := f.store.nodeByRoot[indexToHash(4)]
		require.Equal(t, node3.weight, uint64(24))
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
		//         A
		//        / \
		//      (B?) \
		//            \
		//             C <- Slot 2 HEAD
		honestBlockSlot := types.Slot(2)
		honestBlock := indexToHash(2)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				honestBlockSlot,
				honestBlock,
				zeroHash,
				jEpoch,
				fEpoch,
				true,
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
				jEpoch,
				fEpoch,
				true,
			),
		)

		// Ensure the head is C, the honest block.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")

		// We boost the honest proposal at slot 2.
		secondsPerSlot := time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot)
		genesis := time.Now().Add(-2 * secondsPerSlot)
		require.NoError(t, f.BoostProposerRoot(ctx, honestBlockSlot, honestBlock, genesis))

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
		//         A
		//        / \
		//	    (B?) \
		//            \
		//             C <- Slot 2 HEAD
		honestBlockSlot := types.Slot(2)
		honestBlock := indexToHash(2)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				honestBlockSlot,
				honestBlock,
				zeroHash,
				jEpoch,
				fEpoch,
				true,
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
				jEpoch,
				fEpoch,
				true,
			),
		)

		// Ensure C is still the head after the malicious proposer reveals their block.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, honestBlock, r, "Incorrect head for justified epoch at slot 2")

		// We boost the honest proposal at slot 2.
		secondsPerSlot := time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot)
		genesis := time.Now().Add(-2 * secondsPerSlot)
		require.NoError(t, f.BoostProposerRoot(ctx, honestBlockSlot, honestBlock, genesis))

		// An attestation is received for B that has more voting power than C with the proposer boost,
		// allowing B to then become the head if their attestation has enough adversarial votes.
		votes := []uint64{1, 2}
		f.ProcessAttestation(ctx, votes, maliciouslyWithheldBlock, fEpoch)

		// Expect the head to have switched to B.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, maliciouslyWithheldBlock, r, "Expected B to become the head")
	})
	t.Run("boosting necessary to sandwich attack", func(t *testing.T) {
		// Boosting necessary to sandwich attack.
		// Objects:
		//	Block A - slot N
		//	Block B (parent A) - slot N+1
		//	Block C (parent A) - slot N+2
		//	Block D (parent B) - slot N+3
		//	Attestation_1 (Block C); size 1 - slot N+2 (honest)
		// Steps:
		//	Block A received at N — A is head
		//	Block C received at N+2 — C is head
		//	Block B received at N+2 — C is head
		//	Attestation_1 received at N+3 — C is head
		//	Block D received at N+3 — D is head
		f := setup(jEpoch, fEpoch)
		a := zeroHash

		// The head should always start at the finalized block.
		r, err := f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, zeroHash, r, "Incorrect head with genesis")

		cSlot := types.Slot(2)
		c := indexToHash(2)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				cSlot,
				c,
				a, // parent
				jEpoch,
				fEpoch,
				true,
			),
		)

		// Ensure C is the head.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, c, r, "Incorrect head for justified epoch at slot 2")

		// We boost C.
		secondsPerSlot := time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot)
		genesis := time.Now().Add(-2 * secondsPerSlot)
		require.NoError(t, f.BoostProposerRoot(ctx, cSlot /* slot */, c, genesis))

		bSlot := types.Slot(1)
		b := indexToHash(1)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				bSlot,
				b,
				a, // parent
				jEpoch,
				fEpoch,
				true,
			),
		)

		// Ensure C is still the head.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, c, r, "Incorrect head for justified epoch at slot 2")

		// An attestation for C is received at slot N+3.
		votes := []uint64{1}
		f.ProcessAttestation(ctx, votes, c, fEpoch)

		// A block D, building on B, is received at slot N+3. It should not be able to win without boosting.
		dSlot := types.Slot(3)
		d := indexToHash(3)
		require.NoError(t,
			f.ProcessBlock(
				ctx,
				dSlot,
				d,
				b, // parent
				jEpoch,
				fEpoch,
				true,
			),
		)

		// D cannot win without a boost.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, c, r, "Expected C to remain the head")

		// Block D receives the boost.
		genesis = time.Now().Add(-3 * secondsPerSlot)
		require.NoError(t, f.BoostProposerRoot(ctx, dSlot /* slot */, d, genesis))

		// Ensure D becomes the head thanks to boosting.
		r, err = f.Head(ctx, jEpoch, zeroHash, balances, fEpoch)
		require.NoError(t, err)
		assert.Equal(t, d, r, "Expected D to become the head")
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
		err := f.BoostProposerRoot(ctx, types.Slot(0), blockRoot, genesis)
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
		err := f.BoostProposerRoot(ctx, types.Slot(1), blockRoot, genesis)
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

		err := f.BoostProposerRoot(ctx, types.Slot(1), blockRoot, genesis)
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

		err := f.BoostProposerRoot(ctx, types.Slot(1), blockRoot, genesis)
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
		validatorBalances := make([]uint64, 64) // Num validators
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = 10
		}
		// Avg balance is 10, and the number of validators is 64.
		// With a committee size of num validators (64) / slots per epoch (32) == 2.
		// we then have a committee weight of avg balance * committee size = 10 * 2 = 20.
		// The score then becomes 10 * PROPOSER_SCORE_BOOST // 100, which is
		// 20 * 70 / 100 = 14.
		score, err := computeProposerBoostScore(validatorBalances)
		require.NoError(t, err)
		require.Equal(t, uint64(14), score)
	})
}
