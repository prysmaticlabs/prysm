package types

import (
	"bytes"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestActiveState(t *testing.T) {
	active, _, err := NewGenesisStates()
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}
	if len(active.PendingAttestations()) > 0 {
		t.Errorf("there should be no pending attestations, got %v", len(active.PendingAttestations()))
	}
	if active.LatestPendingAttestation() != nil {
		t.Errorf("there should be no latest pending attestation, got %v", active.LatestPendingAttestation())
	}

	record := NewAttestationRecord()
	active.NewPendingAttestation(record)
	if len(active.PendingAttestations()) != 1 {
		t.Errorf("there should be 1 pending attestation, got %v", len(active.PendingAttestations()))
	}
	if !reflect.DeepEqual(active.LatestPendingAttestation(), record) {
		t.Errorf("latest pending attestation record did not match, received %v", active.LatestPendingAttestation())
	}
	active.ClearPendingAttestations()
	if !reflect.DeepEqual(active.LatestPendingAttestation(), &pb.AttestationRecord{}) {
		t.Errorf("latest pending attestation record did not match, received %v", active.LatestPendingAttestation())
	}

	if !reflect.DeepEqual(active.data, active.Proto()) {
		t.Errorf("inner active state data did not match proto: received %v, wanted %v", active.Proto(), active.data)
	}

	active.ClearRecentBlockHashes()
	if len(active.data.RecentBlockHashes) > 0 {
		t.Errorf("there should be no recent block hashes, received %v", len(active.data.RecentBlockHashes))
	}

	bvc := active.GetBlockVoteCache()
	bvc[nil] = &VoteCache{
		VoterIndices:     []uint32{0, 1, 2},
		VoteTotalDeposit: 1000,
	}
	active.SetBlockVoteCache(bvc)
	if !active.IsVoteCacheThere(nil) {
		t.Errorf("block vote cache should be there but recevied false")
	}

	emptyActive := &ActiveState{}
	if _, err := emptyActive.Marshal(); err == nil {
		t.Error("marshal with empty data should fail")
	}
	if _, err := emptyActive.Hash(); err == nil {
		t.Error("hash with empty data should fail")
	}
	if _, err := active.Hash(); err != nil {
		t.Errorf("hashing with data should not fail, received %v", err)
	}
}

func TestCrystallizedState(t *testing.T) {
	if !reflect.DeepEqual(NewCrystallizedState(nil), &CrystallizedState{}) {
		t.Errorf("Crystallized state mismatch, want %v, received %v", NewCrystallizedState(nil), &CrystallizedState{})
	}
	_, crystallized, err := NewGenesisStates()
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}
	emptyCrystallized := &CrystallizedState{}
	if _, err := emptyCrystallized.Marshal(); err == nil {
		t.Error("marshal with empty data should fail")
	}
	if _, err := emptyCrystallized.Hash(); err == nil {
		t.Error("hash with empty data should fail")
	}
	if _, err := crystallized.Hash(); err != nil {
		t.Errorf("hashing with data should not fail, received %v", err)
	}
	if !reflect.DeepEqual(crystallized.data, crystallized.Proto()) {
		t.Errorf("inner crystallized state data did not match proto: received %v, wanted %v", crystallized.Proto(), crystallized.data)
	}

	crystallized.SetStateRecalc(5)
	if crystallized.LastStateRecalc() != 5 {
		t.Errorf("mistmatched last state recalc: wanted 5, received %v", crystallized.LastStateRecalc())
	}

	crystallized.data.JustifiedStreak = 1
	if crystallized.JustifiedStreak() != 1 {
		t.Errorf("mistmatched streak: wanted 1, received %v", crystallized.JustifiedStreak())
	}
	crystallized.ClearJustifiedStreak()
	if crystallized.JustifiedStreak() != 0 {
		t.Errorf("mistmatched streak: wanted 0, received %v", crystallized.JustifiedStreak())
	}
	if crystallized.CrosslinkingStartShard() != 0 {
		t.Errorf("mistmatched streak: wanted 0, received %v", crystallized.JustifiedStreak())
	}
	crystallized.SetLastJustifiedSlot(5)
	if crystallized.LastJustifiedSlot() != 5 {
		t.Errorf("mistmatched justified slot: wanted 5, received %v", crystallized.LastJustifiedSlot())
	}
	crystallized.SetLastFinalizedSlot(5)
	if crystallized.LastFinalizedSlot() != 5 {
		t.Errorf("mistmatched finalized slot: wanted 5, received %v", crystallized.LastFinalizedSlot())
	}
	crystallized.IncrementCurrentDynasty()
	if crystallized.CurrentDynasty() != 2 {
		t.Errorf("mistmatched current dynasty: wanted 2, received %v", crystallized.CurrentDynasty())
	}
	crystallized.SetTotalDeposits(1000)
	if crystallized.TotalDeposits() != 1000 {
		t.Errorf("mistmatched total deposits: wanted 1000, received %v", crystallized.TotalDeposits())
	}
	crystallized.data.DynastySeedLastReset = 1000
	if crystallized.DynastySeedLastReset() != 1000 {
		t.Errorf("mistmatched total deposits: wanted 1000, received %v", crystallized.DynastySeedLastReset())
	}
	crystallized.DynastySeed()

	validators := []*pb.ValidatorRecord{}
	crystallized.SetValidators(validators)
	if crystallized.ValidatorsLength() > 0 {
		t.Errorf("wanted 0 validators, received %v", crystallized.ValidatorsLength())
	}
	if !reflect.DeepEqual(crystallized.Validators(), validators) {
		t.Errorf("mismatched validator set: wanted %v, received %v", validators, crystallized.Validators())
	}
	crystallized.ClearIndicesForSlots()
	if !reflect.DeepEqual(crystallized.IndicesForSlots(), []*pb.ShardAndCommitteeArray{}) {
		t.Errorf("mismatched indices for heights: wanted %v, received %v", []*pb.ShardAndCommitteeArray{}, crystallized.IndicesForSlots())
	}
	crystallized.CrosslinkRecords()
	crystallized.UpdateJustifiedSlot(6)
	if crystallized.LastFinalizedSlot() != 5 {
		t.Errorf("mistmatched finalized slot: wanted 5, received %v", crystallized.LastFinalizedSlot())
	}
}

func TestBlockHashForSlot(t *testing.T) {
	var recentBlockHash [][]byte
	for i := 0; i < 256; i++ {
		recentBlockHash = append(recentBlockHash, []byte{byte(i)})
	}
	state := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHash,
	}, nil)
	block := newTestBlock(t, &pb.BeaconBlock{SlotNumber: 7})
	if _, err := state.BlockHashForSlot(200, block); err == nil {
		t.Error("getBlockHash should have failed with invalid height")
	}
	hash, err := state.BlockHashForSlot(0, block)
	if err != nil {
		t.Errorf("getBlockHash failed: %v", err)
	}
	if bytes.Equal(hash, []byte{'A'}) {
		t.Errorf("getBlockHash returns hash should be A, got: %v", hash)
	}
	hash, err = state.BlockHashForSlot(5, block)
	if err != nil {
		t.Errorf("getBlockHash failed: %v", err)
	}
	if bytes.Equal(hash, []byte{'F'}) {
		t.Errorf("getBlockHash returns hash should be F, got: %v", hash)
	}
	block = newTestBlock(t, &pb.BeaconBlock{SlotNumber: 201})
	hash, err = state.BlockHashForSlot(200, block)
	if err != nil {
		t.Errorf("getBlockHash failed: %v", err)
	}
	if hash[len(hash)-1] != 127 {
		t.Errorf("getBlockHash returns hash should be 127, got: %v", hash)
	}

}

// newTestBlock is a helper method to create blocks with valid defaults.
// For a generic block, use NewBlock(t, nil).
func newTestBlock(t *testing.T, b *pb.BeaconBlock) *Block {
	if b == nil {
		b = &pb.BeaconBlock{}
	}
	if b.ActiveStateHash == nil {
		b.ActiveStateHash = make([]byte, 32)
	}
	if b.CrystallizedStateHash == nil {
		b.CrystallizedStateHash = make([]byte, 32)
	}
	if b.ParentHash == nil {
		b.ParentHash = make([]byte, 32)
	}

	return NewBlock(b)
}
