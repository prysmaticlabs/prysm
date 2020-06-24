package streaming

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestStreamDuties_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	v.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	v.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			CommitteeIndex: 1,
		},
	}

	expected := errors.New("bad")

	client.EXPECT().StreamDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, expected)

	if err := v.StreamDuties(context.Background()); !strings.Contains(err.Error(), "bad") {
		t.Errorf("Bad error; want=%v got=%v", expected, err)
	}
}

func TestStreamDuties_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	resp := &ethpb.DutiesResponse{
		CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{
				AttesterSlot:   params.BeaconConfig().SlotsPerEpoch,
				ValidatorIndex: 200,
				CommitteeIndex: 100,
				Committee:      []uint64{0, 1, 2, 3},
				PublicKey:      []byte("testPubKey_1"),
				ProposerSlots:  []uint64{params.BeaconConfig().SlotsPerEpoch + 1},
			},
			{
				AttesterSlot:   params.BeaconConfig().SlotsPerEpoch,
				ValidatorIndex: 201,
				CommitteeIndex: 101,
				Committee:      []uint64{0, 1, 2, 3},
				PublicKey:      []byte("testPubKey_2"),
				ProposerSlots:  []uint64{params.BeaconConfig().SlotsPerEpoch + 2},
			},
		},
	}
	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	v.genesisTime = uint64(time.Now().Unix()) + 500
	v.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	stream := mock.NewMockBeaconNodeValidator_StreamDutiesClient(ctrl)
	client.EXPECT().StreamDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)
	ctx := context.Background()
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	stream.EXPECT().Recv().Return(
		resp,
		nil,
	)

	client.EXPECT().SubscribeCommitteeSubnets(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, nil)

	stream.EXPECT().Recv().Return(
		nil,
		io.EOF,
	)

	if err := v.StreamDuties(ctx); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}
	if v.dutiesByEpoch[0][0].ProposerSlots[0] != params.BeaconConfig().SlotsPerEpoch+1 {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			params.BeaconConfig().SlotsPerEpoch+1,
			v.dutiesByEpoch[0][0].ProposerSlots[0],
		)
	}
	if v.dutiesByEpoch[0][0].AttesterSlot != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			params.BeaconConfig().SlotsPerEpoch,
			v.dutiesByEpoch[0][0].AttesterSlot,
		)
	}
	if v.dutiesByEpoch[0][0].CommitteeIndex != resp.CurrentEpochDuties[0].CommitteeIndex {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			resp.Duties[0].CommitteeIndex,
			v.dutiesByEpoch[0][0].CommitteeIndex,
		)
	}
	if v.dutiesByEpoch[0][0].ValidatorIndex != resp.CurrentEpochDuties[0].ValidatorIndex {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			resp.CurrentEpochDuties[0].ValidatorIndex,
			v.dutiesByEpoch[0][0].ValidatorIndex,
		)
	}
	if v.dutiesByEpoch[0][1].ValidatorIndex != resp.CurrentEpochDuties[1].ValidatorIndex {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			resp.CurrentEpochDuties[1].ValidatorIndex,
			v.dutiesByEpoch[0][1].ValidatorIndex,
		)
	}
}
