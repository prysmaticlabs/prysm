package fieldtrie

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	customtypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/custom-types"
	nativeStateTypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	stateTypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func Test_handlePendingAttestation_OutOfRange(t *testing.T) {
	items := make([]*ethpb.PendingAttestation, 1)
	indices := []uint64{3}
	_, err := handlePendingAttestationSlice(items, indices, false)
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
		field:       stateTypes.Balances,
		dataType:    stateTypes.CompressedArray,
		length:      params.BeaconConfig().ValidatorRegistryLimit / 4,
		numOfElems:  0,
	}
	goodIdx := params.BeaconConfig().ValidatorRegistryLimit - 1
	assert.NoError(t, fakeTrie.validateIndices([]uint64{goodIdx}))

	badIdx := goodIdx + 1
	assert.ErrorContains(t, "invalid index for field balances", fakeTrie.validateIndices([]uint64{badIdx}))

}

func TestFieldTrie_NativeState_fieldConvertersNative(t *testing.T) {
	features.Init(&features.Flags{EnableNativeState: true})
	type args struct {
		field      stateTypes.BeaconStateField
		indices    []uint64
		elements   interface{}
		convertAll bool
	}
	tests := []struct {
		name           string
		args           *args
		wantHex        []string
		errMsg         string
		expectedLength int
	}{
		{
			name: "BlockRoots [][]bytes",
			args: &args{
				field:      nativeStateTypes.FieldIndex(5),
				indices:    []uint64{},
				elements:   [][]byte{[]byte("dfsadfsadf")},
				convertAll: true,
			},
			wantHex: []string{"0x6466736164667361646600000000000000000000000000000000000000000000"},
		},
		{
			name: "BlockRoots customtypes.BlockRoots",
			args: &args{
				field:      nativeStateTypes.FieldIndex(5),
				indices:    []uint64{},
				elements:   &customtypes.BlockRoots{},
				convertAll: true,
			},
			wantHex:        []string{"0x0000000000000000000000000000000000000000000000000000000000000000"},
			expectedLength: 8192,
		},
		{
			name: "BlockRoots type not found",
			args: &args{
				field:      nativeStateTypes.FieldIndex(5),
				indices:    []uint64{},
				elements:   123,
				convertAll: true,
			},
			wantHex: nil,
			errMsg:  "Incorrect type used for block roots",
		},
		{
			name: "BlockRoots [][]bytes",
			args: &args{
				field:      nativeStateTypes.FieldIndex(5),
				indices:    []uint64{},
				elements:   [][]byte{[]byte("dfsadfsadf")},
				convertAll: true,
			},
			wantHex: []string{"0x6466736164667361646600000000000000000000000000000000000000000000"},
		},
		{
			name: "StateRoots [][]bytes",
			args: &args{
				field:      nativeStateTypes.FieldIndex(6),
				indices:    []uint64{},
				elements:   [][]byte{[]byte("dfsadfsadf")},
				convertAll: true,
			},
			wantHex: []string{"0x6466736164667361646600000000000000000000000000000000000000000000"},
		},
		{
			name: "StateRoots customtypes.StateRoots",
			args: &args{
				field:      nativeStateTypes.FieldIndex(6),
				indices:    []uint64{},
				elements:   &customtypes.StateRoots{},
				convertAll: true,
			},
			wantHex:        []string{"0x0000000000000000000000000000000000000000000000000000000000000000"},
			expectedLength: 8192,
		},
		{
			name: "StateRoots type not found",
			args: &args{
				field:      nativeStateTypes.FieldIndex(6),
				indices:    []uint64{},
				elements:   123,
				convertAll: true,
			},
			wantHex: nil,
			errMsg:  "Incorrect type used for state roots",
		},
		{
			name: "StateRoots [][]bytes convert all false",
			args: &args{
				field:      nativeStateTypes.FieldIndex(6),
				indices:    []uint64{},
				elements:   [][]byte{[]byte("dfsadfsadf")},
				convertAll: false,
			},
			wantHex: []string{"0x6466736164667361646600000000000000000000000000000000000000000000"},
		},
		{
			name: "StateRoots customtypes.StateRoots convert all false",
			args: &args{
				field:      nativeStateTypes.FieldIndex(6),
				indices:    []uint64{},
				elements:   &customtypes.StateRoots{},
				convertAll: false,
			},
			wantHex:        []string{"0x0000000000000000000000000000000000000000000000000000000000000000"},
			expectedLength: 8192,
		},
		{
			name: "RandaoMixes [][]bytes",
			args: &args{
				field:      nativeStateTypes.FieldIndex(13),
				indices:    []uint64{},
				elements:   [][]byte{[]byte("dfsadfsadf")},
				convertAll: true,
			},
			wantHex: []string{"0x6466736164667361646600000000000000000000000000000000000000000000"},
		},
		{
			name: "RandaoMixes customtypes.RandaoMixes",
			args: &args{
				field:      nativeStateTypes.FieldIndex(13),
				indices:    []uint64{},
				elements:   &customtypes.RandaoMixes{},
				convertAll: true,
			},
			wantHex:        []string{"0x0000000000000000000000000000000000000000000000000000000000000000"},
			expectedLength: 65536,
		},
		{
			name: "RandaoMixes type not found",
			args: &args{
				field:      nativeStateTypes.FieldIndex(13),
				indices:    []uint64{},
				elements:   123,
				convertAll: true,
			},
			wantHex: nil,
			errMsg:  "Incorrect type used for randao mixes",
		},
		{
			name: "Eth1DataVotes type not found",
			args: &args{
				field:   nativeStateTypes.FieldIndex(9),
				indices: []uint64{},
				elements: []*ethpb.Eth1Data{
					{
						DepositRoot:  make([]byte, fieldparams.RootLength),
						DepositCount: 1,
					},
				},
				convertAll: true,
			},
			wantHex: []string{"0x4833912e1264aef8a18392d795f3f2eed17cf5c0e8471cb0c0db2ec5aca10231"},
		},
		{
			name: "Eth1DataVotes convertAll false",
			args: &args{
				field:   nativeStateTypes.FieldIndex(9),
				indices: []uint64{1},
				elements: []*ethpb.Eth1Data{
					{
						DepositRoot:  make([]byte, fieldparams.RootLength),
						DepositCount: 1,
					},
				},
				convertAll: false,
			},
			wantHex: []string{"0x4833912e1264aef8a18392d795f3f2eed17cf5c0e8471cb0c0db2ec5aca10231"},
		},
		{
			name: "Eth1DataVotes type not found",
			args: &args{
				field:      nativeStateTypes.FieldIndex(9),
				indices:    []uint64{},
				elements:   123,
				convertAll: true,
			},
			wantHex: nil,
			errMsg:  fmt.Sprintf("Wanted type of %v", reflect.TypeOf([]*ethpb.Eth1Data{}).Name()),
		},
		{
			name: "Balance",
			args: &args{
				field:      nativeStateTypes.FieldIndex(12),
				indices:    []uint64{},
				elements:   []uint64{12321312321, 12131241234123123},
				convertAll: true,
			},
			wantHex: []string{"0x414e68de0200000073c971b44c192b0000000000000000000000000000000000"},
		},
		{
			name: "Validators",
			args: &args{
				field:   nativeStateTypes.FieldIndex(11),
				indices: []uint64{},
				elements: []*ethpb.Validator{
					{
						ActivationEpoch: 1,
					},
				},
				convertAll: true,
			},
			wantHex: []string{"0x79817c24fc7ba90cdac48fd462fafc1cb501884e847b18733f7ca6df214a301e"},
		},
		{
			name: "Validators not found",
			args: &args{
				field:      nativeStateTypes.FieldIndex(11),
				indices:    []uint64{},
				elements:   123,
				convertAll: true,
			},
			wantHex: nil,
			errMsg:  fmt.Sprintf("Wanted type of %v", reflect.TypeOf([]*ethpb.Validator{}).Name()),
		},
		{
			name: "Attestations",
			args: &args{
				field:   nativeStateTypes.FieldIndex(15),
				indices: []uint64{},
				elements: []*ethpb.PendingAttestation{
					{
						ProposerIndex: 1,
					},
				},
				convertAll: true,
			},
			wantHex: []string{"0x7d7696e7f12593934afcd87a0d38e1a981bee63cb4cf0568ba36a6e0596eeccb"},
		},
		{
			name: "Attestations",
			args: &args{
				field:   nativeStateTypes.FieldIndex(15),
				indices: []uint64{1},
				elements: []*ethpb.PendingAttestation{
					{
						ProposerIndex: 1,
					},
				},
				convertAll: false,
			},
			wantHex: []string{"0x7d7696e7f12593934afcd87a0d38e1a981bee63cb4cf0568ba36a6e0596eeccb"},
		},
		{
			name: "Type not found",
			args: &args{
				field:   nativeStateTypes.FieldIndex(999),
				indices: []uint64{},
				elements: []*ethpb.PendingAttestation{
					{
						ProposerIndex: 1,
					},
				},
				convertAll: true,
			},
			errMsg: "got unsupported type of",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots, err := fieldConvertersNative(tt.args.field, tt.args.indices, tt.args.elements, tt.args.convertAll)
			if err != nil && tt.errMsg != "" {
				require.ErrorContains(t, tt.errMsg, err)
			} else {
				for i, root := range roots {
					hex := hexutil.Encode(root[:])
					require.Equal(t, tt.wantHex[i], hex)
					if tt.expectedLength != 0 {
						require.Equal(t, len(roots), tt.expectedLength)
						break
					}
				}
			}
		})
	}
	features.Init(&features.Flags{EnableNativeState: false})
}
