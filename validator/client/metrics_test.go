package client

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestUpdateLogAggregateStats(t *testing.T) {
	v := &validator{
		logValidatorBalances: true,
		startBalances:        make(map[[48]byte]uint64),
		prevBalance:          make(map[[48]byte]uint64),
		voteStats: voteStats{
			startEpoch: 0, // this would otherwise have been previously set in LogValidatorGainsAndLosses()
		},
	}

	pubKeyBytes := [][48]byte{
		bytesutil.ToBytes48([]byte("000000000000000000000000000000000000000012345678")),
		bytesutil.ToBytes48([]byte("000000000000000000000000000000000000000099999999")),
		bytesutil.ToBytes48([]byte("000000000000000000000000000000000000000055555555")),
	}

	v.startBalances[pubKeyBytes[0]] = uint64(32100000000)
	v.startBalances[pubKeyBytes[1]] = uint64(32200000000)
	v.startBalances[pubKeyBytes[2]] = uint64(33000000000)

	responses := []*ethpb.ValidatorPerformanceResponse{
		{
			PublicKeys: [][]byte{
				bytesutil.FromBytes48(pubKeyBytes[0]),
				bytesutil.FromBytes48(pubKeyBytes[1]),
				bytesutil.FromBytes48(pubKeyBytes[2]),
			},
			InclusionSlots:       []types.Slot{types.Slot(^uint64(0)), 10, 11}, // exact slot doesn't matter, only if it is == or != ^uint64(0)
			InclusionDistances:   []types.Slot{0, 5, 2},
			CorrectlyVotedHead:   []bool{false, true, false},
			CorrectlyVotedSource: []bool{false, true, true},
			CorrectlyVotedTarget: []bool{false, true, true},
		},
		{
			PublicKeys: [][]byte{
				bytesutil.FromBytes48(pubKeyBytes[0]),
				bytesutil.FromBytes48(pubKeyBytes[1]),
				bytesutil.FromBytes48(pubKeyBytes[2]),
			},
			InclusionSlots:       []types.Slot{33, 34, 35},
			InclusionDistances:   []types.Slot{1, 2, 3},
			CorrectlyVotedHead:   []bool{true, true, true},
			CorrectlyVotedSource: []bool{true, true, true},
			CorrectlyVotedTarget: []bool{true, true, true},
		},
		{
			PublicKeys: [][]byte{
				bytesutil.FromBytes48(pubKeyBytes[0]),
				bytesutil.FromBytes48(pubKeyBytes[1]),
				bytesutil.FromBytes48(pubKeyBytes[2]),
			},
			InclusionSlots:       []types.Slot{65, types.Slot(^uint64(0)), 67},
			InclusionDistances:   []types.Slot{1, 0, 2},
			CorrectlyVotedHead:   []bool{true, false, true},
			CorrectlyVotedSource: []bool{true, false, true},
			CorrectlyVotedTarget: []bool{false, false, true},
		},
	}

	v.prevBalance[pubKeyBytes[0]] = uint64(33200000000)
	v.prevBalance[pubKeyBytes[1]] = uint64(33300000000)
	v.prevBalance[pubKeyBytes[2]] = uint64(31000000000)

	var hook *logTest.Hook

	for i, val := range responses {
		if i == len(responses)-1 { // Handle last log.
			hook = logTest.NewGlobal()
		}
		v.UpdateLogAggregateStats(val, types.Slot(params.BeaconConfig().SlotsPerEpoch*types.Slot(i+1)))
	}

	require.LogsContain(t, hook, "msg=\"Previous epoch aggregated voting summary\" attestationInclusionPct=\"67%\" "+
		"correctlyVotedHeadPct=\"100%\" correctlyVotedSourcePct=\"100%\" correctlyVotedTargetPct=\"50%\" epoch=2")
	require.LogsContain(t, hook, "msg=\"Vote summary since launch\" attestationsInclusionPct=\"78%\" "+
		"averageInclusionDistance=\"2.29 slots\" correctlyVotedHeadPct=\"86%\" correctlyVotedSourcePct=\"100%\" "+
		"correctlyVotedTargetPct=\"86%\" numberOfEpochs=3 pctChangeCombinedBalance=\"0.20555%\"")

}

func TestUpdateLogAltairAggregateStats(t *testing.T) {
	v := &validator{
		logValidatorBalances: true,
		startBalances:        make(map[[48]byte]uint64),
		prevBalance:          make(map[[48]byte]uint64),
		voteStats: voteStats{
			startEpoch: params.BeaconConfig().AltairForkEpoch, // this would otherwise have been previously set in LogValidatorGainsAndLosses()
		},
	}

	pubKeyBytes := [][48]byte{
		bytesutil.ToBytes48([]byte("000000000000000000000000000000000000000012345678")),
		bytesutil.ToBytes48([]byte("000000000000000000000000000000000000000099999999")),
		bytesutil.ToBytes48([]byte("000000000000000000000000000000000000000055555555")),
	}

	v.startBalances[pubKeyBytes[0]] = uint64(32100000000)
	v.startBalances[pubKeyBytes[1]] = uint64(32200000000)
	v.startBalances[pubKeyBytes[2]] = uint64(33000000000)

	// 7 attestations included
	responses := []*ethpb.ValidatorPerformanceResponse{
		{
			PublicKeys: [][]byte{
				bytesutil.FromBytes48(pubKeyBytes[0]),
				bytesutil.FromBytes48(pubKeyBytes[1]),
				bytesutil.FromBytes48(pubKeyBytes[2]),
			},
			CorrectlyVotedHead:   []bool{false, true, false},
			CorrectlyVotedSource: []bool{false, true, true},
			CorrectlyVotedTarget: []bool{false, false, true}, // test one fast target that is included
		},
		{
			PublicKeys: [][]byte{
				bytesutil.FromBytes48(pubKeyBytes[0]),
				bytesutil.FromBytes48(pubKeyBytes[1]),
				bytesutil.FromBytes48(pubKeyBytes[2]),
			},
			CorrectlyVotedHead:   []bool{true, true, true},
			CorrectlyVotedSource: []bool{true, true, true},
			CorrectlyVotedTarget: []bool{true, true, true},
		},
		{
			PublicKeys: [][]byte{
				bytesutil.FromBytes48(pubKeyBytes[0]),
				bytesutil.FromBytes48(pubKeyBytes[1]),
				bytesutil.FromBytes48(pubKeyBytes[2]),
			},
			CorrectlyVotedHead:   []bool{true, false, true},
			CorrectlyVotedSource: []bool{true, false, true},
			CorrectlyVotedTarget: []bool{false, false, true},
		},
	}

	v.prevBalance[pubKeyBytes[0]] = uint64(33200000000)
	v.prevBalance[pubKeyBytes[1]] = uint64(33300000000)
	v.prevBalance[pubKeyBytes[2]] = uint64(31000000000)

	var hook *logTest.Hook

	for i, val := range responses {
		if i == len(responses)-1 { // Handle last log.
			hook = logTest.NewGlobal()
		}
		altairStart, err := slots.EpochStart(params.BeaconConfig().AltairForkEpoch)
		require.NoError(t, err)

		v.UpdateLogAggregateStats(val, altairStart+params.BeaconConfig().SlotsPerEpoch*types.Slot(i+1))
	}

	require.LogsContain(t, hook, "msg=\"Previous epoch aggregated voting summary\" attestationInclusionPct=\"67%\" "+
		"averageInactivityScore=0 correctlyVotedHeadPct=\"100%\" correctlyVotedSourcePct=\"100%\" correctlyVotedTargetPct=\"50%\" epoch=74242")
	require.LogsContain(t, hook, "msg=\"Vote summary since launch\" attestationsInclusionPct=\"78%\" "+
		"correctlyVotedHeadPct=\"86%\" correctlyVotedSourcePct=\"100%\" "+
		"correctlyVotedTargetPct=\"71%\" numberOfEpochs=3 pctChangeCombinedBalance=\"0.20555%\"")
}
