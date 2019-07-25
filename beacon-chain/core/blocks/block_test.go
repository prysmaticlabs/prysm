package blocks

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestGenesisBlock_InitializedCorrectly(t *testing.T) {
	stateHash := []byte{0}
	b1 := NewGenesisBlock(stateHash)

	if b1.ParentRoot == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if !bytes.Equal(b1.StateRoot, stateHash) {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}
}

func TestHeaderFromBlock(t *testing.T) {
	dummyBody := &ethpb.BeaconBlockBody{
		Eth1Data:          &ethpb.Eth1Data{},
		Graffiti:          []byte{},
		RandaoReveal:      []byte("Reveal"),
		AttesterSlashings: []*ethpb.AttesterSlashing{},
		ProposerSlashings: []*ethpb.ProposerSlashing{},
		Attestations:      []*ethpb.Attestation{},
		Transfers:         []*ethpb.Transfer{},
		Deposits:          []*ethpb.Deposit{},
		VoluntaryExits:    []*ethpb.VoluntaryExit{},
	}

	dummyBlock := &ethpb.BeaconBlock{
		Slot:       10,
		Signature:  []byte{'S'},
		ParentRoot: []byte("Parent"),
		StateRoot:  []byte("State"),
		Body:       dummyBody,
	}

	header, err := HeaderFromBlock(dummyBlock)
	if err != nil {
		t.Fatal(err)
	}

	expectedHeader := &ethpb.BeaconBlockHeader{
		Slot:       dummyBlock.Slot,
		Signature:  dummyBlock.Signature,
		ParentRoot: dummyBlock.ParentRoot,
		StateRoot:  dummyBlock.StateRoot,
	}

	bodyRoot, err := ssz.HashTreeRoot(dummyBody)
	if err != nil {
		t.Fatal(err)
	}

	expectedHeader.BodyRoot = bodyRoot[:]

	if !proto.Equal(expectedHeader, header) {
		t.Errorf("Expected Header not Equal to Retrieved Header. Expected %v , Got %v",
			proto.MarshalTextString(expectedHeader), proto.MarshalTextString(header))
	}
}

func TestBlockFromHeader(t *testing.T) {
	dummyHeader := &ethpb.BeaconBlockHeader{
		Slot:       10,
		Signature:  []byte{'S'},
		ParentRoot: []byte("Parent"),
		StateRoot:  []byte("State"),
	}

	block := BlockFromHeader(dummyHeader)

	expectedBlock := &ethpb.BeaconBlock{
		Slot:       dummyHeader.Slot,
		Signature:  dummyHeader.Signature,
		ParentRoot: dummyHeader.ParentRoot,
		StateRoot:  dummyHeader.StateRoot,
	}

	if !proto.Equal(expectedBlock, block) {
		t.Errorf("Expected block not equal to retrieved block. Expected %v , Got %v",
			proto.MarshalTextString(expectedBlock), proto.MarshalTextString(block))
	}
}
