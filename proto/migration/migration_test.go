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
	validatorIndex   = types.ValidatorIndex(1)
	committeeIndex   = types.CommitteeIndex(1)
	depositCount     = uint64(2)
	attestingIndices = []uint64{1, 2}
	parentRoot       = bytesutil.PadTo([]byte("parentroot"), 32)
	stateRoot        = bytesutil.PadTo([]byte("stateroot"), 32)
	signature        = bytesutil.PadTo([]byte("signature"), 96)
	randaoReveal     = bytesutil.PadTo([]byte("randaoreveal"), 96)
	depositRoot      = bytesutil.PadTo([]byte("depositroot"), 32)
	blockHash        = bytesutil.PadTo([]byte("blockhash"), 32)
	beaconBlockRoot  = bytesutil.PadTo([]byte("beaconblockroot"), 32)
	sourceRoot       = bytesutil.PadTo([]byte("sourceroot"), 32)
	targetRoot       = bytesutil.PadTo([]byte("targetroot"), 32)
	bodyRoot         = bytesutil.PadTo([]byte("bodyroot"), 32)
)

func Test_V1Alpha1BlockToV1BlockHeader(t *testing.T) {
	alphaBlock := testutil.HydrateSignedBeaconBlock(&ethpb_alpha.SignedBeaconBlock{})
	alphaBlock.Block.Slot = slot
	alphaBlock.Block.ProposerIndex = validatorIndex
	alphaBlock.Block.ParentRoot = parentRoot
	alphaBlock.Block.StateRoot = stateRoot
	alphaBlock.Signature = signature

	v1Header, err := V1Alpha1BlockToV1BlockHeader(alphaBlock)
	require.NoError(t, err)
	bodyRoot, err := alphaBlock.Block.Body.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, bodyRoot[:], v1Header.Header.BodyRoot)
	assert.Equal(t, slot, v1Header.Header.Slot)
	assert.Equal(t, validatorIndex, v1Header.Header.ProposerIndex)
	assert.DeepEqual(t, parentRoot, v1Header.Header.ParentRoot)
	assert.DeepEqual(t, stateRoot, v1Header.Header.StateRoot)
	assert.DeepEqual(t, signature, v1Header.Signature)
}

func Test_V1Alpha1ToV1Block(t *testing.T) {
	alphaBlock := testutil.HydrateSignedBeaconBlock(&ethpb_alpha.SignedBeaconBlock{})
	alphaBlock.Block.Slot = slot
	alphaBlock.Block.ProposerIndex = validatorIndex
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
	assert.Equal(t, validatorIndex, v1Block.Block.ProposerIndex)
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
	v1Block.Block.ProposerIndex = validatorIndex
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
	assert.Equal(t, validatorIndex, alphaBlock.Block.ProposerIndex)
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

func Test_V1Alpha1ProposerSlashingToV1(t *testing.T) {
	alphaHeader := testutil.HydrateSignedBeaconHeader(&ethpb_alpha.SignedBeaconBlockHeader{})
	alphaHeader.Header.Slot = slot
	alphaHeader.Header.ProposerIndex = validatorIndex
	alphaHeader.Header.ParentRoot = parentRoot
	alphaHeader.Header.StateRoot = stateRoot
	alphaHeader.Header.BodyRoot = bodyRoot
	alphaHeader.Signature = signature
	alphaSlashing := &ethpb_alpha.ProposerSlashing{
		Header_1: alphaHeader,
		Header_2: alphaHeader,
	}

	v1Slashing := V1Alpha1ProposerSlashingToV1(alphaSlashing)
	assert.Equal(t, slot, v1Slashing.Header_1.Header.Slot)
	assert.Equal(t, validatorIndex, v1Slashing.Header_1.Header.ProposerIndex)
	assert.DeepEqual(t, parentRoot, v1Slashing.Header_1.Header.ParentRoot)
	assert.DeepEqual(t, stateRoot, v1Slashing.Header_1.Header.StateRoot)
	assert.DeepEqual(t, bodyRoot, v1Slashing.Header_1.Header.BodyRoot)
	assert.DeepEqual(t, signature, v1Slashing.Header_1.Signature)
	assert.Equal(t, slot, v1Slashing.Header_2.Header.Slot)
	assert.Equal(t, validatorIndex, v1Slashing.Header_2.Header.ProposerIndex)
	assert.DeepEqual(t, parentRoot, v1Slashing.Header_2.Header.ParentRoot)
	assert.DeepEqual(t, stateRoot, v1Slashing.Header_2.Header.StateRoot)
	assert.DeepEqual(t, bodyRoot, v1Slashing.Header_2.Header.BodyRoot)
	assert.DeepEqual(t, signature, v1Slashing.Header_2.Signature)
}

func Test_V1Alpha1ExitToV1(t *testing.T) {
	alphaExit := &ethpb_alpha.SignedVoluntaryExit{
		Exit: &ethpb_alpha.VoluntaryExit{
			Epoch:          epoch,
			ValidatorIndex: validatorIndex,
		},
		Signature: signature,
	}

	v1Exit := V1Alpha1ExitToV1(alphaExit)
	assert.Equal(t, epoch, v1Exit.Exit.Epoch)
	assert.Equal(t, validatorIndex, v1Exit.Exit.ValidatorIndex)
	assert.DeepEqual(t, signature, v1Exit.Signature)
}

func Test_V1AttSlashingToV1Alpha1(t *testing.T) {
	v1Attestation := &ethpb.IndexedAttestation{
		AttestingIndices: attestingIndices,
		Data: &ethpb.AttestationData{
			Slot:            slot,
			CommitteeIndex:  committeeIndex,
			BeaconBlockRoot: beaconBlockRoot,
			Source: &ethpb.Checkpoint{
				Epoch: epoch,
				Root:  sourceRoot,
			},
			Target: &ethpb.Checkpoint{
				Epoch: epoch,
				Root:  targetRoot,
			},
		},
		Signature: signature,
	}
	v1Slashing := &ethpb.AttesterSlashing{
		Attestation_1: v1Attestation,
		Attestation_2: v1Attestation,
	}

	alphaSlashing := V1AttSlashingToV1Alpha1(v1Slashing)
	assert.DeepEqual(t, attestingIndices, alphaSlashing.Attestation_1.AttestingIndices)
	assert.Equal(t, slot, alphaSlashing.Attestation_1.Data.Slot)
	assert.Equal(t, committeeIndex, alphaSlashing.Attestation_1.Data.CommitteeIndex)
	assert.DeepEqual(t, beaconBlockRoot, alphaSlashing.Attestation_1.Data.BeaconBlockRoot)
	assert.Equal(t, epoch, alphaSlashing.Attestation_1.Data.Source.Epoch)
	assert.DeepEqual(t, sourceRoot, alphaSlashing.Attestation_1.Data.Source.Root)
	assert.Equal(t, epoch, alphaSlashing.Attestation_1.Data.Target.Epoch)
	assert.DeepEqual(t, targetRoot, alphaSlashing.Attestation_1.Data.Target.Root)
	assert.DeepEqual(t, signature, alphaSlashing.Attestation_1.Signature)
	assert.Equal(t, slot, alphaSlashing.Attestation_2.Data.Slot)
	assert.Equal(t, committeeIndex, alphaSlashing.Attestation_2.Data.CommitteeIndex)
	assert.DeepEqual(t, beaconBlockRoot, alphaSlashing.Attestation_2.Data.BeaconBlockRoot)
	assert.Equal(t, epoch, alphaSlashing.Attestation_2.Data.Source.Epoch)
	assert.DeepEqual(t, sourceRoot, alphaSlashing.Attestation_2.Data.Source.Root)
	assert.Equal(t, epoch, alphaSlashing.Attestation_2.Data.Target.Epoch)
	assert.DeepEqual(t, targetRoot, alphaSlashing.Attestation_2.Data.Target.Root)
	assert.DeepEqual(t, signature, alphaSlashing.Attestation_2.Signature)
}
