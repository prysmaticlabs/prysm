package blockutil

import (
	"testing"

	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconBlockHeaderFromBlock(t *testing.T) {
	hashLen := 32
	blk := &eth.BeaconBlock{
		Slot:          200,
		ProposerIndex: 2,
		ParentRoot:    bytesutil.PadTo([]byte("parent root"), hashLen),
		StateRoot:     bytesutil.PadTo([]byte("state root"), hashLen),
		Body: &eth.BeaconBlockBody{
			Eth1Data: &eth.Eth1Data{
				BlockHash:    bytesutil.PadTo([]byte("block hash"), hashLen),
				DepositRoot:  bytesutil.PadTo([]byte("deposit root"), hashLen),
				DepositCount: 1,
			},
			RandaoReveal:      bytesutil.PadTo([]byte("randao"), params.BeaconConfig().BLSSignatureLength),
			Graffiti:          bytesutil.PadTo([]byte("teehee"), hashLen),
			ProposerSlashings: []*eth.ProposerSlashing{},
			AttesterSlashings: []*eth.AttesterSlashing{},
			Attestations:      []*eth.Attestation{},
			Deposits:          []*eth.Deposit{},
			VoluntaryExits:    []*eth.SignedVoluntaryExit{},
		},
	}
	bodyRoot, err := blk.Body.HashTreeRoot()
	require.NoError(t, err)
	want := &eth.BeaconBlockHeader{
		Slot:          blk.Slot,
		ProposerIndex: blk.ProposerIndex,
		ParentRoot:    blk.ParentRoot,
		StateRoot:     blk.StateRoot,
		BodyRoot:      bodyRoot[:],
	}

	bh, err := BeaconBlockHeaderFromBlock(blk)
	require.NoError(t, err)
	assert.DeepEqual(t, want, bh)
}

func TestBeaconBlockHeaderFromBlock_NilBlockBody(t *testing.T) {
	hashLen := 32
	blk := &eth.BeaconBlock{
		Slot:          200,
		ProposerIndex: 2,
		ParentRoot:    bytesutil.PadTo([]byte("parent root"), hashLen),
		StateRoot:     bytesutil.PadTo([]byte("state root"), hashLen),
	}
	_, err := BeaconBlockHeaderFromBlock(blk)
	require.ErrorContains(t, "nil block body", err)
}

func TestSignedBeaconBlockHeaderFromBlock(t *testing.T) {
	hashLen := 32
	blk := &eth.SignedBeaconBlock{Block: &eth.BeaconBlock{
		Slot:          200,
		ProposerIndex: 2,
		ParentRoot:    bytesutil.PadTo([]byte("parent root"), hashLen),
		StateRoot:     bytesutil.PadTo([]byte("state root"), hashLen),
		Body: &eth.BeaconBlockBody{
			Eth1Data: &eth.Eth1Data{
				BlockHash:    bytesutil.PadTo([]byte("block hash"), hashLen),
				DepositRoot:  bytesutil.PadTo([]byte("deposit root"), hashLen),
				DepositCount: 1,
			},
			RandaoReveal:      bytesutil.PadTo([]byte("randao"), params.BeaconConfig().BLSSignatureLength),
			Graffiti:          bytesutil.PadTo([]byte("teehee"), hashLen),
			ProposerSlashings: []*eth.ProposerSlashing{},
			AttesterSlashings: []*eth.AttesterSlashing{},
			Attestations:      []*eth.Attestation{},
			Deposits:          []*eth.Deposit{},
			VoluntaryExits:    []*eth.SignedVoluntaryExit{},
		},
	},
		Signature: bytesutil.PadTo([]byte("signature"), params.BeaconConfig().BLSSignatureLength),
	}
	bodyRoot, err := blk.Block.Body.HashTreeRoot()
	require.NoError(t, err)
	want := &eth.SignedBeaconBlockHeader{Header: &eth.BeaconBlockHeader{
		Slot:          blk.Block.Slot,
		ProposerIndex: blk.Block.ProposerIndex,
		ParentRoot:    blk.Block.ParentRoot,
		StateRoot:     blk.Block.StateRoot,
		BodyRoot:      bodyRoot[:],
	},
		Signature: blk.Signature,
	}

	wrappedBlk := wrapper.WrappedPhase0SignedBeaconBlock(blk)
	bh, err := SignedBeaconBlockHeaderFromBlock(wrappedBlk)
	require.NoError(t, err)
	assert.DeepEqual(t, want, bh)
}

func TestSignedBeaconBlockHeaderFromBlock_NilBlockBody(t *testing.T) {
	hashLen := 32
	blk := &eth.SignedBeaconBlock{Block: &eth.BeaconBlock{
		Slot:          200,
		ProposerIndex: 2,
		ParentRoot:    bytesutil.PadTo([]byte("parent root"), hashLen),
		StateRoot:     bytesutil.PadTo([]byte("state root"), hashLen),
	},
		Signature: bytesutil.PadTo([]byte("signature"), params.BeaconConfig().BLSSignatureLength),
	}
	wrappedBlk := wrapper.WrappedPhase0SignedBeaconBlock(blk)
	_, err := SignedBeaconBlockHeaderFromBlock(wrappedBlk)
	require.ErrorContains(t, "nil block", err)
}
