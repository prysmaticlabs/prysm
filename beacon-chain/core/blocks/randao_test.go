package blocks_test

import (
	"context"
	"encoding/binary"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	testing2 "github.com/prysmaticlabs/prysm/testing"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestProcessRandao_IncorrectProposerFailsVerification(t *testing.T) {
	beaconState, privKeys := testing2.DeterministicGenesisState(t, 100)
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	require.NoError(t, err)
	epoch := types.Epoch(0)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, uint64(epoch))
	domain, err := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainRandao, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	root, err := (&ethpb.SigningData{ObjectRoot: buf, Domain: domain}).HashTreeRoot()
	require.NoError(t, err)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx-1].Sign(root[:])
	b := testing2.NewBeaconBlock()
	b.Block = &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature.Marshal(),
		},
	}

	want := "block randao: signature did not verify"
	_, err = blocks.ProcessRandao(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(b))
	assert.ErrorContains(t, want, err)
}

func TestProcessRandao_SignatureVerifiesAndUpdatesLatestStateMixes(t *testing.T) {
	beaconState, privKeys := testing2.DeterministicGenesisState(t, 100)

	epoch := core.CurrentEpoch(beaconState)
	epochSignature, err := testing2.RandaoReveal(beaconState, epoch, privKeys)
	require.NoError(t, err)

	b := testing2.NewBeaconBlock()
	b.Block = &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature,
		},
	}

	newState, err := blocks.ProcessRandao(
		context.Background(),
		beaconState,
		wrapper.WrappedPhase0SignedBeaconBlock(b),
	)
	require.NoError(t, err, "Unexpected error processing block randao")
	currentEpoch := core.CurrentEpoch(beaconState)
	mix := newState.RandaoMixes()[currentEpoch%params.BeaconConfig().EpochsPerHistoricalVector]
	assert.DeepNotEqual(t, params.BeaconConfig().ZeroHash[:], mix, "Expected empty signature to be overwritten by randao reveal")
}

func TestRandaoSignatureSet_OK(t *testing.T) {
	beaconState, privKeys := testing2.DeterministicGenesisState(t, 100)

	epoch := core.CurrentEpoch(beaconState)
	epochSignature, err := testing2.RandaoReveal(beaconState, epoch, privKeys)
	require.NoError(t, err)

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature,
		},
	}

	set, err := blocks.RandaoSignatureSet(beaconState, block.Body.RandaoReveal)
	require.NoError(t, err)
	verified, err := set.Verify()
	require.NoError(t, err)
	assert.Equal(t, true, verified, "Unable to verify randao signature set")
}
