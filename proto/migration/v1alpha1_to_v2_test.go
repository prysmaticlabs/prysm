package migration

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestV1Alpha1SignedContributionAndProofToV2(t *testing.T) {
	alphaContribution := &ethpbalpha.SignedContributionAndProof{
		Message: &ethpbalpha.ContributionAndProof{
			AggregatorIndex: validatorIndex,
			Contribution: &ethpbalpha.SyncCommitteeContribution{
				Slot:              slot,
				BlockRoot:         blockHash,
				SubcommitteeIndex: 1,
				AggregationBits:   bitfield.NewBitvector128(),
				Signature:         signature,
			},
			SelectionProof: signature,
		},
		Signature: signature,
	}
	v2Contribution := V1Alpha1SignedContributionAndProofToV2(alphaContribution)
	require.NotNil(t, v2Contribution)
	require.NotNil(t, v2Contribution.Message)
	require.NotNil(t, v2Contribution.Message.Contribution)
	assert.DeepEqual(t, signature, v2Contribution.Signature)
	msg := v2Contribution.Message
	assert.Equal(t, validatorIndex, msg.AggregatorIndex)
	assert.DeepEqual(t, signature, msg.SelectionProof)
	contrib := msg.Contribution
	assert.Equal(t, slot, contrib.Slot)
	assert.DeepEqual(t, blockHash, contrib.BeaconBlockRoot)
	assert.Equal(t, uint64(1), contrib.SubcommitteeIndex)
	assert.DeepEqual(t, bitfield.NewBitvector128(), contrib.AggregationBits)
	assert.DeepEqual(t, signature, contrib.Signature)
}
func Test_V1Alpha1BeaconBlockAltairToV2(t *testing.T) {
	alphaBlock := util.HydrateBeaconBlockAltair(&ethpbalpha.BeaconBlockAltair{})
	alphaBlock.Slot = slot
	alphaBlock.ProposerIndex = validatorIndex
	alphaBlock.ParentRoot = parentRoot
	alphaBlock.StateRoot = stateRoot
	alphaBlock.Body.RandaoReveal = randaoReveal
	alphaBlock.Body.Eth1Data = &ethpbalpha.Eth1Data{
		DepositRoot:  depositRoot,
		DepositCount: depositCount,
		BlockHash:    blockHash,
	}
	syncCommitteeBits := bitfield.NewBitvector512()
	syncCommitteeBits.SetBitAt(100, true)
	alphaBlock.Body.SyncAggregate = &ethpbalpha.SyncAggregate{
		SyncCommitteeBits:      syncCommitteeBits,
		SyncCommitteeSignature: signature,
	}

	v2Block, err := V1Alpha1BeaconBlockAltairToV2(alphaBlock)
	require.NoError(t, err)
	alphaRoot, err := alphaBlock.HashTreeRoot()
	require.NoError(t, err)
	v2Root, err := v2Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, alphaRoot, v2Root)
}

func Test_AltairToV1Alpha1SignedBlock(t *testing.T) {
	v2Block := util.HydrateV2SignedBeaconBlock(&ethpbv2.SignedBeaconBlockAltair{})
	v2Block.Message.Slot = slot
	v2Block.Message.ProposerIndex = validatorIndex
	v2Block.Message.ParentRoot = parentRoot
	v2Block.Message.StateRoot = stateRoot
	v2Block.Message.Body.RandaoReveal = randaoReveal
	v2Block.Message.Body.Eth1Data = &ethpbv1.Eth1Data{
		DepositRoot:  depositRoot,
		DepositCount: depositCount,
		BlockHash:    blockHash,
	}
	syncCommitteeBits := bitfield.NewBitvector512()
	syncCommitteeBits.SetBitAt(100, true)
	v2Block.Message.Body.SyncAggregate = &ethpbv1.SyncAggregate{
		SyncCommitteeBits:      syncCommitteeBits,
		SyncCommitteeSignature: signature,
	}
	v2Block.Signature = signature

	alphaBlock, err := AltairToV1Alpha1SignedBlock(v2Block)
	require.NoError(t, err)
	alphaRoot, err := alphaBlock.HashTreeRoot()
	require.NoError(t, err)
	v2Root, err := v2Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, v2Root, alphaRoot)
}
