package state

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type mockDB struct {
	blockVoteCache utils.BlockVoteCache
}

func (f *mockDB) HasBlock(h [32]byte) bool {
	return true
}

func (f *mockDB) ReadBlockVoteCache(blockHashes [][32]byte) (utils.BlockVoteCache, error) {
	return f.blockVoteCache, nil
}

func TestBlockValidity(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("failed to generate beacon state: %v", err)
	}

	recentBlockHashes := make([][]byte, 2*params.BeaconConfig().CycleLength)
	for i := 0; i < 2*int(params.BeaconConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 32))
	}
	randaoPreCommit := [32]byte{'A'}
	hashedRandaoPreCommit := hashutil.Hash(randaoPreCommit[:])
	validators := beaconState.ValidatorRegistry()
	validators[1].RandaoCommitmentHash32 = hashedRandaoPreCommit[:]
	beaconState.SetValidatorRegistry(validators)
	beaconState.SetLatestBlockHashes(recentBlockHashes)

	b := types.NewBlock(&pb.BeaconBlock{
		Slot:               1,
		RandaoRevealHash32: randaoPreCommit[:],
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{
				{
					ParticipationBitfield: []byte{128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
						0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					Data: &pb.AttestationData{
						Slot:          0,
						Shard:         1,
						JustifiedSlot: 0,
					},
				},
			},
		},
	})

	parentSlot := uint64(0)
	db := &mockDB{}

	genesisTime := params.BeaconConfig().GenesisTime
	if err := IsValidBlock(b, beaconState, parentSlot, genesisTime, db.HasBlock); err != nil {
		t.Fatalf("failed block validation: %v", err)
	}
}

func TestBlockValidityNoParentProposer(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("failed to generate beacon state: %v", err)
	}

	recentBlockHashes := make([][]byte, 2*params.BeaconConfig().CycleLength)
	for i := 0; i < 2*int(params.BeaconConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 32))
	}

	beaconState.SetLatestBlockHashes(recentBlockHashes)

	parentSlot := uint64(1)
	db := &mockDB{}

	// Test case with invalid RANDAO reveal.
	badRandaoBlock := types.NewBlock(&pb.BeaconBlock{
		Slot:               2,
		RandaoRevealHash32: []byte{'B'},
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{
				{
					ParticipationBitfield: []byte{64, 0},
					Data: &pb.AttestationData{
						Slot:          0,
						Shard:         1,
						JustifiedSlot: 0,
					},
				},
			},
		},
	})
	genesisTime := params.BeaconConfig().GenesisTime
	if err := IsValidBlock(badRandaoBlock, beaconState, parentSlot, genesisTime, db.HasBlock); err == nil {
		t.Fatal("test should have failed without a parent proposer")
	}
}

func TestBlockValidityInvalidRandao(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("failed to generate beacon state: %v", err)
	}

	recentBlockHashes := make([][]byte, 2*params.BeaconConfig().CycleLength)
	for i := 0; i < 2*int(params.BeaconConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 32))
	}

	beaconState.SetLatestBlockHashes(recentBlockHashes)

	parentSlot := uint64(0)
	db := &mockDB{}

	// Test case with invalid RANDAO reveal.
	badRandaoBlock := types.NewBlock(&pb.BeaconBlock{
		Slot:               1,
		RandaoRevealHash32: []byte{'B'},
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{
				{
					ParticipationBitfield: []byte{64, 0},
					Data: &pb.AttestationData{
						Slot:          0,
						Shard:         1,
						JustifiedSlot: 0,
					},
				},
			},
		},
	})

	genesisTime := params.BeaconConfig().GenesisTime
	if err := IsValidBlock(badRandaoBlock, beaconState, parentSlot, genesisTime, db.HasBlock); err == nil {
		t.Fatal("should have failed with invalid RANDAO")
	}
}

func TestIsAttestationSlotNumberValid(t *testing.T) {
	if err := isAttestationSlotNumberValid(2, 1); err == nil {
		t.Error("attestation slot number can't be higher than parent block's slot number")
	}

	if err := isAttestationSlotNumberValid(1, params.BeaconConfig().CycleLength+1); err == nil {
		t.Error("attestation slot number can't be lower than parent block's slot number by one CycleLength and 1")
	}

	if err := isAttestationSlotNumberValid(2, 2); err != nil {
		t.Errorf("attestation slot number could be less than or equal to parent block's slot number: %v", err)
	}

	if err := isAttestationSlotNumberValid(2, 10); err != nil {
		t.Errorf("attestation slot number could be less than or equal to parent block's slot number: %v", err)
	}
}
