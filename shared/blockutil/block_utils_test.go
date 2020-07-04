package blockutil

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	bodyRoot, err := stateutil.BlockBodyRoot(blk.Body)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to get body root of block"))
	}
	want := &eth.BeaconBlockHeader{
		Slot:          blk.Slot,
		ProposerIndex: blk.ProposerIndex,
		ParentRoot:    blk.ParentRoot,
		StateRoot:     blk.StateRoot,
		BodyRoot:      bodyRoot[:],
	}

	bh, err := BeaconBlockHeaderFromBlock(blk)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, bh) {
		t.Errorf("BeaconBlockHeaderFromBlock() got = %v, want %v", bh, want)
	}

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
	bodyRoot, err := stateutil.BlockBodyRoot(blk.Block.Body)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to get body root of block"))
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, bh) {
		t.Errorf("SignedBeaconBlockHeaderFromBlock() got = %v, want %v", bh, want)
	}

}
