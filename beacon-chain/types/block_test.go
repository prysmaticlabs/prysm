package types

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestGenesisBlock(t *testing.T) {
	b1, err1 := NewGenesisBlock()
	b2, err2 := NewGenesisBlock()
	if err1 != nil || err2 != nil {
		t.Fatalf("failed to instantiate genesis block: %v %v", err1, err2)
	}

	h1, err1 := b1.Hash()
	h2, err2 := b2.Hash()
	if err1 != nil || err2 != nil {
		t.Fatalf("failed to hash genesis block: %v %v", err1, err2)
	}

	if h1 != h2 {
		t.Fatalf("genesis block hash should be identical: %x %x", h1, h2)
	}

	b3 := NewBlock(nil)
	h3, err3 := b3.Hash()
	if err3 != nil {
		t.Fatalf("failed to hash genesis block: %v", err3)
	}

	if h1 == h3 {
		t.Fatalf("genesis block and new block should not have identical hash: %x", h1)
	}
}

func TestBlockValidity(t *testing.T) {
	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("failed to generate crystallized state: %v", err)
	}

	recentBlockHashes := make([][]byte, 2*params.CycleLength)
	for i := 0; i < 2*params.CycleLength; i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 32))
	}
	aState := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHashes,
	}, make(map[[32]byte]*VoteCache))

	b := NewBlock(&pb.BeaconBlock{
		SlotNumber: 1,
		Attestations: []*pb.AttestationRecord{
			{
				Slot:             0,
				ShardId:          0,
				JustifiedSlot:    0,
				AttesterBitfield: []byte{8, 8},
			},
		},
	})

	if !b.isAttestationValid(0, aState, cState) {
		t.Fatalf("failed attestation validation")
	}

	if !b.IsValid(aState, cState) {
		t.Fatalf("failed block validation")
	}
}
