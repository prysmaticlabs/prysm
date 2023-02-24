package depositcache

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

const nilDepositErr = "Ignoring nil deposit insertion"

var _ DepositFetcher = (*DepositCache)(nil)

func TestInsertDeposit_LogsOnNilDepositInsertion(t *testing.T) {
	hook := logTest.NewGlobal()
	dc, err := New()
	require.NoError(t, err)

	assert.ErrorContains(t, "nil deposit inserted into the cache", dc.InsertDeposit(context.Background(), nil, 1, 0, [32]byte{}))

	require.Equal(t, 0, len(dc.deposits), "Number of deposits changed")
	assert.Equal(t, nilDepositErr, hook.LastEntry().Message)
}

func TestInsertDeposit_MaintainsSortedOrderByIndex(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	insertions := []struct {
		blkNum      uint64
		deposit     *ethpb.Deposit
		index       int64
		expectedErr string
	}{
		{
			blkNum:      0,
			deposit:     &ethpb.Deposit{Data: &ethpb.Deposit_Data{PublicKey: []byte{'A'}}},
			index:       0,
			expectedErr: "",
		},
		{
			blkNum:      0,
			deposit:     &ethpb.Deposit{Data: &ethpb.Deposit_Data{PublicKey: []byte{'B'}}},
			index:       3,
			expectedErr: "wanted deposit with index 1 to be inserted but received 3",
		},
		{
			blkNum:      0,
			deposit:     &ethpb.Deposit{Data: &ethpb.Deposit_Data{PublicKey: []byte{'C'}}},
			index:       1,
			expectedErr: "",
		},
		{
			blkNum:      0,
			deposit:     &ethpb.Deposit{Data: &ethpb.Deposit_Data{PublicKey: []byte{'D'}}},
			index:       4,
			expectedErr: "wanted deposit with index 2 to be inserted but received 4",
		},
		{
			blkNum:      0,
			deposit:     &ethpb.Deposit{Data: &ethpb.Deposit_Data{PublicKey: []byte{'E'}}},
			index:       2,
			expectedErr: "",
		},
	}

	for _, ins := range insertions {
		if ins.expectedErr != "" {
			assert.ErrorContains(t, ins.expectedErr, dc.InsertDeposit(context.Background(), ins.deposit, ins.blkNum, ins.index, [32]byte{}))
		} else {
			assert.NoError(t, dc.InsertDeposit(context.Background(), ins.deposit, ins.blkNum, ins.index, [32]byte{}))
		}
	}

	expectedIndices := []int64{0, 1, 2}
	for i, ei := range expectedIndices {
		assert.Equal(t, ei, dc.deposits[i].Index,
			fmt.Sprintf("dc.deposits[%d].Index = %d, wanted %d", i, dc.deposits[i].Index, ei))
	}
}

func TestAllDeposits_ReturnsAllDeposits(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	deposits := []*ethpb.DepositContainer{
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
	assert.Equal(t, len(deposits), len(d))
}

func TestAllDeposits_FiltersDepositUpToAndIncludingBlockNumber(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	deposits := []*ethpb.DepositContainer{
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
	assert.Equal(t, 5, len(d))
}

func TestDepositsNumberAndRootAtHeight(t *testing.T) {
	wantedRoot := bytesutil.PadTo([]byte("root"), 32)
	t.Run("requesting_last_item_works", func(t *testing.T) {
		dc, err := New()
		require.NoError(t, err)
		dc.deposits = []*ethpb.DepositContainer{
			{
				Eth1BlockHeight: 10,
				Index:           0,
				Deposit:         &ethpb.Deposit{},
			},
			{
				Eth1BlockHeight: 10,
				Index:           1,
				Deposit:         &ethpb.Deposit{},
			},
			{
				Eth1BlockHeight: 11,
				Index:           2,
				Deposit:         &ethpb.Deposit{},
			},
			{
				Eth1BlockHeight: 13,
				Index:           3,
				Deposit:         &ethpb.Deposit{},
				DepositRoot:     wantedRoot,
			},
		}
		n, root := dc.DepositsNumberAndRootAtHeight(context.Background(), big.NewInt(13))
		assert.Equal(t, 4, int(n))
		require.DeepEqual(t, wantedRoot, root[:])
	})
	t.Run("only_one_item", func(t *testing.T) {
		dc, err := New()
		require.NoError(t, err)

		dc.deposits = []*ethpb.DepositContainer{
			{
				Eth1BlockHeight: 10,
				Index:           0,
				Deposit:         &ethpb.Deposit{},
				DepositRoot:     wantedRoot,
			},
		}
		n, root := dc.DepositsNumberAndRootAtHeight(context.Background(), big.NewInt(10))
		assert.Equal(t, 1, int(n))
		require.DeepEqual(t, wantedRoot, root[:])
	})
	t.Run("none_at_height_some_below", func(t *testing.T) {
		dc, err := New()
		require.NoError(t, err)

		dc.deposits = []*ethpb.DepositContainer{
			{
				Eth1BlockHeight: 8,
				Index:           0,
				Deposit:         &ethpb.Deposit{},
			},
			{
				Eth1BlockHeight: 9,
				Index:           1,
				Deposit:         &ethpb.Deposit{},
				DepositRoot:     wantedRoot,
			},
			{
				Eth1BlockHeight: 11,
				Index:           2,
				Deposit:         &ethpb.Deposit{},
			},
		}
		n, root := dc.DepositsNumberAndRootAtHeight(context.Background(), big.NewInt(10))
		assert.Equal(t, 2, int(n))
		require.DeepEqual(t, wantedRoot, root[:])
	})
	t.Run("none_at_height_none_below", func(t *testing.T) {
		dc, err := New()
		require.NoError(t, err)

		dc.deposits = []*ethpb.DepositContainer{
			{
				Eth1BlockHeight: 8,
				Index:           0,
				Deposit:         &ethpb.Deposit{},
				DepositRoot:     wantedRoot,
			},
		}
		n, root := dc.DepositsNumberAndRootAtHeight(context.Background(), big.NewInt(7))
		assert.Equal(t, 0, int(n))
		require.DeepEqual(t, params.BeaconConfig().ZeroHash, root)
	})
	t.Run("none_at_height_one_below", func(t *testing.T) {
		dc, err := New()
		require.NoError(t, err)

		dc.deposits = []*ethpb.DepositContainer{
			{
				Eth1BlockHeight: 8,
				Index:           0,
				Deposit:         &ethpb.Deposit{},
				DepositRoot:     wantedRoot,
			},
		}
		n, root := dc.DepositsNumberAndRootAtHeight(context.Background(), big.NewInt(10))
		assert.Equal(t, 1, int(n))
		require.DeepEqual(t, wantedRoot, root[:])
	})
	t.Run("some_greater_some_lower", func(t *testing.T) {
		dc, err := New()
		require.NoError(t, err)

		dc.deposits = []*ethpb.DepositContainer{
			{
				Eth1BlockHeight: 8,
				Index:           0,
				Deposit:         &ethpb.Deposit{},
			},
			{
				Eth1BlockHeight: 8,
				Index:           1,
				Deposit:         &ethpb.Deposit{},
			},
			{
				Eth1BlockHeight: 9,
				Index:           2,
				Deposit:         &ethpb.Deposit{},
				DepositRoot:     wantedRoot,
			},
			{
				Eth1BlockHeight: 10,
				Index:           3,
				Deposit:         &ethpb.Deposit{},
			},
			{
				Eth1BlockHeight: 10,
				Index:           4,
				Deposit:         &ethpb.Deposit{},
			},
		}
		n, root := dc.DepositsNumberAndRootAtHeight(context.Background(), big.NewInt(9))
		assert.Equal(t, 3, int(n))
		require.DeepEqual(t, wantedRoot, root[:])
	})
}

func TestDepositByPubkey_ReturnsFirstMatchingDeposit(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)
	ctrs := []*ethpb.DepositContainer{
		{
			Eth1BlockHeight: 9,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("pk0"), 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
		},
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("pk1"), 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
		},
		{
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("pk1"), 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
		},
		{
			Eth1BlockHeight: 12,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("pk2"), 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
		},
	}
	dc.InsertDepositContainers(context.Background(), ctrs)

	pk1 := bytesutil.PadTo([]byte("pk1"), 48)
	dep, blkNum := dc.DepositByPubkey(context.Background(), pk1)

	if dep == nil || !bytes.Equal(dep.Data.PublicKey, pk1) {
		t.Error("Returned wrong deposit")
	}
	assert.Equal(t, 0, blkNum.Cmp(big.NewInt(10)),
		fmt.Sprintf("Returned wrong block number %v", blkNum))
}

func TestFinalizedDeposits_DepositsCachedCorrectly(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{0}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 0,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{1}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 1,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{2}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 2,
		},
	}
	dc.deposits = append(finalizedDeposits, &ethpb.DepositContainer{
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             bytesutil.PadTo([]byte{3}, 48),
				WithdrawalCredentials: make([]byte, 32),
				Signature:             make([]byte, 96),
			},
		},
		Index: 3,
	})

	dc.InsertFinalizedDeposits(context.Background(), 2)

	cachedDeposits := dc.FinalizedDeposits(context.Background())
	require.NotNil(t, cachedDeposits, "Deposits not cached")
	assert.Equal(t, int64(2), cachedDeposits.MerkleTrieIndex)

	var deps [][]byte
	for _, d := range finalizedDeposits {
		hash, err := d.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Could not hash deposit data")
		deps = append(deps, hash[:])
	}
	generatedTrie, err := trie.GenerateTrieFromItems(deps, params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not generate deposit trie")
	rootA, err := generatedTrie.HashTreeRoot()
	require.NoError(t, err)
	rootB, err := cachedDeposits.Deposits.HashTreeRoot()
	require.NoError(t, err)
	assert.Equal(t, rootA, rootB)
}

func TestFinalizedDeposits_UtilizesPreviouslyCachedDeposits(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	oldFinalizedDeposits := []*ethpb.DepositContainer{
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{0}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 0,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{1}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 1,
		},
	}
	newFinalizedDeposit := &ethpb.DepositContainer{
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             bytesutil.PadTo([]byte{2}, 48),
				WithdrawalCredentials: make([]byte, 32),
				Signature:             make([]byte, 96),
			},
		},
		Index: 2,
	}
	dc.deposits = oldFinalizedDeposits
	dc.InsertFinalizedDeposits(context.Background(), 1)

	dc.InsertFinalizedDeposits(context.Background(), 2)

	dc.deposits = append(dc.deposits, []*ethpb.DepositContainer{newFinalizedDeposit}...)

	cachedDeposits := dc.FinalizedDeposits(context.Background())
	require.NotNil(t, cachedDeposits, "Deposits not cached")
	assert.Equal(t, int64(1), cachedDeposits.MerkleTrieIndex)

	var deps [][]byte
	for _, d := range oldFinalizedDeposits {
		hash, err := d.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Could not hash deposit data")
		deps = append(deps, hash[:])
	}
	generatedTrie, err := trie.GenerateTrieFromItems(deps, params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not generate deposit trie")
	rootA, err := generatedTrie.HashTreeRoot()
	require.NoError(t, err)
	rootB, err := cachedDeposits.Deposits.HashTreeRoot()
	require.NoError(t, err)
	assert.Equal(t, rootA, rootB)
}

func TestFinalizedDeposits_HandleZeroDeposits(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	dc.InsertFinalizedDeposits(context.Background(), 2)

	cachedDeposits := dc.FinalizedDeposits(context.Background())
	require.NotNil(t, cachedDeposits, "Deposits not cached")
	assert.Equal(t, int64(-1), cachedDeposits.MerkleTrieIndex)
}

func TestFinalizedDeposits_HandleSmallerThanExpectedDeposits(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{0}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 0,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{1}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 1,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{2}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 2,
		},
	}
	dc.deposits = finalizedDeposits

	dc.InsertFinalizedDeposits(context.Background(), 5)

	cachedDeposits := dc.FinalizedDeposits(context.Background())
	require.NotNil(t, cachedDeposits, "Deposits not cached")
	assert.Equal(t, int64(2), cachedDeposits.MerkleTrieIndex)
}

func TestFinalizedDeposits_HandleLowerEth1DepositIndex(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{0}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 0,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{1}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 1,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{2}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 2,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{3}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 3,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{4}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 4,
		},
		{
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{5}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 5,
		},
	}
	dc.deposits = finalizedDeposits

	dc.InsertFinalizedDeposits(context.Background(), 5)

	// Reinsert finalized deposits with a lower index.
	dc.InsertFinalizedDeposits(context.Background(), 2)

	cachedDeposits := dc.FinalizedDeposits(context.Background())
	require.NotNil(t, cachedDeposits, "Deposits not cached")
	assert.Equal(t, int64(5), cachedDeposits.MerkleTrieIndex)
}

func TestFinalizedDeposits_InitializedCorrectly(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	finalizedDeposits := dc.finalizedDeposits
	assert.NotNil(t, finalizedDeposits)
	assert.NotNil(t, finalizedDeposits.Deposits)
	assert.Equal(t, int64(-1), finalizedDeposits.MerkleTrieIndex)
}

func TestNonFinalizedDeposits_ReturnsAllNonFinalizedDeposits(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{0}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 0,
		},
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{1}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 1,
		},
	}
	dc.deposits = append(finalizedDeposits,
		&ethpb.DepositContainer{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{2}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 2,
		},
		&ethpb.DepositContainer{
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{3}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 3,
		})
	dc.InsertFinalizedDeposits(context.Background(), 1)

	deps := dc.NonFinalizedDeposits(context.Background(), 1, nil)
	assert.Equal(t, 2, len(deps))
}

func TestNonFinalizedDeposits_ReturnsNonFinalizedDepositsUpToBlockNumber(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{0}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 0,
		},
		{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{1}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 1,
		},
	}
	dc.deposits = append(finalizedDeposits,
		&ethpb.DepositContainer{
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{2}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 2,
		},
		&ethpb.DepositContainer{
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{3}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: 3,
		})
	dc.InsertFinalizedDeposits(context.Background(), 1)

	deps := dc.NonFinalizedDeposits(context.Background(), 1, big.NewInt(10))
	assert.Equal(t, 1, len(deps))
}

func TestFinalizedDeposits_ReturnsTrieCorrectly(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	generateCtr := func(height uint64, index int64) *ethpb.DepositContainer {
		return &ethpb.DepositContainer{
			Eth1BlockHeight: height,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{uint8(index)}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
			Index: index,
		}
	}

	finalizedDeposits := []*ethpb.DepositContainer{
		generateCtr(10, 0),
		generateCtr(11, 1),
		generateCtr(12, 2),
		generateCtr(12, 3),
		generateCtr(13, 4),
		generateCtr(13, 5),
		generateCtr(13, 6),
		generateCtr(14, 7),
	}
	dc.deposits = append(finalizedDeposits,
		generateCtr(15, 8),
		generateCtr(15, 9),
		generateCtr(30, 10))
	trieItems := make([][]byte, 0, len(dc.deposits))
	for _, dep := range dc.allDeposits(big.NewInt(30)) {
		depHash, err := dep.Data.HashTreeRoot()
		assert.NoError(t, err)
		trieItems = append(trieItems, depHash[:])
	}
	depositTrie, err := trie.GenerateTrieFromItems(trieItems, params.BeaconConfig().DepositContractTreeDepth)
	assert.NoError(t, err)

	// Perform this in a non-sensical ordering
	dc.InsertFinalizedDeposits(context.Background(), 10)
	dc.InsertFinalizedDeposits(context.Background(), 2)
	dc.InsertFinalizedDeposits(context.Background(), 3)
	dc.InsertFinalizedDeposits(context.Background(), 4)

	// Mimick finalized deposit trie fetch.
	fd := dc.FinalizedDeposits(context.Background())
	deps := dc.NonFinalizedDeposits(context.Background(), fd.MerkleTrieIndex, big.NewInt(14))
	insertIndex := fd.MerkleTrieIndex + 1

	for _, dep := range deps {
		depHash, err := dep.Data.HashTreeRoot()
		assert.NoError(t, err)
		if err = fd.Deposits.Insert(depHash[:], int(insertIndex)); err != nil {
			assert.NoError(t, err)
		}
		insertIndex++
	}
	dc.InsertFinalizedDeposits(context.Background(), 15)
	dc.InsertFinalizedDeposits(context.Background(), 15)
	dc.InsertFinalizedDeposits(context.Background(), 14)

	fd = dc.FinalizedDeposits(context.Background())
	deps = dc.NonFinalizedDeposits(context.Background(), fd.MerkleTrieIndex, big.NewInt(30))
	insertIndex = fd.MerkleTrieIndex + 1

	for _, dep := range deps {
		depHash, err := dep.Data.HashTreeRoot()
		assert.NoError(t, err)
		if err = fd.Deposits.Insert(depHash[:], int(insertIndex)); err != nil {
			assert.NoError(t, err)
		}
		insertIndex++
	}
	assert.Equal(t, fd.Deposits.NumOfItems(), depositTrie.NumOfItems())
}

func TestPruneProofs_Ok(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	deposits := []struct {
		blkNum  uint64
		deposit *ethpb.Deposit
		index   int64
	}{
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk0"), 48)}},
			index: 0,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk1"), 48)}},
			index: 1,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk2"), 48)}},
			index: 2,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk3"), 48)}},
			index: 3,
		},
	}

	for _, ins := range deposits {
		assert.NoError(t, dc.InsertDeposit(context.Background(), ins.deposit, ins.blkNum, ins.index, [32]byte{}))
	}

	require.NoError(t, dc.PruneProofs(context.Background(), 1))

	assert.DeepEqual(t, [][]byte(nil), dc.deposits[0].Deposit.Proof)
	assert.DeepEqual(t, [][]byte(nil), dc.deposits[1].Deposit.Proof)
	assert.NotNil(t, dc.deposits[2].Deposit.Proof)
	assert.NotNil(t, dc.deposits[3].Deposit.Proof)
}

func TestPruneProofs_SomeAlreadyPruned(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	deposits := []struct {
		blkNum  uint64
		deposit *ethpb.Deposit
		index   int64
	}{
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: nil, Data: &ethpb.Deposit_Data{
				PublicKey: bytesutil.PadTo([]byte("pk0"), 48)}},
			index: 0,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: nil, Data: &ethpb.Deposit_Data{
				PublicKey: bytesutil.PadTo([]byte("pk1"), 48)}}, index: 1,
		},
		{
			blkNum:  0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(), Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk2"), 48)}},
			index:   2,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk3"), 48)}},
			index: 3,
		},
	}

	for _, ins := range deposits {
		assert.NoError(t, dc.InsertDeposit(context.Background(), ins.deposit, ins.blkNum, ins.index, [32]byte{}))
	}

	require.NoError(t, dc.PruneProofs(context.Background(), 2))

	assert.DeepEqual(t, [][]byte(nil), dc.deposits[2].Deposit.Proof)
}

func TestPruneProofs_PruneAllWhenDepositIndexTooBig(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	deposits := []struct {
		blkNum  uint64
		deposit *ethpb.Deposit
		index   int64
	}{
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk0"), 48)}},
			index: 0,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk1"), 48)}},
			index: 1,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk2"), 48)}},
			index: 2,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk3"), 48)}},
			index: 3,
		},
	}

	for _, ins := range deposits {
		assert.NoError(t, dc.InsertDeposit(context.Background(), ins.deposit, ins.blkNum, ins.index, [32]byte{}))
	}

	require.NoError(t, dc.PruneProofs(context.Background(), 99))

	assert.DeepEqual(t, [][]byte(nil), dc.deposits[0].Deposit.Proof)
	assert.DeepEqual(t, [][]byte(nil), dc.deposits[1].Deposit.Proof)
	assert.DeepEqual(t, [][]byte(nil), dc.deposits[2].Deposit.Proof)
	assert.DeepEqual(t, [][]byte(nil), dc.deposits[3].Deposit.Proof)
}

func TestPruneProofs_CorrectlyHandleLastIndex(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	deposits := []struct {
		blkNum  uint64
		deposit *ethpb.Deposit
		index   int64
	}{
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk0"), 48)}},
			index: 0,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk1"), 48)}},
			index: 1,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk2"), 48)}},
			index: 2,
		},
		{
			blkNum: 0,
			deposit: &ethpb.Deposit{Proof: makeDepositProof(),
				Data: &ethpb.Deposit_Data{PublicKey: bytesutil.PadTo([]byte("pk3"), 48)}},
			index: 3,
		},
	}

	for _, ins := range deposits {
		assert.NoError(t, dc.InsertDeposit(context.Background(), ins.deposit, ins.blkNum, ins.index, [32]byte{}))
	}

	require.NoError(t, dc.PruneProofs(context.Background(), 4))

	assert.DeepEqual(t, [][]byte(nil), dc.deposits[0].Deposit.Proof)
	assert.DeepEqual(t, [][]byte(nil), dc.deposits[1].Deposit.Proof)
	assert.DeepEqual(t, [][]byte(nil), dc.deposits[2].Deposit.Proof)
	assert.DeepEqual(t, [][]byte(nil), dc.deposits[3].Deposit.Proof)
}

func TestDepositMap_WorksCorrectly(t *testing.T) {
	dc, err := New()
	require.NoError(t, err)

	pk0 := bytesutil.PadTo([]byte("pk0"), 48)
	dep, _ := dc.DepositByPubkey(context.Background(), pk0)
	var nilDep *ethpb.Deposit
	assert.DeepEqual(t, nilDep, dep)

	dep = &ethpb.Deposit{Proof: makeDepositProof(), Data: &ethpb.Deposit_Data{PublicKey: pk0, Amount: 1000}}
	assert.NoError(t, dc.InsertDeposit(context.Background(), dep, 1000, 0, [32]byte{}))

	dep, _ = dc.DepositByPubkey(context.Background(), pk0)
	assert.NotEqual(t, nilDep, dep)
	assert.Equal(t, uint64(1000), dep.Data.Amount)

	dep = &ethpb.Deposit{Proof: makeDepositProof(), Data: &ethpb.Deposit_Data{PublicKey: pk0, Amount: 10000}}
	assert.NoError(t, dc.InsertDeposit(context.Background(), dep, 1000, 1, [32]byte{}))

	// Make sure we have the same deposit returned over here.
	dep, _ = dc.DepositByPubkey(context.Background(), pk0)
	assert.NotEqual(t, nilDep, dep)
	assert.Equal(t, uint64(1000), dep.Data.Amount)

	// Make sure another key doesn't work.
	pk1 := bytesutil.PadTo([]byte("pk1"), 48)
	dep, _ = dc.DepositByPubkey(context.Background(), pk1)
	assert.DeepEqual(t, nilDep, dep)
}

func makeDepositProof() [][]byte {
	proof := make([][]byte, int(params.BeaconConfig().DepositContractTreeDepth)+1)
	for i := range proof {
		proof[i] = make([]byte, 32)
	}
	return proof
}
