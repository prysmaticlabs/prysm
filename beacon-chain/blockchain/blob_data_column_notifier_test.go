package blockchain

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/prysmaticlabs/prysm/v5/config/features"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestBlobDataColumnNotifier(t *testing.T) {
	resetFn := features.InitWithReset(&features.Flags{
		EnablePeerDAS: true,
	})
	defer resetFn()

	nodeID := util.Random32Bytes(t)
	bn, err := newBlobDataColumnNotifier(enode.ID(nodeID))
	require.NoError(t, err)

	expectedColumnNum := params.BeaconConfig().CustodyRequirement * params.BeaconConfig().NumberOfColumns / params.BeaconConfig().DataColumnSidecarSubnetCount
	require.Equal(t, int(expectedColumnNum), len(bn.columnsNeedsCustody))

	// Check available before receiving any data columns.
	blockRoot := [32]byte{1, 2, 3}
	dataAvailableCh := bn.dataAvailable(blockRoot)
	go func() {
		for col := range bn.columnsNeedsCustody {
			bn.receiveBlobDataColumn(blockRoot, col)
		}
	}()

	select {
	case <-dataAvailableCh:
	case <-time.After(10 * time.Second):
		t.Fatalf("Timed out waiting for data columns to be available")
	}

	// receive column first, then call DataAvailable
	blockRoot = [32]byte{4, 5, 6}
	for k := range bn.columnsNeedsCustody {
		bn.receiveBlobDataColumn(blockRoot, k)
		break
	}

	dataAvailableCh = bn.dataAvailable(blockRoot)
	go func() {
		for col := range bn.columnsNeedsCustody {
			bn.receiveBlobDataColumn(blockRoot, col)
		}
	}()

	select {
	case <-dataAvailableCh:
	case <-time.After(10 * time.Second):
		t.Fatalf("Timed out waiting for data columns to be available")
	}

	// received all columns first, then call DataAvailable
	blockRoot = [32]byte{7, 8, 9}
	for col := range bn.columnsNeedsCustody {
		bn.receiveBlobDataColumn(blockRoot, col)
	}

	dataAvailableCh = bn.dataAvailable(blockRoot)
	select {
	case <-dataAvailableCh:
	case <-time.After(10 * time.Second):
		t.Fatalf("Timed out waiting for data columns to be available")
	}

	// data columns are not available
	blockRoot = [32]byte{10, 11, 12}
	dataAvailableCh = bn.dataAvailable(blockRoot)
	go func() {
		for col := range bn.columnsNeedsCustody {
			bn.receiveBlobDataColumn(blockRoot, col)
			break
		}
	}()

	select {
	case <-dataAvailableCh:
		t.Fatalf("Data columns should not be available")
	case <-time.After(5 * time.Second):
		require.Equal(t, 3, len(bn.missingBlobDataColumns(blockRoot)))
	}
}
