package migration

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

var (
	slot             = types.Slot(1)
	epoch            = types.Epoch(1)
	proposerIndex    = types.ValidatorIndex(1)
	committeeIndex   = types.CommitteeIndex(1)
	depositCount     = uint64(1)
	parentRoot       = bytesutil.PadTo([]byte("parentroot"), 32)
	stateRoot        = bytesutil.PadTo([]byte("stateroot"), 32)
	signature        = bytesutil.PadTo([]byte("signature"), 96)
	randaoReveal     = bytesutil.PadTo([]byte("randaoreveal"), 96)
	depositRoot      = bytesutil.PadTo([]byte("depositroot"), 32)
	blockHash        = bytesutil.PadTo([]byte("blockhash"), 32)
	beaconBlockRoot  = bytesutil.PadTo([]byte("beaconblockroot"), 32)
	sourceRoot       = bytesutil.PadTo([]byte("sourceRoot"), 32)
	targetRoot       = bytesutil.PadTo([]byte("targetRoot"), 32)
	attestingIndices = []uint64{1, 2}
)

func Test_V1Alpha1BlockToV1BlockHeader(t *testing.T) {
	alphaBlock := testutil.HydrateSignedBeaconBlock(&ethpb_alpha.SignedBeaconBlock{})
	alphaBlock.Block.Slot = slot
	alphaBlock.Block.ProposerIndex = proposerIndex
	alphaBlock.Block.ParentRoot = parentRoot
	alphaBlock.Block.StateRoot = stateRoot
	alphaBlock.Signature = signature

	v1Header, err := V1Alpha1BlockToV1BlockHeader(alphaBlock)
	require.NoError(t, err)
	bodyRoot, err := alphaBlock.Block.Body.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, bodyRoot[:], v1Header.Header.BodyRoot)
	assert.Equal(t, slot, v1Header.Header.Slot)
	assert.Equal(t, proposerIndex, v1Header.Header.ProposerIndex)
	assert.DeepEqual(t, parentRoot, v1Header.Header.ParentRoot)
	assert.DeepEqual(t, stateRoot, v1Header.Header.StateRoot)
	assert.DeepEqual(t, signature, v1Header.Signature)
}

func Test_V1Alpha1ToV1Block(t *testing.T) {
	alphaBlock := testutil.HydrateSignedBeaconBlock(&ethpb_alpha.SignedBeaconBlock{})
	alphaBlock.Block.Slot = slot
	alphaBlock.Block.ProposerIndex = proposerIndex
	alphaBlock.Block.ParentRoot = parentRoot
	alphaBlock.Block.StateRoot = stateRoot
	alphaBlock.Block.Body.RandaoReveal = randaoReveal
	alphaBlock.Block.Body.Eth1Data = &ethpb_alpha.Eth1Data{
		DepositRoot:  depositRoot,
		DepositCount: depositCount,
		BlockHash:    blockHash,
	}
	alphaBlock.Signature = signature

	v1Block, err := V1Alpha1ToV1Block(alphaBlock)
	require.NoError(t, err)
	assert.Equal(t, slot, v1Block.Block.Slot)
	assert.Equal(t, proposerIndex, v1Block.Block.ProposerIndex)
	assert.DeepEqual(t, parentRoot, v1Block.Block.ParentRoot)
	assert.DeepEqual(t, stateRoot, v1Block.Block.StateRoot)
	assert.DeepEqual(t, randaoReveal, v1Block.Block.Body.RandaoReveal)
	assert.DeepEqual(t, depositRoot, v1Block.Block.Body.Eth1Data.DepositRoot)
	assert.Equal(t, depositCount, v1Block.Block.Body.Eth1Data.DepositCount)
	assert.DeepEqual(t, blockHash, v1Block.Block.Body.Eth1Data.BlockHash)
	assert.DeepEqual(t, signature, v1Block.Signature)
}

func Test_V1ToV1Alpha1Block(t *testing.T) {
	v1Block := testutil.HydrateV1SignedBeaconBlock(&ethpb.SignedBeaconBlock{})
	v1Block.Block.Slot = slot
	v1Block.Block.ProposerIndex = proposerIndex
	v1Block.Block.ParentRoot = parentRoot
	v1Block.Block.StateRoot = stateRoot
	v1Block.Block.Body.RandaoReveal = randaoReveal
	v1Block.Block.Body.Eth1Data = &ethpb.Eth1Data{
		DepositRoot:  depositRoot,
		DepositCount: depositCount,
		BlockHash:    blockHash,
	}
	v1Block.Signature = signature

	alphaBlock, err := V1ToV1Alpha1Block(v1Block)
	require.NoError(t, err)
	assert.Equal(t, slot, alphaBlock.Block.Slot)
	assert.Equal(t, proposerIndex, alphaBlock.Block.ProposerIndex)
	assert.DeepEqual(t, parentRoot, alphaBlock.Block.ParentRoot)
	assert.DeepEqual(t, stateRoot, alphaBlock.Block.StateRoot)
	assert.DeepEqual(t, randaoReveal, alphaBlock.Block.Body.RandaoReveal)
	assert.DeepEqual(t, depositRoot, alphaBlock.Block.Body.Eth1Data.DepositRoot)
	assert.Equal(t, depositCount, alphaBlock.Block.Body.Eth1Data.DepositCount)
	assert.DeepEqual(t, blockHash, alphaBlock.Block.Body.Eth1Data.BlockHash)
	assert.DeepEqual(t, signature, alphaBlock.Signature)
}

func Test_V1Alpha1AttSlashingToV1(t *testing.T) {
	alphaAttestation := &ethpb_alpha.IndexedAttestation{
		AttestingIndices: attestingIndices,
		Data: &ethpb_alpha.AttestationData{
			Slot:            slot,
			CommitteeIndex:  committeeIndex,
			BeaconBlockRoot: beaconBlockRoot,
			Source: &ethpb_alpha.Checkpoint{
				Epoch: epoch,
				Root:  sourceRoot,
			},
			Target: &ethpb_alpha.Checkpoint{
				Epoch: epoch,
				Root:  targetRoot,
			},
		},
		Signature: signature,
	}
	alphaSlashing := &ethpb_alpha.AttesterSlashing{
		Attestation_1: alphaAttestation,
		Attestation_2: alphaAttestation,
	}

	v1Slashing := V1Alpha1AttSlashingToV1(alphaSlashing)
	assert.DeepEqual(t, attestingIndices, v1Slashing.Attestation_1.AttestingIndices)
	assert.Equal(t, slot, v1Slashing.Attestation_1.Data.Slot)
	assert.Equal(t, committeeIndex, v1Slashing.Attestation_1.Data.CommitteeIndex)
	assert.DeepEqual(t, beaconBlockRoot, v1Slashing.Attestation_1.Data.BeaconBlockRoot)
	assert.Equal(t, epoch, v1Slashing.Attestation_1.Data.Source.Epoch)
	assert.DeepEqual(t, sourceRoot, v1Slashing.Attestation_1.Data.Source.Root)
	assert.Equal(t, epoch, v1Slashing.Attestation_1.Data.Target.Epoch)
	assert.DeepEqual(t, targetRoot, v1Slashing.Attestation_1.Data.Target.Root)
	assert.DeepEqual(t, signature, v1Slashing.Attestation_1.Signature)
	assert.Equal(t, slot, v1Slashing.Attestation_2.Data.Slot)
	assert.Equal(t, committeeIndex, v1Slashing.Attestation_2.Data.CommitteeIndex)
	assert.DeepEqual(t, beaconBlockRoot, v1Slashing.Attestation_2.Data.BeaconBlockRoot)
	assert.Equal(t, epoch, v1Slashing.Attestation_2.Data.Source.Epoch)
	assert.DeepEqual(t, sourceRoot, v1Slashing.Attestation_2.Data.Source.Root)
	assert.Equal(t, epoch, v1Slashing.Attestation_2.Data.Target.Epoch)
	assert.DeepEqual(t, targetRoot, v1Slashing.Attestation_2.Data.Target.Root)
	assert.DeepEqual(t, signature, v1Slashing.Attestation_2.Signature)
}
