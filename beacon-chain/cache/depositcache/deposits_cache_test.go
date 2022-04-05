package depositcache

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
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
	trie, err := trie.GenerateTrieFromItems(deps, params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not generate deposit trie")
	assert.Equal(t, trie.HashTreeRoot(), cachedDeposits.Deposits.HashTreeRoot())
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
	newFinalizedDeposit := ethpb.DepositContainer{
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
	// Artificially exclude old deposits so that they can only be retrieved from previously finalized deposits.
	dc.deposits = []*ethpb.DepositContainer{&newFinalizedDeposit}

	dc.InsertFinalizedDeposits(context.Background(), 2)

	cachedDeposits := dc.FinalizedDeposits(context.Background())
	require.NotNil(t, cachedDeposits, "Deposits not cached")
	assert.Equal(t, int64(2), cachedDeposits.MerkleTrieIndex)

	var deps [][]byte
	for _, d := range append(oldFinalizedDeposits, &newFinalizedDeposit) {
		hash, err := d.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Could not hash deposit data")
		deps = append(deps, hash[:])
	}
	trie, err := trie.GenerateTrieFromItems(deps, params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not generate deposit trie")
	assert.Equal(t, trie.HashTreeRoot(), cachedDeposits.Deposits.HashTreeRoot())
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

	deps := dc.NonFinalizedDeposits(context.Background(), nil)
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

	deps := dc.NonFinalizedDeposits(context.Background(), big.NewInt(10))
	assert.Equal(t, 1, len(deps))
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
