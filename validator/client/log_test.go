package client

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLogNextDutyCountDown_NoDuty(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &validator{
		logDutyCountDown: true,
		duties: &ethpb.DutiesResponse{CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{AttesterSlot: 100, ProposerSlots: []types.Slot{105}},
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
		logDutyCountDown: true,
		duties: &ethpb.DutiesResponse{CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{AttesterSlot: 100, ProposerSlots: []types.Slot{105}},
			{AttesterSlot: 110},
			{AttesterSlot: 120},
		}},
	}
	require.NoError(t, v.LogNextDutyTimeLeft(115))
	require.LogsContain(t, hook, "\"Next duty\" attesting=1 currentSlot=115 dutySlot=120 prefix=validator proposing=0")
}

func TestLogNextDutyCountDown_HasDutyProposer(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &validator{
		logDutyCountDown: true,
		duties: &ethpb.DutiesResponse{CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{AttesterSlot: 100, ProposerSlots: []types.Slot{105}},
			{AttesterSlot: 110},
			{AttesterSlot: 120},
		}},
	}
	require.NoError(t, v.LogNextDutyTimeLeft(101))
	require.LogsContain(t, hook, "\"Next duty\" attesting=0 currentSlot=101 dutySlot=105 prefix=validator proposing=1")
}

func TestLogNextDutyCountDown_HasMultipleDuties(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &validator{
		logDutyCountDown: true,
		duties: &ethpb.DutiesResponse{CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{AttesterSlot: 120},
			{AttesterSlot: 110},
			{AttesterSlot: 105},
			{AttesterSlot: 105},
			{AttesterSlot: 100, ProposerSlots: []types.Slot{105}},
		}},
	}
	require.NoError(t, v.LogNextDutyTimeLeft(101))
	require.LogsContain(t, hook, "\"Next duty\" attesting=2 currentSlot=101 dutySlot=105 prefix=validator proposing=1")
}

func TestLogNextDutyCountDown_NilDuty(t *testing.T) {
	v := &validator{
		logDutyCountDown: true,
	}
	require.NoError(t, v.LogNextDutyTimeLeft(101))
}
