package client

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/internal"
)

func TestStreamDuties_DoesNothingWhenNotEpochStart_AlreadyExistingAssignments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
		duties: &ethpb.DutiesResponse{
			CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
				{
					Committee:      []uint64{},
					AttesterSlot:   10,
					CommitteeIndex: 20,
				},
			},
		},
	}
	client.EXPECT().StreamDuties(
		gomock.Any(),
		gomock.Any(),
	).Times(0)

	if err := v.StreamDuties(context.Background()); err != nil {
		t.Errorf("Could not update assignments: %v", err)
	}
}

func TestStreamDuties_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
		duties: &ethpb.DutiesResponse{
			Duties: []*ethpb.DutiesResponse_Duty{
				{
					CommitteeIndex: 1,
				},
			},
		},
	}

	expected := errors.New("bad")

	client.EXPECT().GetDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, expected)

	if err := v.StreamDuties(context.Background()); err != expected {
		t.Errorf("Bad error; want=%v got=%v", expected, err)
	}
	if v.duties != nil {
		t.Error("Assignments should have been cleared on failure")
	}
}

func TestStreamDuties_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockBeaconNodeValidatorClient(ctrl)

	resp := &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				AttesterSlot:   params.BeaconConfig().SlotsPerEpoch,
				ValidatorIndex: 200,
				CommitteeIndex: 100,
				Committee:      []uint64{0, 1, 2, 3},
				PublicKey:      []byte("testPubKey_1"),
				ProposerSlots:  []uint64{params.BeaconConfig().SlotsPerEpoch + 1},
			},
		},
	}
	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	client.EXPECT().StreamDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	client.EXPECT().StreamDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	client.EXPECT().SubscribeCommitteeSubnets(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, nil)

	if err := v.StreamDuties(context.Background()); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}
	if v.duties.CurrentEpochDuties[0].ProposerSlots[0] != params.BeaconConfig().SlotsPerEpoch+1 {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			params.BeaconConfig().SlotsPerEpoch+1,
			v.duties.Duties[0].ProposerSlots[0],
		)
	}
	if v.duties.CurrentEpochDuties[0].AttesterSlot != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			params.BeaconConfig().SlotsPerEpoch,
			v.duties.Duties[0].AttesterSlot,
		)
	}
	if v.duties.CurrentEpochDuties[0].CommitteeIndex != resp.Duties[0].CommitteeIndex {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			resp.Duties[0].CommitteeIndex,
			v.duties.Duties[0].CommitteeIndex,
		)
	}
	if v.duties.CurrentEpochDuties[0].ValidatorIndex != resp.Duties[0].ValidatorIndex {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			resp.Duties[0].ValidatorIndex,
			v.duties.Duties[0].ValidatorIndex,
		)
	}
}
