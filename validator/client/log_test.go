package client

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLogNextDutyCountDown_NoDuty(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &validator{
		duties: &ethpb.DutiesResponse{CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{AttesterSlot: 100, ProposerSlots: []uint64{105}},
			{AttesterSlot: 110},
			{AttesterSlot: 120},
		}},
	}
	require.NoError(t, v.LogNextDutyTimeLeft(121))
	require.LogsContain(t, hook, "No duty until next epoch")
}

func TestLogNextDutyCountDown_HasDutyAttester(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &validator{
		duties: &ethpb.DutiesResponse{CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{AttesterSlot: 100, ProposerSlots: []uint64{105}},
			{AttesterSlot: 110},
			{AttesterSlot: 120},
		}},
	}
	require.NoError(t, v.LogNextDutyTimeLeft(115))
	require.LogsContain(t, hook, "\"Next duty\" currentSlot=115 dutySlot=120 prefix=validator role=attester")
}

func TestLogNextDutyCountDown_HasDutyProposer(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &validator{
		duties: &ethpb.DutiesResponse{CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{AttesterSlot: 100, ProposerSlots: []uint64{105}},
			{AttesterSlot: 110},
			{AttesterSlot: 120},
		}},
	}
	require.NoError(t, v.LogNextDutyTimeLeft(101))
	require.LogsContain(t, hook, "\"Next duty\" currentSlot=101 dutySlot=105 prefix=validator role=proposer")
}
