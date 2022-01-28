package fieldtrie

import (
	"encoding/binary"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
)

func Test_handlePendingAttestation_OutOfRange(t *testing.T) {
	items := make([]*ethpb.PendingAttestation, 1)
	indices := []uint64{3}
	_, err := handlePendingAttestation(items, indices, false)
	assert.ErrorContains(t, "index 3 greater than number of pending attestations 1", err)
}

func Test_handleEth1DataSlice_OutOfRange(t *testing.T) {
	items := make([]*ethpb.Eth1Data, 1)
	indices := []uint64{3}
	_, err := handleEth1DataSlice(items, indices, false)
	assert.ErrorContains(t, "index 3 greater than number of items in eth1 data slice 1", err)

}

func Test_handleValidatorSlice_OutOfRange(t *testing.T) {
	vals := make([]*ethpb.Validator, 1)
	indices := []uint64{3}
	_, err := handleValidatorSlice(vals, indices, false)
	assert.ErrorContains(t, "index 3 greater than number of validators 1", err)
}

func TestBalancesSlice_CorrectRoots_All(t *testing.T) {
	balances := []uint64{5, 2929, 34, 1291, 354305}
	roots, err := handleBalanceSlice(balances, []uint64{}, true)
	assert.NoError(t, err)

	root1 := [32]byte{}
	binary.LittleEndian.PutUint64(root1[:8], balances[0])
	binary.LittleEndian.PutUint64(root1[8:16], balances[1])
	binary.LittleEndian.PutUint64(root1[16:24], balances[2])
	binary.LittleEndian.PutUint64(root1[24:32], balances[3])

	root2 := [32]byte{}
	binary.LittleEndian.PutUint64(root2[:8], balances[4])

	assert.DeepEqual(t, roots, [][32]byte{root1, root2})
}

func TestBalancesSlice_CorrectRoots_Some(t *testing.T) {
	balances := []uint64{5, 2929, 34, 1291, 354305}
	roots, err := handleBalanceSlice(balances, []uint64{2, 3}, false)
	assert.NoError(t, err)

	root1 := [32]byte{}
	binary.LittleEndian.PutUint64(root1[:8], balances[0])
	binary.LittleEndian.PutUint64(root1[8:16], balances[1])
	binary.LittleEndian.PutUint64(root1[16:24], balances[2])
	binary.LittleEndian.PutUint64(root1[24:32], balances[3])

	// Returns root for each indice(even if duplicated)
	assert.DeepEqual(t, roots, [][32]byte{root1, root1})
}
