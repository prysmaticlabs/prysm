package depositcache

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

var _ = PendingDepositsFetcher(&DepositCache{})

func TestInsertPendingDeposit_OK(t *testing.T) {
	dc := DepositCache{}
	dc.InsertPendingDeposit(context.Background(), &ethpb.Deposit{}, big.NewInt(111), 100, [32]byte{})

	if len(dc.pendingDeposits) != 1 {
		t.Error("Deposit not inserted")
	}
}

func TestInsertPendingDeposit_ignoresNilDeposit(t *testing.T) {
	dc := DepositCache{}
	dc.InsertPendingDeposit(context.Background(), nil /*deposit*/, nil /*blockNum*/, 0, [32]byte{})

	if len(dc.pendingDeposits) > 0 {
		t.Error("Unexpected deposit insertion")
	}
}

func TestRemovePendingDeposit_OK(t *testing.T) {
	db := DepositCache{}
	depToRemove := &ethpb.Deposit{Proof: [][]byte{[]byte("A")}}
	otherDep := &ethpb.Deposit{Proof: [][]byte{[]byte("B")}}
	db.pendingDeposits = []*DepositContainer{
		{Deposit: depToRemove, Index: 1},
		{Deposit: otherDep, Index: 5},
	}
	db.RemovePendingDeposit(context.Background(), depToRemove)

	if len(db.pendingDeposits) != 1 || !proto.Equal(db.pendingDeposits[0].Deposit, otherDep) {
		t.Error("Failed to remove deposit")
	}
}

func TestRemovePendingDeposit_IgnoresNilDeposit(t *testing.T) {
	dc := DepositCache{}
	dc.pendingDeposits = []*DepositContainer{{Deposit: &ethpb.Deposit{}}}
	dc.RemovePendingDeposit(context.Background(), nil /*deposit*/)
	if len(dc.pendingDeposits) != 1 {
		t.Errorf("Deposit unexpectedly removed")
	}
}

func TestPendingDeposit_RoundTrip(t *testing.T) {
	dc := DepositCache{}
	dep := &ethpb.Deposit{Proof: [][]byte{[]byte("A")}}
	dc.InsertPendingDeposit(context.Background(), dep, big.NewInt(111), 100, [32]byte{})
	dc.RemovePendingDeposit(context.Background(), dep)
	if len(dc.pendingDeposits) != 0 {
		t.Error("Failed to insert & delete a pending deposit")
	}
}

func TestPendingDeposits_OK(t *testing.T) {
	dc := DepositCache{}

	dc.pendingDeposits = []*DepositContainer{
		{Block: big.NewInt(2), Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("A")}}},
		{Block: big.NewInt(4), Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("B")}}},
		{Block: big.NewInt(6), Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("c")}}},
	}

	deposits := dc.PendingDeposits(context.Background(), big.NewInt(4))
	expected := []*ethpb.Deposit{
		{Proof: [][]byte{[]byte("A")}},
		{Proof: [][]byte{[]byte("B")}},
	}

	if !reflect.DeepEqual(deposits, expected) {
		t.Errorf("Unexpected deposits. got=%+v want=%+v", deposits, expected)
	}

	all := dc.PendingDeposits(context.Background(), nil)
	if len(all) != len(dc.pendingDeposits) {
		t.Error("PendingDeposits(ctx, nil) did not return all deposits")
	}
}

func TestPrunePendingDeposits_ZeroMerkleIndex(t *testing.T) {
	dc := DepositCache{}

	dc.pendingDeposits = []*DepositContainer{
		{Block: big.NewInt(2), Index: 2},
		{Block: big.NewInt(4), Index: 4},
		{Block: big.NewInt(6), Index: 6},
		{Block: big.NewInt(8), Index: 8},
		{Block: big.NewInt(10), Index: 10},
		{Block: big.NewInt(12), Index: 12},
	}

	dc.PrunePendingDeposits(context.Background(), 0)
	expected := []*DepositContainer{
		{Block: big.NewInt(2), Index: 2},
		{Block: big.NewInt(4), Index: 4},
		{Block: big.NewInt(6), Index: 6},
		{Block: big.NewInt(8), Index: 8},
		{Block: big.NewInt(10), Index: 10},
		{Block: big.NewInt(12), Index: 12},
	}

	if !reflect.DeepEqual(dc.pendingDeposits, expected) {
		t.Errorf("Unexpected deposits. got=%+v want=%+v", dc.pendingDeposits, expected)
	}
}

func TestPrunePendingDeposits_OK(t *testing.T) {
	dc := DepositCache{}

	dc.pendingDeposits = []*DepositContainer{
		{Block: big.NewInt(2), Index: 2},
		{Block: big.NewInt(4), Index: 4},
		{Block: big.NewInt(6), Index: 6},
		{Block: big.NewInt(8), Index: 8},
		{Block: big.NewInt(10), Index: 10},
		{Block: big.NewInt(12), Index: 12},
	}

	dc.PrunePendingDeposits(context.Background(), 6)
	expected := []*DepositContainer{
		{Block: big.NewInt(6), Index: 6},
		{Block: big.NewInt(8), Index: 8},
		{Block: big.NewInt(10), Index: 10},
		{Block: big.NewInt(12), Index: 12},
	}

	if !reflect.DeepEqual(dc.pendingDeposits, expected) {
		t.Errorf("Unexpected deposits. got=%+v want=%+v", dc.pendingDeposits, expected)
	}

	dc.pendingDeposits = []*DepositContainer{
		{Block: big.NewInt(2), Index: 2},
		{Block: big.NewInt(4), Index: 4},
		{Block: big.NewInt(6), Index: 6},
		{Block: big.NewInt(8), Index: 8},
		{Block: big.NewInt(10), Index: 10},
		{Block: big.NewInt(12), Index: 12},
	}

	dc.PrunePendingDeposits(context.Background(), 10)
	expected = []*DepositContainer{
		{Block: big.NewInt(10), Index: 10},
		{Block: big.NewInt(12), Index: 12},
	}

	if !reflect.DeepEqual(dc.pendingDeposits, expected) {
		t.Errorf("Unexpected deposits. got=%+v want=%+v", dc.pendingDeposits, expected)
	}

}
