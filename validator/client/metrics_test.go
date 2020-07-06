package client

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestUpdateLogAggregateStats(t *testing.T) {
	v := &validator{
		logValidatorBalances: true,
		startBalances:        make(map[[48]byte]uint64),
		prevBalance:          make(map[[48]byte]uint64),
		voteStats: voteStats{
			startEpoch: 0, //this would otherwise have been previously set in LogValidatorGainsAndLosses()
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
			InclusionSlots:       []uint64{^uint64(0), 10, 11}, //exact slot doesn't matter, only if it is == or != ^uint64(0)
			InclusionDistances:   []uint64{0, 5, 2},
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
			InclusionSlots:       []uint64{33, 34, 35},
			InclusionDistances:   []uint64{1, 2, 3},
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
			InclusionSlots:       []uint64{65, ^uint64(0), 67},
			InclusionDistances:   []uint64{1, 0, 2},
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
		if i == len(responses)-1 { //check final log output
			hook = logTest.NewGlobal()
		}
		v.UpdateLogAggregateStats(val, uint64(32*(i+1)))
	}

	testutil.AssertLogsContain(t, hook, "msg=\"Previous epoch aggregated voting summary\" attestationInclusionPct=\"67%\" "+
		"correctlyVotedHeadPct=\"100%\" correctlyVotedSourcePct=\"100%\" correctlyVotedTargetPct=\"50%\" epoch=2")
	testutil.AssertLogsContain(t, hook, "msg=\"Vote summary since launch\" attestationsInclusionPct=\"78%\" "+
		"averageInclusionDistance=\"2.29 slots\" correctlyVotedHeadPct=\"86%\" correctlyVotedSourcePct=\"100%\" "+
		"correctlyVotedTargetPct=\"86%\" numberOfEpochs=3 pctChangeCombinedBalance=\"0.20555%\"")

}
