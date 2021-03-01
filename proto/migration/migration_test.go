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
	alphaRoot, err := alphaBlock.HashTreeRoot()
	require.NoError(t, err)
	v1Root, err := v1Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, alphaRoot, v1Root)
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
	alphaRoot, err := alphaBlock.HashTreeRoot()
	require.NoError(t, err)
	v1Root, err := v1Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, v1Root, alphaRoot)
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
	alphaRoot, err := alphaSlashing.HashTreeRoot()
	require.NoError(t, err)
	v1Root, err := v1Slashing.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, alphaRoot, v1Root)
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
	alphaRoot, err := alphaSlashing.HashTreeRoot()
	require.NoError(t, err)
	v1Root, err := v1Slashing.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, alphaRoot, v1Root)
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
	alphaRoot, err := alphaExit.HashTreeRoot()
	require.NoError(t, err)
	v1Root, err := v1Exit.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, alphaRoot, v1Root)
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
	alphaRoot, err := alphaSlashing.HashTreeRoot()
	require.NoError(t, err)
	v1Root, err := v1Slashing.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, v1Root, alphaRoot)
}
