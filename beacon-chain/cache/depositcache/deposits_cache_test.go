package depositcache

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

const nilDepositErr = "Ignoring nil deposit insertion"

var _ = DepositFetcher(&DepositCache{})

func TestInsertDeposit_LogsOnNilDepositInsertion(t *testing.T) {
	hook := logTest.NewGlobal()
	dc, err := NewDepositCache()
	require.NoError(t, err)

	dc.InsertDeposit(context.Background(), nil, 1, 0, [32]byte{})

	if len(dc.deposits) != 0 {
		t.Fatal("Number of deposits changed")
	}
	if hook.LastEntry().Message != nilDepositErr {
		t.Errorf("Did not log correct message, wanted \"Ignoring nil deposit insertion\", got \"%s\"", hook.LastEntry().Message)
	}
}

func TestInsertDeposit_MaintainsSortedOrderByIndex(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	insertions := []struct {
		blkNum  uint64
		deposit *ethpb.Deposit
		index   int64
	}{
		{
			blkNum:  0,
			deposit: &ethpb.Deposit{},
			index:   0,
		},
		{
			blkNum:  0,
			deposit: &ethpb.Deposit{},
			index:   3,
		},
		{
			blkNum:  0,
			deposit: &ethpb.Deposit{},
			index:   1,
		},
		{
			blkNum:  0,
			deposit: &ethpb.Deposit{},
			index:   4,
		},
	}

	for _, ins := range insertions {
		dc.InsertDeposit(context.Background(), ins.deposit, ins.blkNum, ins.index, [32]byte{})
	}

	expectedIndices := []int64{0, 1, 3, 4}
	for i, ei := range expectedIndices {
		if dc.deposits[i].Index != ei {
			t.Errorf("dc.deposits[%d].Index = %d, wanted %d", i, dc.deposits[i].Index, ei)
		}
	}
}

func TestAllDeposits_ReturnsAllDeposits(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	deposits := []*dbpb.DepositContainer{
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 11,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 11,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 12,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 12,
			Deposit:         &ethpb.Deposit{},
		},
	}
	dc.deposits = deposits

	d := dc.AllDeposits(context.Background(), nil)
	if len(d) != len(deposits) {
		t.Errorf("Return the wrong number of deposits (%d) wanted %d", len(d), len(deposits))
	}
}

func TestAllDeposits_FiltersDepositUpToAndIncludingBlockNumber(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	deposits := []*dbpb.DepositContainer{
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 11,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 11,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 12,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 12,
			Deposit:         &ethpb.Deposit{},
		},
	}
	dc.deposits = deposits

	d := dc.AllDeposits(context.Background(), big.NewInt(11))
	expected := 5
	if len(d) != expected {
		t.Errorf("Return the wrong number of deposits (%d) wanted %d", len(d), expected)
	}
}

func TestDepositsNumberAndRootAtHeight_ReturnsAppropriateCountAndRoot(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	dc.deposits = []*dbpb.DepositContainer{
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 11,
			Deposit:         &ethpb.Deposit{},
			DepositRoot:     []byte("root"),
		},
		{
			Eth1BlockHeight: 12,
			Deposit:         &ethpb.Deposit{},
		},
		{
			Eth1BlockHeight: 12,
			Deposit:         &ethpb.Deposit{},
		},
	}

	n, root := dc.DepositsNumberAndRootAtHeight(context.Background(), big.NewInt(11))
	if int(n) != 5 {
		t.Errorf("Returned unexpected deposits number %d wanted %d", n, 5)
	}

	if root != bytesutil.ToBytes32([]byte("root")) {
		t.Errorf("Returned unexpected root: %v", root)
	}
}

func TestDepositsNumberAndRootAtHeight_ReturnsEmptyTrieIfBlockHeightLessThanOldestDeposit(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	dc.deposits = []*dbpb.DepositContainer{
		{
			Eth1BlockHeight: 10,
			Deposit:         &ethpb.Deposit{},
			DepositRoot:     []byte("root"),
		},
		{
			Eth1BlockHeight: 11,
			Deposit:         &ethpb.Deposit{},
			DepositRoot:     []byte("root"),
		},
	}

	n, root := dc.DepositsNumberAndRootAtHeight(context.Background(), big.NewInt(2))
	if int(n) != 0 {
		t.Errorf("Returned unexpected deposits number %d wanted %d", n, 0)
	}

	if root != [32]byte{} {
		t.Errorf("Returned unexpected root: %v", root)
	}
}

func TestDepositByPubkey_ReturnsFirstMatchingDeposit(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	dc.deposits = []*dbpb.DepositContainer{
		{
			Eth1BlockHeight: 9,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte("pk0"),
				},
			},
		},
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte("pk1"),
				},
			},
		},
		{
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte("pk1"),
				},
			},
		},
		{
			Eth1BlockHeight: 12,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte("pk2"),
				},
			},
		},
	}

	dep, blkNum := dc.DepositByPubkey(context.Background(), []byte("pk1"))

	if !bytes.Equal(dep.Data.PublicKey, []byte("pk1")) {
		t.Error("Returned wrong deposit")
	}
	if blkNum.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("Returned wrong block number %v", blkNum)
	}
}

func TestFinalizedDeposits_DepositsCachedCorrectly(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	finalizedDeposits := []*dbpb.DepositContainer{
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{0},
				},
			},
			Index: 0,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{1},
				},
			},
			Index: 1,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{2},
				},
			},
			Index: 2,
		},
	}
	dc.deposits = append(finalizedDeposits, &dbpb.DepositContainer{
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey: []byte{3},
			},
		},
		Index: 3,
	})

	dc.InsertFinalizedDeposits(context.Background(), 2)

	cachedDeposits := dc.FinalizedDeposits(context.Background())
	if cachedDeposits == nil {
		t.Fatalf("Deposits not cached")
	}
	if cachedDeposits.MerkleTrieIndex != 2 {
		t.Errorf("Incorrect index of last deposit (%d) vs expected 2", cachedDeposits.MerkleTrieIndex)
	}

	var deps [][]byte
	for _, d := range finalizedDeposits {
		hash, err := ssz.HashTreeRoot(d.Deposit.Data)
		if err != nil {
			t.Fatalf("Could not hash deposit data")
		}
		deps = append(deps, hash[:])
	}
	trie, err := trieutil.GenerateTrieFromItems(deps, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate deposit trie")
	}

	actualRoot := cachedDeposits.Deposits.HashTreeRoot()
	expectedRoot := trie.HashTreeRoot()
	if actualRoot != expectedRoot {
		t.Errorf("Incorrect deposit trie root (%x) vs expected %x", actualRoot, expectedRoot)
	}
}

func TestFinalizedDeposits_UtilizesPreviouslyCachedDeposits(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	oldFinalizedDeposits := []*dbpb.DepositContainer{
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{0},
				},
			},
			Index: 0,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{1},
				},
			},
			Index: 1,
		},
	}
	newFinalizedDeposit := dbpb.DepositContainer{
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey: []byte{2},
			},
		},
		Index: 2,
	}
	dc.deposits = oldFinalizedDeposits
	dc.InsertFinalizedDeposits(context.Background(), 1)
	// Artificially exclude old deposits so that they can only be retrieved from previously finalized deposits.
	dc.deposits = []*dbpb.DepositContainer{&newFinalizedDeposit}

	dc.InsertFinalizedDeposits(context.Background(), 2)

	cachedDeposits := dc.FinalizedDeposits(context.Background())
	if cachedDeposits == nil {
		t.Fatalf("Deposits not cached")
	}
	if cachedDeposits.MerkleTrieIndex != 2 {
		t.Errorf("Incorrect index of last deposit (%d) vs expected 3", cachedDeposits.MerkleTrieIndex)
	}

	var deps [][]byte
	for _, d := range append(oldFinalizedDeposits, &newFinalizedDeposit) {
		hash, err := ssz.HashTreeRoot(d.Deposit.Data)
		if err != nil {
			t.Fatalf("Could not hash deposit data")
		}
		deps = append(deps, hash[:])
	}
	trie, err := trieutil.GenerateTrieFromItems(deps, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate deposit trie")
	}

	actualRoot := cachedDeposits.Deposits.HashTreeRoot()
	expectedRoot := trie.HashTreeRoot()
	if actualRoot != expectedRoot {
		t.Errorf("Incorrect deposit trie root (%x) vs expected %x", actualRoot, expectedRoot)
	}
}

func TestFinalizedDeposits_InitializedCorrectly(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	finalizedDeposits := dc.finalizedDeposits
	assert.NotNil(t, finalizedDeposits)
	assert.NotNil(t, finalizedDeposits.Deposits)
	assert.Equal(t, int64(-1), finalizedDeposits.MerkleTrieIndex)
}

func TestNonFinalizedDeposits_ReturnsAllNonFinalizedDeposits(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	finalizedDeposits := []*dbpb.DepositContainer{
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{0},
				},
			},
			Index: 0,
		},
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{1},
				},
			},
			Index: 1,
		},
	}
	dc.deposits = append(finalizedDeposits,
		&dbpb.DepositContainer{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{2},
				},
			},
			Index: 2,
		},
		&dbpb.DepositContainer{
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{3},
				},
			},
			Index: 3,
		})
	dc.InsertFinalizedDeposits(context.Background(), 1)

	deps := dc.NonFinalizedDeposits(context.Background(), nil)
	if len(deps) != 2 {
		t.Errorf("Incorrect number of non-finalized deposits (%d) vs expected 2", len(deps))
	}
}

func TestNonFinalizedDeposits_ReturnsNonFinalizedDepositsUpToBlockNumber(t *testing.T) {
	dc, err := NewDepositCache()
	require.NoError(t, err)

	finalizedDeposits := []*dbpb.DepositContainer{
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{0},
				},
			},
			Index: 0,
		},
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{1},
				},
			},
			Index: 1,
		},
	}
	dc.deposits = append(finalizedDeposits,
		&dbpb.DepositContainer{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{2},
				},
			},
			Index: 2,
		},
		&dbpb.DepositContainer{
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey: []byte{3},
				},
			},
			Index: 3,
		})
	dc.InsertFinalizedDeposits(context.Background(), 1)

	deps := dc.NonFinalizedDeposits(context.Background(), big.NewInt(10))
	if len(deps) != 1 {
		t.Errorf("Incorrect number of non-finalized deposits (%d) vs expected 1", len(deps))
	}
}
