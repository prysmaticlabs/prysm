package depositcache

import (
	"context"
	"math/big"
	"testing"

	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"google.golang.org/protobuf/proto"
)

var _ PendingDepositsFetcher = (*DepositCache)(nil)

func TestInsertPendingDeposit_OK(t *testing.T) {
	dc := DepositCache{}
	dc.InsertPendingDeposit(context.Background(), &ethpb.Deposit{}, 111, 100, [32]byte{})

	assert.Equal(t, 1, len(dc.pendingDeposits), "Deposit not inserted")
}

func TestInsertPendingDeposit_ignoresNilDeposit(t *testing.T) {
	dc := DepositCache{}
	dc.InsertPendingDeposit(context.Background(), nil /*deposit*/, 0 /*blockNum*/, 0, [32]byte{})

	assert.Equal(t, 0, len(dc.pendingDeposits))
}

func TestRemovePendingDeposit_OK(t *testing.T) {
	db := DepositCache{}
	proof1 := makeDepositProof()
	proof1[0] = bytesutil.PadTo([]byte{'A'}, 32)
	proof2 := makeDepositProof()
	proof2[0] = bytesutil.PadTo([]byte{'A'}, 32)
	data := &ethpb.Deposit_Data{
		PublicKey:             make([]byte, 48),
		WithdrawalCredentials: make([]byte, 32),
		Amount:                0,
		Signature:             make([]byte, 96),
	}
	depToRemove := &ethpb.Deposit{Proof: proof1, Data: data}
	otherDep := &ethpb.Deposit{Proof: proof2, Data: data}
	db.pendingDeposits = []*ethpb.DepositContainer{
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
	dc.pendingDeposits = []*ethpb.DepositContainer{{Deposit: &ethpb.Deposit{}}}
	dc.RemovePendingDeposit(context.Background(), nil /*deposit*/)
	assert.Equal(t, 1, len(dc.pendingDeposits), "Deposit unexpectedly removed")
}

func TestPendingDeposit_RoundTrip(t *testing.T) {
	dc := DepositCache{}
	proof := makeDepositProof()
	proof[0] = bytesutil.PadTo([]byte{'A'}, 32)
	data := &ethpb.Deposit_Data{
		PublicKey:             make([]byte, 48),
		WithdrawalCredentials: make([]byte, 32),
		Amount:                0,
		Signature:             make([]byte, 96),
	}
	dep := &ethpb.Deposit{Proof: proof, Data: data}
	dc.InsertPendingDeposit(context.Background(), dep, 111, 100, [32]byte{})
	dc.RemovePendingDeposit(context.Background(), dep)
	assert.Equal(t, 0, len(dc.pendingDeposits), "Failed to insert & delete a pending deposit")
}

func TestPendingDeposits_OK(t *testing.T) {
	dc := DepositCache{}

	dc.pendingDeposits = []*ethpb.DepositContainer{
		{Eth1BlockHeight: 2, Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("A")}}},
		{Eth1BlockHeight: 4, Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("B")}}},
		{Eth1BlockHeight: 6, Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("c")}}},
	}

	deposits := dc.PendingDeposits(context.Background(), big.NewInt(4))
	expected := []*ethpb.Deposit{
		{Proof: [][]byte{[]byte("A")}},
		{Proof: [][]byte{[]byte("B")}},
	}
	assert.DeepSSZEqual(t, expected, deposits)

	all := dc.PendingDeposits(context.Background(), nil)
	assert.Equal(t, len(dc.pendingDeposits), len(all), "PendingDeposits(ctx, nil) did not return all deposits")
}

func TestPrunePendingDeposits_ZeroMerkleIndex(t *testing.T) {
	dc := DepositCache{}

	dc.pendingDeposits = []*ethpb.DepositContainer{
		{Eth1BlockHeight: 2, Index: 2},
		{Eth1BlockHeight: 4, Index: 4},
		{Eth1BlockHeight: 6, Index: 6},
		{Eth1BlockHeight: 8, Index: 8},
		{Eth1BlockHeight: 10, Index: 10},
		{Eth1BlockHeight: 12, Index: 12},
	}

	dc.PrunePendingDeposits(context.Background(), 0)
	expected := []*ethpb.DepositContainer{
		{Eth1BlockHeight: 2, Index: 2},
		{Eth1BlockHeight: 4, Index: 4},
		{Eth1BlockHeight: 6, Index: 6},
		{Eth1BlockHeight: 8, Index: 8},
		{Eth1BlockHeight: 10, Index: 10},
		{Eth1BlockHeight: 12, Index: 12},
	}
	assert.DeepEqual(t, expected, dc.pendingDeposits)
}

func TestPrunePendingDeposits_OK(t *testing.T) {
	dc := DepositCache{}

	dc.pendingDeposits = []*ethpb.DepositContainer{
		{Eth1BlockHeight: 2, Index: 2},
		{Eth1BlockHeight: 4, Index: 4},
		{Eth1BlockHeight: 6, Index: 6},
		{Eth1BlockHeight: 8, Index: 8},
		{Eth1BlockHeight: 10, Index: 10},
		{Eth1BlockHeight: 12, Index: 12},
	}

	dc.PrunePendingDeposits(context.Background(), 6)
	expected := []*ethpb.DepositContainer{
		{Eth1BlockHeight: 6, Index: 6},
		{Eth1BlockHeight: 8, Index: 8},
		{Eth1BlockHeight: 10, Index: 10},
		{Eth1BlockHeight: 12, Index: 12},
	}

	assert.DeepEqual(t, expected, dc.pendingDeposits)

	dc.pendingDeposits = []*ethpb.DepositContainer{
		{Eth1BlockHeight: 2, Index: 2},
		{Eth1BlockHeight: 4, Index: 4},
		{Eth1BlockHeight: 6, Index: 6},
		{Eth1BlockHeight: 8, Index: 8},
		{Eth1BlockHeight: 10, Index: 10},
		{Eth1BlockHeight: 12, Index: 12},
	}

	dc.PrunePendingDeposits(context.Background(), 10)
	expected = []*ethpb.DepositContainer{
		{Eth1BlockHeight: 10, Index: 10},
		{Eth1BlockHeight: 12, Index: 12},
	}

	assert.DeepEqual(t, expected, dc.pendingDeposits)
}
