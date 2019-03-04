package db

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/trieutil"

	"github.com/ethereum/go-ethereum/common"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func setupPOWDeposiState(t *testing.T) *pb.POWDepositState {
	powDepState := &pb.POWDepositState{}
	(*powDepState).LatestBlockHash = common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c")
	(*powDepState).LastBlockHeight = uint64(7299837)
	(*powDepState).DepositCount = uint64(1000)
	dt := trieutil.NewDepositTrie()
	dt.UpdateDepositTrie([]byte{'A'})
	(*powDepState).DepositTrie = dt.ToProtoDepositTrie()
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
		t.Fatalf("Expected %#x and %#x to be equal", (*powDepState).LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c"))
	}
	if (*powDepState).LastBlockHeight != uint64(7299837) {
		t.Fatalf("Expected %v and %v to be equal", (*powDepState).LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c"))
	}
	if (*powDepState).DepositCount != uint64(1000) {
		t.Fatalf("Expected %v and %v to be equal", (*powDepState).LatestBlockHash, common.FromHex("0xaff17b5759cc71f5fb21e595b76abbb5dc27f8691fd228e291f58e5976d6813c"))
	}
	if (*powDepState).DepositTrie.DepositCount != 1 {
		t.Fatalf("Expected %v and %v to be equal", (*powDepState).DepositTrie.DepositCount, 1)
	}
}
