package beacon_api

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

const getSlashableAttestationsTestEndpoint = "/eth/v1/beacon/pool/attester_slashings"

func TestGetSlashableAttestations_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	jsonAttesterSlashings := &apimiddleware.AttesterSlashingsPoolResponseJson{}

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		getSlashableAttestationsTestEndpoint,
		jsonAttesterSlashings,
	).Return(
		nil,
		nil,
	).SetArg(
		2,
		apimiddleware.AttesterSlashingsPoolResponseJson{
			Data: []*apimiddleware.AttesterSlashingJson{
				{
					Attestation_1: &apimiddleware.IndexedAttestationJson{
						AttestingIndices: []string{"1", "2", "3", "4"},
						Data: &apimiddleware.AttestationDataJson{
							Slot:            "5",
							CommitteeIndex:  "6",
							BeaconBlockRoot: hexutil.Encode([]byte{7}),
							Source: &apimiddleware.CheckpointJson{
								Epoch: "8",
								Root:  hexutil.Encode([]byte{9}),
							},
							Target: &apimiddleware.CheckpointJson{
								Epoch: "10",
								Root:  hexutil.Encode([]byte{11}),
							},
						},
						Signature: hexutil.Encode([]byte{12}),
					},
					Attestation_2: &apimiddleware.IndexedAttestationJson{
						AttestingIndices: []string{"3", "4", "13", "14"},
						Data: &apimiddleware.AttestationDataJson{
							Slot:            "15",
							CommitteeIndex:  "16",
							BeaconBlockRoot: hexutil.Encode([]byte{17}),
							Source: &apimiddleware.CheckpointJson{
								Epoch: "18",
								Root:  hexutil.Encode([]byte{19}),
							},
							Target: &apimiddleware.CheckpointJson{
								Epoch: "10",
								Root:  hexutil.Encode([]byte{21}),
							},
						},
						Signature: hexutil.Encode([]byte{22}),
					},
				},
				{
					Attestation_1: &apimiddleware.IndexedAttestationJson{
						AttestingIndices: []string{"23", "24"},
						Data: &apimiddleware.AttestationDataJson{
							Slot:            "25",
							CommitteeIndex:  "26",
							BeaconBlockRoot: hexutil.Encode([]byte{27}),
							Source: &apimiddleware.CheckpointJson{
								Epoch: "28",
								Root:  hexutil.Encode([]byte{29}),
							},
							Target: &apimiddleware.CheckpointJson{
								Epoch: "30",
								Root:  hexutil.Encode([]byte{31}),
							},
						},
						Signature: hexutil.Encode([]byte{32}),
					},
					Attestation_2: &apimiddleware.IndexedAttestationJson{
						AttestingIndices: []string{"33", "34"},
						Data: &apimiddleware.AttestationDataJson{
							Slot:            "35",
							CommitteeIndex:  "36",
							BeaconBlockRoot: hexutil.Encode([]byte{37}),
							Source: &apimiddleware.CheckpointJson{
								Epoch: "38",
								Root:  hexutil.Encode([]byte{39}),
							},
							Target: &apimiddleware.CheckpointJson{
								Epoch: "40",
								Root:  hexutil.Encode([]byte{41}),
							},
						},
						Signature: hexutil.Encode([]byte{42}),
					},
				},
			},
		},
	).Times(1)

	slasherClient := beaconApiSlasherClient{jsonRestHandler: jsonRestHandler}

	expectedAttesterSlashingResponse := &ethpb.AttesterSlashingResponse{
		AttesterSlashings: []*ethpb.AttesterSlashing{
			{
				Attestation_1: &ethpb.IndexedAttestation{
					AttestingIndices: []uint64{1, 2, 3, 4},
					Data: &ethpb.AttestationData{
						Slot:            5,
						CommitteeIndex:  6,
						BeaconBlockRoot: []byte{7},
						Source: &ethpb.Checkpoint{
							Epoch: 8,
							Root:  []byte{9},
						},
						Target: &ethpb.Checkpoint{
							Epoch: 10,
							Root:  []byte{11},
						},
					},
					Signature: []byte{12},
				},
				Attestation_2: &ethpb.IndexedAttestation{
					AttestingIndices: []uint64{3, 4, 13, 14},
					Data: &ethpb.AttestationData{
						Slot:            15,
						CommitteeIndex:  16,
						BeaconBlockRoot: []byte{17},
						Source: &ethpb.Checkpoint{
							Epoch: 18,
							Root:  []byte{19},
						},
						Target: &ethpb.Checkpoint{
							Epoch: 10,
							Root:  []byte{21},
						},
					},
					Signature: []byte{22},
				},
			},
		},
	}

	attesterSlashingResponse, err := slasherClient.getSlashableAttestations(ctx, &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{13, 2, 14, 4, 3, 1, 23, 24, 33, 34},
		Data: &ethpb.AttestationData{
			Target: &ethpb.Checkpoint{
				Epoch: 10,
			},
		},
	})

	require.NoError(t, err)
	assert.DeepEqual(t, expectedAttesterSlashingResponse, attesterSlashingResponse)
}
