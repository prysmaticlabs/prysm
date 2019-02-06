package blocks

import (
	"bytes"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGenesisBlock(t *testing.T) {
	stateHash := []byte{0}
	b1 := NewGenesisBlock(stateHash)

	if b1.ParentRootHash32 == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if !reflect.DeepEqual(b1.Body.Attestations, []*pb.Attestation{}) {
		t.Errorf("genesis block should have 0 attestations")
	}

	if !bytes.Equal(b1.RandaoRevealHash32, params.BeaconConfig().ZeroHash[:]) {
		t.Error("genesis block missing RandaoRevealHash32 field")
	}

	if !bytes.Equal(b1.StateRootHash32, stateHash) {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}

	rd := []byte{}
	if IsRandaoValid(b1.RandaoRevealHash32, rd) {
		t.Error("RANDAO should be empty")
	}

}

func TestBlockRootAtSlot_OK(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("epochLength should be 64 for these tests to pass")
	}
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
		LatestBlockRootHash32S: blockRoots,
	}

	tests := []struct {
		slot         uint64
		stateSlot    uint64
		expectedRoot []byte
	}{
		{
			slot:         0,
			stateSlot:    1,
			expectedRoot: []byte{0},
		},
		{
			slot:         2,
			stateSlot:    5,
			expectedRoot: []byte{2},
		},
		{
			slot:         64,
			stateSlot:    128,
			expectedRoot: []byte{64},
		}, {
			slot:         2999,
			stateSlot:    3000,
			expectedRoot: []byte{55},
		}, {
			slot:         2873,
			stateSlot:    3000,
			expectedRoot: []byte{57},
		},
	}
	for _, tt := range tests {
		state.Slot = tt.stateSlot
		result, err := BlockRoot(state, tt.slot)
		if err != nil {
			t.Errorf("failed to get block root at slot %d: %v", tt.slot, err)
		}
		if !bytes.Equal(result, tt.expectedRoot) {
			t.Errorf(
				"result block root was an unexpected value. Wanted %d, got %d",
				tt.expectedRoot,
				result,
			)
		}
	}
}

func TestBlockRootAtSlot_OutOfBounds(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("epochLength should be 64 for these tests to pass")
	}

	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
		LatestBlockRootHash32S: blockRoots,
	}

	tests := []struct {
		slot        uint64
		stateSlot   uint64
		expectedErr string
	}{
		{
			slot:        1000,
			stateSlot:   500,
			expectedErr: "slot 1000 is not within expected range of 372 to 499",
		},
		{
			slot:        129,
			stateSlot:   400,
			expectedErr: "slot 129 is not within expected range of 272 to 399",
		},
	}
	for _, tt := range tests {
		state.Slot = tt.stateSlot
		_, err := BlockRoot(state, tt.slot)
		if err != nil && err.Error() != tt.expectedErr {
			t.Errorf("Expected error \"%s\" got \"%v\"", tt.expectedErr, err)
		}
	}
}

func TestProcessBlockRoots(t *testing.T) {
	state := &pb.BeaconState{}

	state.LatestBlockRootHash32S = make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	state.Slot = params.BeaconConfig().LatestBlockRootsLength + 1

	testRoot := [32]byte{'a'}

	newState := ProcessBlockRoots(state, testRoot)
	if !bytes.Equal(newState.LatestBlockRootHash32S[0], testRoot[:]) {
		t.Fatalf("Latest Block root hash not saved."+
			" Supposed to get %#x , but got %#x", testRoot, newState.LatestBlockRootHash32S[0])
	}

	newState.Slot = newState.Slot - 1

	newState = ProcessBlockRoots(newState, testRoot)
	expectedHashes := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	expectedHashes[0] = testRoot[:]
	expectedHashes[params.BeaconConfig().LatestBlockRootsLength-1] = testRoot[:]

	expectedRoot := hashutil.MerkleRoot(expectedHashes)

	if !bytes.Equal(newState.BatchedBlockRootHash32S[0], expectedRoot[:]) {
		t.Errorf("saved merkle root is not equal to expected merkle root"+
			"\n expected %#x but got %#x", expectedRoot, newState.BatchedBlockRootHash32S[0])
	}
}

func TestDecodeDepositAmountAndTimeStamp(t *testing.T) {

	tests := []struct {
		depositData *pb.DepositInput
		amount      uint64
		timestamp   int64
	}{
		{
			depositData: &pb.DepositInput{
				Pubkey:                      []byte("testing"),
				RandaoCommitmentHash32:      []byte("randao"),
				CustodyCommitmentHash32:     []byte("commitment"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    8749343850,
			timestamp: 458739850,
		},
		{
			depositData: &pb.DepositInput{
				Pubkey:                      []byte("testing"),
				CustodyCommitmentHash32:     []byte("commitment"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    657660,
			timestamp: 67750,
		},
		{
			depositData: &pb.DepositInput{
				Pubkey:                      []byte("testing"),
				RandaoCommitmentHash32:      []byte("randao"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    5445540,
			timestamp: 34340,
		}, {
			depositData: &pb.DepositInput{
				RandaoCommitmentHash32:      []byte("randao"),
				CustodyCommitmentHash32:     []byte("commitment"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    4545,
			timestamp: 4343,
		}, {
			depositData: &pb.DepositInput{
				Pubkey:                  []byte("testing"),
				RandaoCommitmentHash32:  []byte("randao"),
				CustodyCommitmentHash32: []byte("commitment"),
			},
			amount:    76706966,
			timestamp: 34394393,
		},
	}

	for _, tt := range tests {
		data, err := EncodeDepositData(tt.depositData, tt.amount, tt.timestamp)
		if err != nil {
			t.Fatalf("could not encode data %v", err)
		}

		decAmount, decTimestamp, err := DecodeDepositAmountAndTimeStamp(data)
		if err != nil {
			t.Fatalf("could not decode data %v", err)
		}

		if tt.amount != decAmount {
			t.Errorf("decoded amount not equal to given amount, %d : %d", decAmount, tt.amount)
		}

		if tt.timestamp != decTimestamp {
			t.Errorf("decoded timestamp not equal to given timestamp, %d : %d", decTimestamp, tt.timestamp)
		}
	}
}

func TestBlockChildren(t *testing.T) {
	genesisBlock := NewGenesisBlock([]byte{})
	genesisHash, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	targets := []*pb.BeaconBlock{
		{
			Slot:             9,
			ParentRootHash32: genesisHash[:],
		},
		{
			Slot:             5,
			ParentRootHash32: []byte{},
		},
		{
			Slot:             8,
			ParentRootHash32: genesisHash[:],
		},
	}
	children, err := BlockChildren(genesisBlock, targets)
	if err != nil {
		t.Fatalf("Could not fetch block children: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("Expected %d children, received %d", 2, len(children))
	}
}
