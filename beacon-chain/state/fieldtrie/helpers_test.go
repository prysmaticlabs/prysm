package fieldtrie

import (
	"encoding/binary"
	"sync"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/params"
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

func TestValidateIndices_CompressedField(t *testing.T) {
	fakeTrie := &FieldTrie{
		RWMutex:     new(sync.RWMutex),
		reference:   stateutil.NewRef(0),
		fieldLayers: nil,
		field:       types.Balances,
		dataType:    types.CompressedArray,
		length:      params.BeaconConfig().ValidatorRegistryLimit / 4,
		numOfElems:  0,
	}
	goodIdx := params.BeaconConfig().ValidatorRegistryLimit - 1
	assert.NoError(t, fakeTrie.validateIndices([]uint64{goodIdx}))

	badIdx := goodIdx + 1
	assert.ErrorContains(t, "invalid index for field balances", fakeTrie.validateIndices([]uint64{badIdx}))

}
