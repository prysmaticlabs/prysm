package blocks_test

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProcessSyncCommittee_OK(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	require.NoError(t, beaconState.SetSlot(1))
	syncBits := bitfield.NewBitvector1024()
	for i := range syncBits {
		syncBits[i] = 0xff
	}

	indices, err := helpers.SyncCommitteeIndices(beaconState, helpers.CurrentEpoch(beaconState))
	require.NoError(t, err)
	ps := helpers.PrevSlot(beaconState.Slot())
	pbr, err := helpers.BlockRootAtSlot(beaconState, ps)
	require.NoError(t, err)
	sigs := make([]bls.Signature, len(indices))
	for i, indice := range indices {
		b := types.SSZBytes(pbr)
		sb, err := helpers.ComputeDomainAndSign(beaconState, helpers.CurrentEpoch(beaconState), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		sigs[i] = sig
	}
	aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
	block := testutil.NewBeaconBlock()
	block.Block.Body.SyncCommitteeBits = syncBits
	block.Block.Body.SyncCommitteeSignature = aggregatedSig

	beaconState, err = blocks.ProcessSyncCommittee(beaconState, block.Block.Body)
	require.NoError(t, err)

	// Find a non sync committee index to compare profitability.
	syncCommittee := make(map[uint64]bool)
	for _, index := range indices {
		syncCommittee[index] = true
	}
	nonSyncIndex := params.BeaconConfig().MaxValidatorsPerCommittee + 1
	for i := uint64(0); i < params.BeaconConfig().MaxValidatorsPerCommittee; i++ {
		if !syncCommittee[i] {
			nonSyncIndex = i
			break
		}
	}

	// Sync committee should be more profitable than non sync committee
	balances := beaconState.Balances()
	require.Equal(t, true, balances[indices[0]] > balances[nonSyncIndex])
	// Proposer should be more profitable than sync committee
	proposerIndex, err := helpers.BeaconProposerIndex(beaconState)
	require.NoError(t, err)
	require.Equal(t, true, balances[proposerIndex] > balances[indices[0]])
	// Sync committee should have the same profits, except you are a proposer
	for i := 1; i < len(indices); i++ {
		if proposerIndex == indices[i-1] || proposerIndex == indices[i] {
			continue
		}
		require.Equal(t, balances[indices[i-1]], balances[indices[i]])
	}
	// Increased balance validator count should equal to sync committee count
	increased := uint64(0)
	for _, balance := range balances {
		if balance > params.BeaconConfig().MaxEffectiveBalance {
			increased++
		}
	}
	require.Equal(t, params.BeaconConfig().SyncCommitteeSize, increased)
}
