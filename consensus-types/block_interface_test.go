package interfaces

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/consensus-types/forks/altair"
	"github.com/prysmaticlabs/prysm/consensus-types/forks/bellatrix"
	"github.com/prysmaticlabs/prysm/consensus-types/forks/phase0"
	"github.com/prysmaticlabs/prysm/consensus-types/wrappers"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

var (
	_ = SignedBeaconBlock(&phase0.SignedBeaconBlock{})
	_ = BeaconBlock(&phase0.BeaconBlock{})
	_ = BeaconBlockBody(&phase0.BeaconBlockBody{})

	_ = SignedBeaconBlock(&altair.SignedBeaconBlock{})
	_ = BeaconBlock(&altair.BeaconBlock{})
	_ = BeaconBlockBody(&altair.BeaconBlockBody{})

	_ = SignedBeaconBlock(&bellatrix.SignedBeaconBlock{})
	_ = BeaconBlock(&bellatrix.BeaconBlock{})
	_ = BeaconBlockBody(&bellatrix.BeaconBlockBody{})

	_ = SignedBeaconBlock(&bellatrix.SignedBlindedBeaconBlock{})
	_ = BeaconBlock(&bellatrix.BlindedBeaconBlock{})
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
			RandaoReveal:      bytesutil.PadTo([]byte("randao"), fieldparams.BLSSignatureLength),
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

func TestBeaconBlockHeaderFromBlockInterface(t *testing.T) {
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
			RandaoReveal:      bytesutil.PadTo([]byte("randao"), fieldparams.BLSSignatureLength),
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

	bh, err := BeaconBlockHeaderFromBlockInterface(phase0.WrappedBeaconBlock(blk))
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
			RandaoReveal:      bytesutil.PadTo([]byte("randao"), fieldparams.BLSSignatureLength),
			Graffiti:          bytesutil.PadTo([]byte("teehee"), hashLen),
			ProposerSlashings: []*eth.ProposerSlashing{},
			AttesterSlashings: []*eth.AttesterSlashing{},
			Attestations:      []*eth.Attestation{},
			Deposits:          []*eth.Deposit{},
			VoluntaryExits:    []*eth.SignedVoluntaryExit{},
		},
	},
		Signature: bytesutil.PadTo([]byte("signature"), fieldparams.BLSSignatureLength),
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

	bh, err := SignedBeaconBlockHeaderFromBlock(blk)
	require.NoError(t, err)
	assert.DeepEqual(t, want, bh)
}

func TestSignedBeaconBlockHeaderFromBlockInterface(t *testing.T) {
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
			RandaoReveal:      bytesutil.PadTo([]byte("randao"), fieldparams.BLSSignatureLength),
			Graffiti:          bytesutil.PadTo([]byte("teehee"), hashLen),
			ProposerSlashings: []*eth.ProposerSlashing{},
			AttesterSlashings: []*eth.AttesterSlashing{},
			Attestations:      []*eth.Attestation{},
			Deposits:          []*eth.Deposit{},
			VoluntaryExits:    []*eth.SignedVoluntaryExit{},
		},
	},
		Signature: bytesutil.PadTo([]byte("signature"), fieldparams.BLSSignatureLength),
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
	wsb, err := wrappers.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	bh, err := SignedBeaconBlockHeaderFromBlockInterface(wsb)
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
		Signature: bytesutil.PadTo([]byte("signature"), fieldparams.BLSSignatureLength),
	}
	_, err := SignedBeaconBlockHeaderFromBlock(blk)
	require.ErrorContains(t, "nil block", err)
}
