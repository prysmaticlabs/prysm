package db

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestInsertPendingDeposit_OK(t *testing.T) {
	db := BeaconDB{}
	db.InsertPendingDeposit(context.Background(), &pb.Deposit{}, big.NewInt(111))

	if len(db.pendingDeposits) != 1 {
		t.Error("Deposit not inserted")
	}
}

func TestInsertPendingDeposit_ignoresNilDeposit(t *testing.T) {
	db := BeaconDB{}
	db.InsertPendingDeposit(context.Background(), nil /*deposit*/, nil /*blockNum*/)

	if len(db.pendingDeposits) > 0 {
		t.Error("Unexpected deposit insertion")
	}
}

func TestRemovePendingDeposit_OK(t *testing.T) {
	db := BeaconDB{}
	depToRemove := &pb.Deposit{MerkleTreeIndex: 1}
	otherDep := &pb.Deposit{MerkleTreeIndex: 5}
	db.pendingDeposits = []*depositContainer{
		{deposit: depToRemove},
		{deposit: otherDep},
	}
	db.RemovePendingDeposit(context.Background(), depToRemove)

	if len(db.pendingDeposits) != 1 || !proto.Equal(db.pendingDeposits[0].deposit, otherDep) {
		t.Error("Failed to remove deposit")
	}
}

func TestRemovePendingDeposit_IgnoresNilDeposit(t *testing.T) {
	db := BeaconDB{}
	db.pendingDeposits = []*depositContainer{{deposit: &pb.Deposit{}}}
	db.RemovePendingDeposit(context.Background(), nil /*deposit*/)
	if len(db.pendingDeposits) != 1 {
		t.Errorf("Deposit unexpectedly removed")
	}
}

func TestPendingDeposit_RoundTrip(t *testing.T) {
	db := BeaconDB{}
	dep := &pb.Deposit{MerkleTreeIndex: 123}
	db.InsertPendingDeposit(context.Background(), dep, big.NewInt(111))
	db.RemovePendingDeposit(context.Background(), dep)
	if len(db.pendingDeposits) != 0 {
		t.Error("Failed to insert & delete a pending deposit")
	}
}

func TestPendingDeposits_OK(t *testing.T) {
	db := BeaconDB{}

	db.pendingDeposits = []*depositContainer{
		{block: big.NewInt(2), deposit: &pb.Deposit{MerkleTreeIndex: 2}},
		{block: big.NewInt(4), deposit: &pb.Deposit{MerkleTreeIndex: 4}},
		{block: big.NewInt(6), deposit: &pb.Deposit{MerkleTreeIndex: 6}},
	}

	deposits := db.PendingDeposits(context.Background(), big.NewInt(4))
	expected := []*pb.Deposit{
		{MerkleTreeIndex: 2},
		{MerkleTreeIndex: 4},
	}

	if !reflect.DeepEqual(deposits, expected) {
		t.Errorf("Unexpected deposits. got=%+v want=%+v", deposits, expected)
	}

	all := db.PendingDeposits(context.Background(), nil)
	if len(all) != len(db.pendingDeposits) {
		t.Error("PendingDeposits(ctx, nil) did not return all deposits")
	}
}

func TestPrunePendingDeposits_ZeroMerkleIndex(t *testing.T) {
	db := BeaconDB{}

	db.pendingDeposits = []*depositContainer{
		{block: big.NewInt(2), deposit: &pb.Deposit{MerkleTreeIndex: 2}},
		{block: big.NewInt(4), deposit: &pb.Deposit{MerkleTreeIndex: 4}},
		{block: big.NewInt(6), deposit: &pb.Deposit{MerkleTreeIndex: 6}},
		{block: big.NewInt(8), deposit: &pb.Deposit{MerkleTreeIndex: 8}},
		{block: big.NewInt(10), deposit: &pb.Deposit{MerkleTreeIndex: 10}},
		{block: big.NewInt(12), deposit: &pb.Deposit{MerkleTreeIndex: 12}},
	}

	db.PrunePendingDeposits(context.Background(), 0)
	expected := []*depositContainer{
		{block: big.NewInt(2), deposit: &pb.Deposit{MerkleTreeIndex: 2}},
		{block: big.NewInt(4), deposit: &pb.Deposit{MerkleTreeIndex: 4}},
		{block: big.NewInt(6), deposit: &pb.Deposit{MerkleTreeIndex: 6}},
		{block: big.NewInt(8), deposit: &pb.Deposit{MerkleTreeIndex: 8}},
		{block: big.NewInt(10), deposit: &pb.Deposit{MerkleTreeIndex: 10}},
		{block: big.NewInt(12), deposit: &pb.Deposit{MerkleTreeIndex: 12}},
	}

	if !reflect.DeepEqual(db.pendingDeposits, expected) {
		t.Errorf("Unexpected deposits. got=%+v want=%+v", db.pendingDeposits, expected)
	}
}

func TestPrunePendingDeposits_OK(t *testing.T) {
	db := BeaconDB{}

	db.pendingDeposits = []*depositContainer{
		{block: big.NewInt(2), deposit: &pb.Deposit{MerkleTreeIndex: 2}},
		{block: big.NewInt(4), deposit: &pb.Deposit{MerkleTreeIndex: 4}},
		{block: big.NewInt(6), deposit: &pb.Deposit{MerkleTreeIndex: 6}},
		{block: big.NewInt(8), deposit: &pb.Deposit{MerkleTreeIndex: 8}},
		{block: big.NewInt(10), deposit: &pb.Deposit{MerkleTreeIndex: 10}},
		{block: big.NewInt(12), deposit: &pb.Deposit{MerkleTreeIndex: 12}},
	}

	db.PrunePendingDeposits(context.Background(), 6)
	expected := []*depositContainer{
		{block: big.NewInt(6), deposit: &pb.Deposit{MerkleTreeIndex: 6}},
		{block: big.NewInt(8), deposit: &pb.Deposit{MerkleTreeIndex: 8}},
		{block: big.NewInt(10), deposit: &pb.Deposit{MerkleTreeIndex: 10}},
		{block: big.NewInt(12), deposit: &pb.Deposit{MerkleTreeIndex: 12}},
	}

	if !reflect.DeepEqual(db.pendingDeposits, expected) {
		t.Errorf("Unexpected deposits. got=%+v want=%+v", db.pendingDeposits, expected)
	}

	db.pendingDeposits = []*depositContainer{
		{block: big.NewInt(2), deposit: &pb.Deposit{MerkleTreeIndex: 2}},
		{block: big.NewInt(4), deposit: &pb.Deposit{MerkleTreeIndex: 4}},
		{block: big.NewInt(6), deposit: &pb.Deposit{MerkleTreeIndex: 6}},
		{block: big.NewInt(8), deposit: &pb.Deposit{MerkleTreeIndex: 8}},
		{block: big.NewInt(10), deposit: &pb.Deposit{MerkleTreeIndex: 10}},
		{block: big.NewInt(12), deposit: &pb.Deposit{MerkleTreeIndex: 12}},
	}

	db.PrunePendingDeposits(context.Background(), 10)
	expected = []*depositContainer{
		{block: big.NewInt(10), deposit: &pb.Deposit{MerkleTreeIndex: 10}},
		{block: big.NewInt(12), deposit: &pb.Deposit{MerkleTreeIndex: 12}},
	}

	if !reflect.DeepEqual(db.pendingDeposits, expected) {
		t.Errorf("Unexpected deposits. got=%+v want=%+v", db.pendingDeposits, expected)
	}

}
