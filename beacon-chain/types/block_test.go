package types

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/ptypes"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type mockDB struct{}

func (f *mockDB) HasBlock(h [32]byte) bool {
	return true
}

func TestGenesisBlock(t *testing.T) {
	aStateHash := [32]byte{0}
	cStateHash := [32]byte{1}
	b1 := NewGenesisBlock(aStateHash, cStateHash)
	b2 := NewGenesisBlock(aStateHash, cStateHash)

	// We ensure that initializing a proto timestamp from
	// genesis time will lead to no error.
	if _, err := ptypes.TimestampProto(params.GetConfig().GenesisTime); err != nil {
		t.Errorf("could not create proto timestamp, expected no error: %v", err)
	}

	h1, err1 := b1.Hash()
	h2, err2 := b2.Hash()
	if err1 != nil || err2 != nil {
		t.Fatalf("failed to hash genesis block: %v %v", err1, err2)
	}

	if h1 != h2 {
		t.Fatalf("genesis block hash should be identical: %#x %#x", h1, h2)
	}

	if b1.data.AncestorHashes == nil {
		t.Fatal("genesis block missing ParentHash field")
	}

	if b1.Specials() == nil {
		t.Fatal("genesis block missing Special field")
	}

	if b1.data.RandaoReveal == nil {
		t.Fatal("genesis block missing RandaoReveal field")
	}

	if b1.data.PowChainRef == nil {
		t.Fatal("genesis block missing PowChainRef field")
	}

	if !bytes.Equal(b1.data.ActiveStateRoot, aStateHash[:]) {
		t.Fatal("genesis block ActiveStateHash isn't initialized correctly")
	}

	if !bytes.Equal(b1.data.CrystallizedStateRoot, cStateHash[:]) {
		t.Fatal("genesis block CrystallizedStateHash isn't initialized correctly")
	}

	b3 := NewBlock(nil)
	h3, err3 := b3.Hash()
	if err3 != nil {
		t.Fatalf("failed to hash genesis block: %v", err3)
	}

	if h1 == h3 {
		t.Fatalf("genesis block and new block should not have identical hash: %#x", h1)
	}
}

func TestBlockValidity(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("failed to generate crystallized state: %v", err)
	}

	recentBlockHashes := make([][]byte, 2*params.GetConfig().CycleLength)
	for i := 0; i < 2*int(params.GetConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 32))
	}
	aState := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHashes,
	}, make(map[[32]byte]*utils.VoteCache))

	randaoPreCommit := [32]byte{'A'}
	hashedRandaoPreCommit := hashutil.Hash(randaoPreCommit[:])
	cState.data.Validators[1].RandaoCommitment = hashedRandaoPreCommit[:]

	b := NewBlock(&pb.BeaconBlock{
		Slot:         1,
		RandaoReveal: randaoPreCommit[:],
		Attestations: []*pb.AggregatedAttestation{
			{
				Slot:             0,
				Shard:            1,
				JustifiedSlot:    0,
				AttesterBitfield: []byte{128, 0},
			},
		},
	})

	parentSlot := uint64(0)
	db := &mockDB{}

	if !b.isAttestationValid(0, db, aState, cState, parentSlot) {
		t.Fatalf("failed attestation validation")
	}

	genesisTime := params.GetConfig().GenesisTime
	if !b.IsValid(db, aState, cState, parentSlot, genesisTime) {
		t.Fatalf("failed block validation")
	}
}

func TestBlockValidityNoParentProposer(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("failed to generate crystallized state: %v", err)
	}

	recentBlockHashes := make([][]byte, 2*params.GetConfig().CycleLength)
	for i := 0; i < 2*int(params.GetConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 32))
	}

	aState := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHashes,
	}, make(map[[32]byte]*utils.VoteCache))
	parentSlot := uint64(1)
	db := &mockDB{}

	// Test case with invalid RANDAO reveal.
	badRandaoBlock := NewBlock(&pb.BeaconBlock{
		Slot:         2,
		RandaoReveal: []byte{'B'},
		Attestations: []*pb.AggregatedAttestation{
			{
				Slot:             0,
				Shard:            1,
				JustifiedSlot:    0,
				AttesterBitfield: []byte{64, 0},
			},
		},
	})
	genesisTime := params.GetConfig().GenesisTime
	if badRandaoBlock.IsValid(db, aState, cState, parentSlot, genesisTime) {
		t.Fatalf("should have failed doesParentProposerExist")
	}
}

func TestBlockValidityInvalidRandao(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("failed to generate crystallized state: %v", err)
	}

	recentBlockHashes := make([][]byte, 2*params.GetConfig().CycleLength)
	for i := 0; i < 2*int(params.GetConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 32))
	}

	aState := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHashes,
	}, make(map[[32]byte]*utils.VoteCache))
	parentSlot := uint64(0)
	db := &mockDB{}

	// Test case with invalid RANDAO reveal.
	badRandaoBlock := NewBlock(&pb.BeaconBlock{
		Slot:         1,
		RandaoReveal: []byte{'B'},
		Attestations: []*pb.AggregatedAttestation{
			{
				Slot:             0,
				Shard:            1,
				JustifiedSlot:    0,
				AttesterBitfield: []byte{64, 0},
			},
		},
	})

	genesisTime := params.GetConfig().GenesisTime
	if badRandaoBlock.IsValid(db, aState, cState, parentSlot, genesisTime) {
		t.Fatalf("should have failed with invalid RANDAO")
	}
}

func TestIsAttestationSlotNumberValid(t *testing.T) {
	if isAttestationSlotNumberValid(2, 1) {
		t.Errorf("attestation slot number can't be higher than parent block's slot number")
	}

	if isAttestationSlotNumberValid(1, params.GetConfig().CycleLength+1) {
		t.Errorf("attestation slot number can't be lower than parent block's slot number by one CycleLength and 1")
	}

	if !isAttestationSlotNumberValid(2, 2) {
		t.Errorf("attestation slot number could be less than or equal to parent block's slot number")
	}

	if !isAttestationSlotNumberValid(2, 10) {
		t.Errorf("attestation slot number could be less than or equal to parent block's slot number")
	}
}

func TestUpdateAncestorHashes(t *testing.T) {
	parentHashes := make([][32]byte, 32)
	for i := 0; i < 32; i++ {
		parentHashes[i] = hashutil.Hash([]byte{byte(i)})
	}

	tests := []struct {
		a uint64
		b [32]byte
		c int
	}{
		{a: 1, b: [32]byte{'a'}, c: 0},
		{a: 2, b: [32]byte{'b'}, c: 1},
		{a: 4, b: [32]byte{'c'}, c: 2},
		{a: 8, b: [32]byte{'d'}, c: 3},
		{a: 16, b: [32]byte{'e'}, c: 4},
		{a: 1 << 29, b: [32]byte{'f'}, c: 29},
		{a: 1 << 30, b: [32]byte{'g'}, c: 30},
		{a: 1 << 31, b: [32]byte{'h'}, c: 31},
	}
	for _, tt := range tests {
		if UpdateAncestorHashes(parentHashes, tt.a, tt.b)[tt.c] != tt.b {
			t.Errorf("Failed to update ancestor hash at index %d. Wanted: %v, got: %v", tt.c, tt.b, UpdateAncestorHashes(parentHashes, tt.a, tt.b)[tt.c])
		}
	}
}
