package monitor

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProcessExitsFromBlockTrackedIndices(t *testing.T) {
	hook := logTest.NewGlobal()
	s := &Service{
		TrackedValidators: map[types.ValidatorIndex]bool{
			1: true,
			2: true,
		},
	}

	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 3,
				Epoch:          1,
			},
		},
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 2,
				Epoch:          0,
			},
		},
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	s.processExitsFromBlock(wrapper.WrappedPhase0BeaconBlock(block))
	require.LogsContain(t, hook, "\"Voluntary exit was included\" Slot=0 ValidatorIndex=2")
}

func TestProcessExitsFromBlockUntrackedIndices(t *testing.T) {
	hook := logTest.NewGlobal()
	s := &Service{
		TrackedValidators: map[types.ValidatorIndex]bool{
			1: true,
			2: true,
		},
	}

	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 3,
				Epoch:          1,
			},
		},
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 4,
				Epoch:          0,
			},
		},
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	s.processExitsFromBlock(wrapper.WrappedPhase0BeaconBlock(block))
	require.LogsDoNotContain(t, hook, "\"Voluntary exit was included\"")
}

func TestProcessExitP2PTrackedIndices(t *testing.T) {
	hook := logTest.NewGlobal()
	s := &Service{
		TrackedValidators: map[types.ValidatorIndex]bool{
			1: true,
			2: true,
		},
	}

	exit := &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			ValidatorIndex: 1,
			Epoch:          1,
		},
		Signature: make([]byte, 96),
	}
	s.processExit(exit)
	require.LogsContain(t, hook, "\"Voluntary exit was processed\" ValidatorIndex=1")
}

func TestProcessExitP2PUntrackedIndices(t *testing.T) {
	hook := logTest.NewGlobal()
	s := &Service{
		TrackedValidators: map[types.ValidatorIndex]bool{
			1: true,
			2: true,
		},
	}

	exit := &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			ValidatorIndex: 3,
			Epoch:          1,
		},
	}
	s.processExit(exit)
	require.LogsDoNotContain(t, hook, "\"Voluntary exit was processed\"")
}
