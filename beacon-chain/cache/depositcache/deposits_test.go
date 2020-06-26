package depositcache

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

const nilDepositErr = "Ignoring nil deposit insertion"

var _ = DepositFetcher(&DepositCache{})

func TestBeaconDB_InsertDeposit_LogsOnNilDepositInsertion(t *testing.T) {
	hook := logTest.NewGlobal()
	dc := DepositCache{}

	dc.InsertDeposit(context.Background(), nil, 1, 0, [32]byte{})

	if len(dc.deposits) != 0 {
		t.Fatal("Number of deposits changed")
	}
	if hook.LastEntry().Message != nilDepositErr {
		t.Errorf("Did not log correct message, wanted \"Ignoring nil deposit insertion\", got \"%s\"", hook.LastEntry().Message)
	}
}

func TestBeaconDB_InsertDeposit_MaintainsSortedOrderByIndex(t *testing.T) {
	dc := DepositCache{}

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

func TestBeaconDB_AllDeposits_ReturnsAllDeposits(t *testing.T) {
	dc := DepositCache{}

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

func TestBeaconDB_AllDeposits_FiltersDepositUpToAndIncludingBlockNumber(t *testing.T) {
	dc := DepositCache{}

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

func TestBeaconDB_DepositsNumberAndRootAtHeight_ReturnsAppropriateCountAndRoot(t *testing.T) {
	dc := DepositCache{}

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
	if n != 5 {
		t.Errorf("Returned unexpected deposits number %d wanted %d", n, 5)
	}

	if root != bytesutil.ToBytes32([]byte("root")) {
		t.Errorf("Returned unexpected root: %v", root)
	}
}

func TestBeaconDB_DepositsNumberAndRootAtHeight_ReturnsEmptyTrieIfBlockHeightLessThanOldestDeposit(t *testing.T) {
	dc := DepositCache{}

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
	if n != 0 {
		t.Errorf("Returned unexpected deposits number %d wanted %d", n, 0)
	}

	if root != [32]byte{} {
		t.Errorf("Returned unexpected root: %v", root)
	}
}

func TestBeaconDB_DepositByPubkey_ReturnsFirstMatchingDeposit(t *testing.T) {
	dc := DepositCache{}

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
