package state

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestInitialBeaconState_Ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	epochLength := params.BeaconConfig().EpochLength

	if params.BeaconConfig().InitialSlotNumber != 0 {
		t.Error("InitialSlotNumber should be 0 for these tests to pass")
	}
	initialSlotNumber := params.BeaconConfig().InitialSlotNumber

	if params.BeaconConfig().InitialForkVersion != 0 {
		t.Error("InitialForkVersion should be 0 for these tests to pass")
	}
	initialForkVersion := params.BeaconConfig().InitialForkVersion

	if params.BeaconConfig().ZeroHash != [32]byte{} {
		t.Error("ZeroHash should be all 0s for these tests to pass")
	}

	if params.BeaconConfig().LatestRandaoMixesLength != 8192 {
		t.Error("LatestRandaoMixesLength should be 8192 for these tests to pass")
	}
	latestRandaoMixesLength := int(params.BeaconConfig().LatestRandaoMixesLength)
	LatestVdfMixesLength := int(params.BeaconConfig().LatestRandaoMixesLength /
		params.BeaconConfig().EpochLength)

	if params.BeaconConfig().ShardCount != 1024 {
		t.Error("ShardCount should be 1024 for these tests to pass")
	}
	shardCount := int(params.BeaconConfig().ShardCount)

	if params.BeaconConfig().LatestBlockRootsLength != 8192 {
		t.Error("LatestBlockRootsLength should be 8192 for these tests to pass")
	}
	latestBlockRootsLength := int(params.BeaconConfig().LatestBlockRootsLength)

	if params.BeaconConfig().DepositsForChainStart != 16384 {
		t.Error("DepositsForChainStart should be 16384 for these tests to pass")
	}
	depositsForChainStart := int(params.BeaconConfig().DepositsForChainStart)

	genesisTime := uint64(99999)
	processedPowReceiptRoot := []byte{'A', 'B', 'C'}
	maxDeposit := params.BeaconConfig().MaxDepositInGwei
	var deposits []*pb.Deposit
	for i := 0; i < depositsForChainStart; i++ {
		deposits = append(deposits, &pb.Deposit{MerkleBranchHash32S: [][]byte{{1}, {2}, {3}}, MerkleTreeIndex: 0,
			DepositData: &pb.DepositData{Value: maxDeposit, DepositInput: &pb.DepositInput{
				Pubkey: []byte(strconv.Itoa(i)), ProofOfPossession: []byte{'B'},
				WithdrawalCredentialsHash32: []byte{'C'}, RandaoCommitmentHash32: []byte{'D'},
				PocCommitment: []byte{'D'},
			}}})
	}

	state, err := InitialBeaconState(
		deposits,
		genesisTime,
		processedPowReceiptRoot)
	if err != nil {
		t.Fatalf("could not execute InitialBeaconState: %v", err)
	}

	// Misc fields checks.
	if state.Slot != initialSlotNumber {
		t.Error("Slot was not correctly initialized")
	}
	if state.GenesisTime != genesisTime {
		t.Error("GenesisTime was not correctly initialized")
	}
	if !reflect.DeepEqual(*state.ForkData, pb.ForkData{
		PreForkVersion:  initialForkVersion,
		PostForkVersion: initialForkVersion,
		ForkSlot:        initialSlotNumber,
	}) {
		t.Error("ForkData was not correctly initialized")
	}

	// Validator registry fields checks.
	if state.ValidatorRegistryLastChangeSlot != initialSlotNumber {
		t.Error("ValidatorRegistryLastChangeSlot was not correctly initialized")
	}
	if state.ValidatorRegistryExitCount != 0 {
		t.Error("ValidatorRegistryExitCount was not correctly initialized")
	}
	if len(state.ValidatorRegistry) != depositsForChainStart {
		t.Error("ValidatorRegistry was not correctly initialized")
	}
	if len(state.ValidatorBalances) != depositsForChainStart {
		t.Error("ValidatorBalances was not correctly initialized")
	}

	// Randomness and committees fields checks.
	if len(state.LatestRandaoMixesHash32S) != latestRandaoMixesLength {
		t.Error("Length of LatestRandaoMixesHash32S was not correctly initialized")
	}
	if len(state.LatestVdfOutputs) != LatestVdfMixesLength {
		t.Error("Length of LatestRandaoMixesHash32S was not correctly initialized")
	}
	if len(state.PersistentCommittees) != shardCount {
		t.Error("PersistentCommittees was not correctly initialized")
	}
	if !reflect.DeepEqual(state.PersistentCommitteeReassignments, []*pb.ShardReassignmentRecord{}) {
		t.Error("PersistentCommitteeReassignments was not correctly initialized")
	}

	// Proof of custody field check.
	if !reflect.DeepEqual(state.PocChallenges, []*pb.ProofOfCustodyChallenge{}) {
		t.Error("PocChallenges was not correctly initialized")
	}

	// Finality fields checks.
	if state.PreviousJustifiedSlot != initialSlotNumber {
		t.Error("PreviousJustifiedSlot was not correctly initialized")
	}
	if state.JustifiedSlot != initialSlotNumber {
		t.Error("JustifiedSlot was not correctly initialized")
	}
	if state.FinalizedSlot != initialSlotNumber {
		t.Error("FinalizedSlot was not correctly initialized")
	}
	if state.JustificationBitfield != 0 {
		t.Error("JustificationBitfield was not correctly initialized")
	}

	// Recent state checks.
	if len(state.LatestCrosslinks) != shardCount {
		t.Error("Length of LatestCrosslinks was not correctly initialized")
	}
	if len(state.LatestBlockRootHash32S) != latestBlockRootsLength {
		t.Error("Length of LatestBlockRootHash32S was not correctly initialized")
	}
	if !reflect.DeepEqual(state.LatestPenalizedExitBalances, []uint64{}) {
		t.Error("LatestPenalizedExitBalances was not correctly initialized")
	}
	if !reflect.DeepEqual(state.LatestAttestations, []*pb.PendingAttestationRecord{}) {
		t.Error("LatestAttestations was not correctly initialized")
	}
	if !reflect.DeepEqual(state.BatchedBlockRootHash32S, [][]byte{}) {
		t.Error("BatchedBlockRootHash32S was not correctly initialized")
	}

	// PoW receipt root checks.
	if !bytes.Equal(state.ProcessedPowReceiptRootHash32, processedPowReceiptRoot) {
		t.Error("ProcessedPowReceiptRootHash32 was not correctly initialized")
	}
	if !reflect.DeepEqual(state.CandidatePowReceiptRoots, []*pb.CandidatePoWReceiptRootRecord{}) {
		t.Error("CandidatePowReceiptRoots was not correctly initialized")
	}

	// Initial committee shuffling check.
	if len(state.ShardAndCommitteesAtSlots) != int(2*epochLength) {
		t.Error("ShardAndCommitteesAtSlots was not correctly initialized")
	}

	for i := 0; i < len(state.ShardAndCommitteesAtSlots); i++ {
		if len(state.ShardAndCommitteesAtSlots[i].ArrayShardAndCommittee[0].Committee) !=
			int(params.BeaconConfig().TargetCommitteeSize) {
			t.Error("ShardAndCommittees was not correctly initialized")
		}
	}
}

func TestGenesisState_HashEquality(t *testing.T) {
	state1, _ := NewGenesisBeaconState(nil)
	state2, _ := NewGenesisBeaconState(nil)

	enc1, err1 := proto.Marshal(state1)
	enc2, err2 := proto.Marshal(state2)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to marshal state to bytes: %v %v", err1, err2)
	}

	h1 := hashutil.Hash(enc1)
	h2 := hashutil.Hash(enc2)
	if h1 != h2 {
		t.Fatalf("Hash of two genesis states should be equal: %#x", h1)
	}
}

func TestGenesisState_InitializesLatestBlockHashes(t *testing.T) {
	s, _ := NewGenesisBeaconState(nil)
	want, got := len(s.GetLatestBlockRootHash32S()), 2*int(params.BeaconConfig().CycleLength)
	if want != got {
		t.Errorf("Wrong number of recent block hashes. Got: %d Want: %d", got, want)
	}

	want = cap(s.GetLatestBlockRootHash32S())
	if want != got {
		t.Errorf("The slice underlying array capacity is wrong. Got: %d Want: %d", got, want)
	}

	zero := make([]byte, 0, 32)
	for _, h := range s.GetLatestBlockRootHash32S() {
		if !bytes.Equal(h, zero) {
			t.Errorf("Unexpected non-zero hash data: %v", h)
		}
	}
}

func TestUpdateAttestationsAfterRecalc(t *testing.T) {
	state, _ := NewGenesisBeaconState(nil)
	newAttestations := []*pb.PendingAttestationRecord{
		{
			Data: &pb.AttestationData{
				Slot:  10,
				Shard: 2,
			},
		},
		{
			Data: &pb.AttestationData{
				Slot:  9,
				Shard: 3,
			},
		},
	}

	state.LatestAttestations = newAttestations
	newAttestations = ClearAttestations(state, 8)
	if len(newAttestations) != 2 {
		t.Fatalf("Updated attestations should be length 2: %d", len(newAttestations))
	}
}

func TestUpdateLatestBlockHashes(t *testing.T) {
	block := &pb.BeaconBlock{
		Slot:             10,
		ParentRootHash32: []byte{'A'},
	}

	recentBlockHashes := make([][]byte, 2*int(params.BeaconConfig().CycleLength))
	for i := 0; i < 2*int(params.BeaconConfig().CycleLength); i++ {
		recentBlockHashes[i] = []byte{0}
	}

	state := &pb.BeaconState{
		LatestBlockRootHash32S: recentBlockHashes,
	}

	updated, err := CalculateNewBlockHashes(state, block, 0)
	if err != nil {
		t.Fatalf("failed to update recent blockhashes: %v", err)
	}

	if len(updated) != 2*int(params.BeaconConfig().CycleLength) {
		t.Fatalf("length of updated recent blockhashes should be %d: found %d", params.BeaconConfig().CycleLength, len(updated))
	}

	for i := 0; i < len(updated); i++ {
		if i < len(updated)-10 {
			if !bytes.Equal(updated[i], []byte{0}) {
				t.Fatalf("update failed: expected %#x got %#x", []byte{0}, updated[i])
			}
		} else if !bytes.Equal(updated[i], block.GetParentRootHash32()) {
			t.Fatalf("update failed: expected %#x got %#x", block.GetParentRootHash32(), updated[i])
		}
	}
}

func TestCalculateNewBlockHashes_DoesNotMutateData(t *testing.T) {
	interestingData := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("block"),
		[]byte("hash"),
	}

	s, _ := NewGenesisBeaconState(nil)
	copy(s.LatestBlockRootHash32S, interestingData)
	original := make([][]byte, 2*params.BeaconConfig().CycleLength)
	copy(original, s.LatestBlockRootHash32S)

	if !reflect.DeepEqual(s.GetLatestBlockRootHash32S(), original) {
		t.Fatal("setup data should be equal!")
	}

	block := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: []byte{},
	}

	result, _ := CalculateNewBlockHashes(s, block, 0 /*parentSlot*/)

	if !reflect.DeepEqual(s.GetLatestBlockRootHash32S(), original) {
		t.Error("data has mutated from the original")
	}

	if reflect.DeepEqual(result, original) {
		t.Error("the resulting data did not change from the original")
	}
}
