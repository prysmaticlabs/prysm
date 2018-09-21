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

type mockChainService struct{}

func (f *mockChainService) ContainsBlock(h [32]byte) (bool, error) {
	return true, nil
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

	if b1.data.ParentHash == nil {
		t.Fatalf("genesis block missing ParentHash field")
	}

	if b1.data.RandaoReveal == nil {
		t.Fatalf("genesis block missing RandaoReveal field")
	}

	if b1.data.PowChainRef == nil {
		t.Fatalf("genesis block missing PowChainRef field")
	}

	if b1.data.ActiveStateHash == nil {
		t.Fatalf("genesis block missing ActiveStateHash field")
	}

	if b1.data.CrystallizedStateHash == nil {
		t.Fatalf("genesis block missing CrystallizedStateHash field")
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
		Attestations: []*pb.AggregatedAttestation{
			{
				Slot:             0,
				ShardId:          0,
				JustifiedSlot:    0,
				AttesterBitfield: []byte{64, 0},
			},
		},
	})

	parentSlot := uint64(1)
	chainService := &mockChainService{}

	if !b.isAttestationValid(0, chainService, aState, cState, parentSlot) {
		t.Fatalf("failed attestation validation")
	}

	if !b.IsValid(chainService, aState, cState, parentSlot) {
		t.Fatalf("failed block validation")
	}
}

func TestIsAttestationSlotNumberValid(t *testing.T) {
	if isAttestationSlotNumberValid(2, 1) {
		t.Errorf("attestation slot number can't be higher than parent block's slot number")
	}

	if isAttestationSlotNumberValid(1, params.CycleLength+1) {
		t.Errorf("attestation slot number can't be lower than parent block's slot number by one CycleLength and 1")
	}

	if !isAttestationSlotNumberValid(2, 2) {
		t.Errorf("attestation slot number could be less than or equal to parent block's slot number")
	}

	if !isAttestationSlotNumberValid(2, 10) {
		t.Errorf("attestation slot number could be less than or equal to parent block's slot number")
	}
}
