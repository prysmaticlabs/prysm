package beacon_api

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestBeaconBlockProtoHelpers_ConvertAttesterSlashingsToProto(t *testing.T) {
	testCases := []struct {
		name                 string
		generateInput        func() []*apimiddleware.AttesterSlashingJson
		expectedResult       []*ethpb.AttesterSlashing
		expectedErrorMessage string
	}{
		{
			name:                 "nil attester slashing",
			expectedErrorMessage: "attester slashing at index `0` is nil",
			generateInput: func() []*apimiddleware.AttesterSlashingJson {
				return []*apimiddleware.AttesterSlashingJson{
					nil,
				}
			},
		},
		{
			name:                 "bad attestation 1",
			expectedErrorMessage: "failed to get attestation 1",
			generateInput: func() []*apimiddleware.AttesterSlashingJson {
				return []*apimiddleware.AttesterSlashingJson{
					{
						Attestation_1: nil,
						Attestation_2: nil,
					},
				}
			},
		},
		{
			name:                 "bad attestation 2",
			expectedErrorMessage: "failed to get attestation 2",
			generateInput: func() []*apimiddleware.AttesterSlashingJson {
				input := generateAttesterSlashingsJson()
				input[0].Attestation_2 = nil
				return input
			},
		},
		{
			name:           "valid",
			generateInput:  generateAttesterSlashingsJson,
			expectedResult: generateAttesterSlashingsProto(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := convertAttesterSlashingsToProto(testCase.generateInput())

			if testCase.expectedResult != nil {
				require.NoError(t, err)
				assert.DeepEqual(t, testCase.expectedResult, result)
			} else if testCase.expectedErrorMessage != "" {
				assert.ErrorContains(t, testCase.expectedErrorMessage, err)
			}
		})
	}
}

func generateAttesterSlashingsJson() []*apimiddleware.AttesterSlashingJson {
	return []*apimiddleware.AttesterSlashingJson{
		{
			Attestation_1: &apimiddleware.IndexedAttestationJson{
				AttestingIndices: []string{"1", "2"},
				Data: &apimiddleware.AttestationDataJson{
					Slot:            "3",
					CommitteeIndex:  "4",
					BeaconBlockRoot: hexutil.Encode([]byte{5}),
					Source: &apimiddleware.CheckpointJson{
						Epoch: "6",
						Root:  hexutil.Encode([]byte{7}),
					},
					Target: &apimiddleware.CheckpointJson{
						Epoch: "8",
						Root:  hexutil.Encode([]byte{9}),
					},
				},
				Signature: hexutil.Encode([]byte{10}),
			},
			Attestation_2: &apimiddleware.IndexedAttestationJson{
				AttestingIndices: []string{"11", "12"},
				Data: &apimiddleware.AttestationDataJson{
					Slot:            "13",
					CommitteeIndex:  "14",
					BeaconBlockRoot: hexutil.Encode([]byte{15}),
					Source: &apimiddleware.CheckpointJson{
						Epoch: "16",
						Root:  hexutil.Encode([]byte{17}),
					},
					Target: &apimiddleware.CheckpointJson{
						Epoch: "18",
						Root:  hexutil.Encode([]byte{19}),
					},
				},
				Signature: hexutil.Encode([]byte{20}),
			},
		},
		{
			Attestation_1: &apimiddleware.IndexedAttestationJson{
				AttestingIndices: []string{"21", "22"},
				Data: &apimiddleware.AttestationDataJson{
					Slot:            "23",
					CommitteeIndex:  "24",
					BeaconBlockRoot: hexutil.Encode([]byte{25}),
					Source: &apimiddleware.CheckpointJson{
						Epoch: "26",
						Root:  hexutil.Encode([]byte{27}),
					},
					Target: &apimiddleware.CheckpointJson{
						Epoch: "28",
						Root:  hexutil.Encode([]byte{29}),
					},
				},
				Signature: hexutil.Encode([]byte{30}),
			},
			Attestation_2: &apimiddleware.IndexedAttestationJson{
				AttestingIndices: []string{"31", "32"},
				Data: &apimiddleware.AttestationDataJson{
					Slot:            "33",
					CommitteeIndex:  "34",
					BeaconBlockRoot: hexutil.Encode([]byte{35}),
					Source: &apimiddleware.CheckpointJson{
						Epoch: "36",
						Root:  hexutil.Encode([]byte{37}),
					},
					Target: &apimiddleware.CheckpointJson{
						Epoch: "38",
						Root:  hexutil.Encode([]byte{39}),
					},
				},
				Signature: hexutil.Encode([]byte{40}),
			},
		},
	}
}

func generateAttesterSlashingsProto() []*ethpb.AttesterSlashing {
	return []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2},
				Data: &ethpb.AttestationData{
					Slot:            3,
					CommitteeIndex:  4,
					BeaconBlockRoot: []byte{5},
					Source: &ethpb.Checkpoint{
						Epoch: 6,
						Root:  []byte{7},
					},
					Target: &ethpb.Checkpoint{
						Epoch: 8,
						Root:  []byte{9},
					},
				},
				Signature: []byte{10},
			},
			Attestation_2: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{11, 12},
				Data: &ethpb.AttestationData{
					Slot:            13,
					CommitteeIndex:  14,
					BeaconBlockRoot: []byte{15},
					Source: &ethpb.Checkpoint{
						Epoch: 16,
						Root:  []byte{17},
					},
					Target: &ethpb.Checkpoint{
						Epoch: 18,
						Root:  []byte{19},
					},
				},
				Signature: []byte{20},
			},
		},
		{
			Attestation_1: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{21, 22},
				Data: &ethpb.AttestationData{
					Slot:            23,
					CommitteeIndex:  24,
					BeaconBlockRoot: []byte{25},
					Source: &ethpb.Checkpoint{
						Epoch: 26,
						Root:  []byte{27},
					},
					Target: &ethpb.Checkpoint{
						Epoch: 28,
						Root:  []byte{29},
					},
				},
				Signature: []byte{30},
			},
			Attestation_2: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{31, 32},
				Data: &ethpb.AttestationData{
					Slot:            33,
					CommitteeIndex:  34,
					BeaconBlockRoot: []byte{35},
					Source: &ethpb.Checkpoint{
						Epoch: 36,
						Root:  []byte{37},
					},
					Target: &ethpb.Checkpoint{
						Epoch: 38,
						Root:  []byte{39},
					},
				},
				Signature: []byte{40},
			},
		},
	}
}
