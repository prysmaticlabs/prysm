package blocks

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGenesisBlock_InitializedCorrectly(t *testing.T) {
	stateHash := []byte{0}
	b1 := NewGenesisBlock(stateHash)

	if b1.ParentRoot == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if !reflect.DeepEqual(b1.Body.Attestations, []*pb.Attestation{}) {
		t.Errorf("genesis block should have 0 attestations")
	}

	if !bytes.Equal(b1.Body.RandaoReveal, params.BeaconConfig().ZeroHash[:]) {
		t.Error("genesis block missing RandaoReveal field")
	}

	if !bytes.Equal(b1.StateRoot, stateHash) {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}
	expectedEth1 := &pb.Eth1Data{
		DepositRoot: params.BeaconConfig().ZeroHash[:],
		BlockHash:   params.BeaconConfig().ZeroHash[:],
	}
	if !proto.Equal(b1.Body.Eth1Data, expectedEth1) {
		t.Error("genesis block Eth1Data isn't initialized correctly")
	}
}

func TestHeaderFromBlock(t *testing.T) {
	dummyBody := &pb.BeaconBlockBody{
		RandaoReveal: []byte("Reveal"),
	}

	dummyBlock := &pb.BeaconBlock{
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

	expectedHeader := &pb.BeaconBlockHeader{
		Slot:       dummyBlock.Slot,
		Signature:  dummyBlock.Signature,
		ParentRoot: dummyBlock.ParentRoot,
		StateRoot:  dummyBlock.StateRoot,
	}

	bodyRoot, err := ssz.TreeHash(dummyBody)
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
	dummyHeader := &pb.BeaconBlockHeader{
		Slot:       10,
		Signature:  []byte{'S'},
		ParentRoot: []byte("Parent"),
		StateRoot:  []byte("State"),
	}

	block := BlockFromHeader(dummyHeader)

	expectedBlock := &pb.BeaconBlock{
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
