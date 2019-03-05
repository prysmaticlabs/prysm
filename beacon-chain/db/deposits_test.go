package db

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func setupPOWDeposiState(t *testing.T) *pb.POWDepositState {
	powDepState := &pb.POWDepositState{}
	powDepState.LatestBlockHash = common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c")
	powDepState.LastBlockHeight = uint64(7299837)
	dt := trieutil.NewDepositTrie()
	dt.UpdateDepositTrie([]byte{'A'})
	powDepState.DepositTrie = dt.GetTrie()
	return powDepState
}

func TestPOWDeposiState_OK(t *testing.T) {
	db, err := SetupDB()
	defer teardownDB(t, db)
	db.SaveDepositState(setupPOWDeposiState(t))
	powDepState, err := db.DepositState()
	if err != nil {
		t.Fatalf("Failed to read pow deposit state drom db: %v", err)
	}

	if !bytes.Equal((*powDepState).LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c")) {
		t.Fatalf("Expected %#x and %#x to be equal", powDepState.LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c"))
	}
	if powDepState.LastBlockHeight != uint64(7299837) {
		t.Fatalf("Expected %v and %v to be equal", powDepState.LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c"))
	}
	if powDepState.DepositTrie.DepositCount != 1 {
		t.Fatalf("Expected %v and %v to be equal", powDepState.DepositTrie.DepositCount, 1)
	}
}

func TestResetDeposiState_OK(t *testing.T) {
	db, err := SetupDB()
	defer teardownDB(t, db)
	db.SaveDepositState(setupPOWDeposiState(t))
	powDepState, err := db.DepositState()
	if err != nil {
		t.Fatalf("Failed to read pow deposit state drom db: %v", err)
	}
	if !bytes.Equal((*powDepState).LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c")) {
		t.Fatalf("Expected %#x and %#x to be equal", (*powDepState).LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c"))
	}
	if err = db.ResetDepositState(); err != nil {
		t.Fatalf("ResetDepositState returned an error: %v", err)
	}
	powDepState, err = db.DepositState()
	if err != nil {
		t.Fatalf("Failed to read pow deposit state drom db: %v", err)
	}
	if powDepState.LastBlockHeight != 0 {
		t.Fatalf("Expected %v and %v to be equal", powDepState.LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c"))
	}

	if powDepState.DepositTrie != nil {
		t.Fatalf("Expected %v and %v to be equal", powDepState.DepositTrie.DepositCount, 0)
	}
}
