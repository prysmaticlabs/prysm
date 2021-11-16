package monitor

import (
	"bytes"
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func setupService(t *testing.T) *Service {
	beaconDB := testDB.SetupDB(t)

	trackedVals := map[types.ValidatorIndex]interface{}{
		1:  nil,
		2:  nil,
		12: nil,
	}
	latestPerformance := map[types.ValidatorIndex]ValidatorLatestPerformance{
		1: {
			balance: 32000000000,
		},
		2: {
			balance: 32000000000,
		},
		12: {
			balance: 31900000000,
		},
	}

	aggregatedPerformance := map[types.ValidatorIndex]ValidatorAggregatedPerformance{
		1:  {},
		2:  {},
		12: {},
	}

	return &Service{
		config: &ValidatorMonitorConfig{
			StateGen:          stategen.New(beaconDB),
			TrackedValidators: trackedVals,
		},
		ctx:                   context.Background(),
		latestPerformance:     latestPerformance,
		aggregatedPerformance: aggregatedPerformance,
	}
}

func TestGetAttestingIndices(t *testing.T) {
	s := setupService(t)
	beaconState, _ := util.DeterministicGenesisState(t, 256)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 0,
		},
		AggregationBits: bitfield.Bitlist{0b11, 0b1},
	}
	attestingIndices, err := attestingIndices(s.ctx, beaconState, att)
	require.NoError(t, err)
	require.DeepEqual(t, attestingIndices, []uint64{0xc, 0x2})

}

func TestProcessIncludedAttestationTwoTracked(t *testing.T) {
	hook := logTest.NewGlobal()
	s := setupService(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 256)
	require.NoError(t, state.SetSlot(2))
	require.NoError(t, state.SetCurrentParticipationBits(bytes.Repeat([]byte{0xff}, 13)))

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			CommitteeIndex:  0,
			BeaconBlockRoot: bytesutil.PadTo([]byte("hello-world"), 32),
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("hello-world"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("hello-world"), 32),
			},
		},
		AggregationBits: bitfield.Bitlist{0b11, 0b1},
	}
	s.processIncludedAttestation(state, att)
	wanted1 := "\"Attestation included\" BalanceChange=0 CorrectHead=true CorrectSource=true CorrectTarget=true Head=0x68656c6c6f2d Inclusion Slot=2 NewBalance=32000000000 Slot=1 Source=0x68656c6c6f2d Target=0x68656c6c6f2d ValidatorIndex=2 prefix=monitor"
	wanted2 := "\"Attestation included\" BalanceChange=100000000 CorrectHead=true CorrectSource=true CorrectTarget=true Head=0x68656c6c6f2d Inclusion Slot=2 NewBalance=32000000000 Slot=1 Source=0x68656c6c6f2d Target=0x68656c6c6f2d ValidatorIndex=12 prefix=monitor"
	require.LogsContain(t, hook, wanted1)
	require.LogsContain(t, hook, wanted2)
}

func TestProcessUnaggregatedAttestationStateNotCached(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	hook := logTest.NewGlobal()

	s := setupService(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 256)
	require.NoError(t, state.SetSlot(2))
	header := state.LatestBlockHeader()
	participation := []byte{0xff, 0xff, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	require.NoError(t, state.SetCurrentParticipationBits(participation))

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			CommitteeIndex:  0,
			BeaconBlockRoot: header.GetStateRoot(),
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("hello-world"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("hello-world"), 32),
			},
		},
		AggregationBits: bitfield.Bitlist{0b11, 0b1},
	}
	s.processUnaggregatedAttestation(att)
	require.LogsContain(t, hook, "Skipping unnagregated attestation due to state not found in cache")
	logrus.SetLevel(logrus.InfoLevel)
}

func TestProcessUnaggregatedAttestationStateCached(t *testing.T) {
	hook := logTest.NewGlobal()

	s := setupService(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 256)
	participation := []byte{0xff, 0xff, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	require.NoError(t, state.SetCurrentParticipationBits(participation))

	root := [32]byte{}
	copy(root[:], "hello-world")

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			CommitteeIndex:  0,
			BeaconBlockRoot: root[:],
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  root[:],
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root[:],
			},
		},
		AggregationBits: bitfield.Bitlist{0b11, 0b1},
	}
	require.NoError(t, s.config.StateGen.SaveState(s.ctx, root, state))
	s.processUnaggregatedAttestation(att)
	wanted1 := "\"Processed unaggregated attestation\" Head=0x68656c6c6f2d Slot=1 Source=0x68656c6c6f2d Target=0x68656c6c6f2d ValidatorIndex=2 prefix=monitor"
	wanted2 := "\"Processed unaggregated attestation\" Head=0x68656c6c6f2d Slot=1 Source=0x68656c6c6f2d Target=0x68656c6c6f2d ValidatorIndex=12 prefix=monitor"
	require.LogsContain(t, hook, wanted1)
	require.LogsContain(t, hook, wanted2)
}

func TestProcessAggregatedAttestationStateNotCached(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	hook := logTest.NewGlobal()

	s := setupService(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 256)
	require.NoError(t, state.SetSlot(2))
	header := state.LatestBlockHeader()
	participation := []byte{0xff, 0xff, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	require.NoError(t, state.SetCurrentParticipationBits(participation))

	att := &ethpb.AggregateAttestationAndProof{
		AggregatorIndex: 2,
		Aggregate: &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Slot:            1,
				CommitteeIndex:  0,
				BeaconBlockRoot: header.GetStateRoot(),
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  bytesutil.PadTo([]byte("hello-world"), 32),
				},
				Target: &ethpb.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("hello-world"), 32),
				},
			},
			AggregationBits: bitfield.Bitlist{0b11, 0b1},
		},
	}
	s.processAggregatedAttestation(att)
	require.LogsContain(t, hook, "\"Processed attestation aggregation\" ValidatorIndex=2 prefix=monitor")
	require.LogsContain(t, hook, "Skipping agregated attestation due to state not found in cache")
	logrus.SetLevel(logrus.InfoLevel)
}

func TestProcessAggregatedAttestationStateCached(t *testing.T) {
	hook := logTest.NewGlobal()

	s := setupService(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 256)
	participation := []byte{0xff, 0xff, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	require.NoError(t, state.SetCurrentParticipationBits(participation))

	root := [32]byte{}
	copy(root[:], "hello-world")

	att := &ethpb.AggregateAttestationAndProof{
		AggregatorIndex: 2,
		Aggregate: &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Slot:            1,
				CommitteeIndex:  0,
				BeaconBlockRoot: root[:],
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  root[:],
				},
				Target: &ethpb.Checkpoint{
					Epoch: 1,
					Root:  root[:],
				},
			},
			AggregationBits: bitfield.Bitlist{0b10, 0b1},
		},
	}

	require.NoError(t, s.config.StateGen.SaveState(s.ctx, root, state))
	s.processAggregatedAttestation(att)
	require.LogsContain(t, hook, "\"Processed attestation aggregation\" ValidatorIndex=2 prefix=monitor")
	require.LogsContain(t, hook, "\"Processed aggregated attestation\" Head=0x68656c6c6f2d Slot=1 Source=0x68656c6c6f2d Target=0x68656c6c6f2d ValidatorIndex=2 prefix=monitor")
	require.LogsDoNotContain(t, hook, "\"Processed aggregated attestation\" Head=0x68656c6c6f2d Slot=1 Source=0x68656c6c6f2d Target=0x68656c6c6f2d ValidatorIndex=12 prefix=monitor")
}

func TestProcessAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	s := setupService(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 256)
	require.NoError(t, state.SetSlot(2))
	require.NoError(t, state.SetCurrentParticipationBits(bytes.Repeat([]byte{0xff}, 13)))

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			CommitteeIndex:  0,
			BeaconBlockRoot: bytesutil.PadTo([]byte("hello-world"), 32),
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("hello-world"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("hello-world"), 32),
			},
		},
		AggregationBits: bitfield.Bitlist{0b11, 0b1},
	}

	block := &ethpb.BeaconBlockAltair{
		Slot: 2,
		Body: &ethpb.BeaconBlockBodyAltair{
			Attestations: []*ethpb.Attestation{att},
		},
	}

	wrappedBlock, err := wrapper.WrappedAltairBeaconBlock(block)
	require.NoError(t, err)
	s.processAttestations(state, wrappedBlock)
	wanted1 := "\"Attestation included\" BalanceChange=0 CorrectHead=true CorrectSource=true CorrectTarget=true Head=0x68656c6c6f2d Inclusion Slot=2 NewBalance=32000000000 Slot=1 Source=0x68656c6c6f2d Target=0x68656c6c6f2d ValidatorIndex=2 prefix=monitor"
	wanted2 := "\"Attestation included\" BalanceChange=100000000 CorrectHead=true CorrectSource=true CorrectTarget=true Head=0x68656c6c6f2d Inclusion Slot=2 NewBalance=32000000000 Slot=1 Source=0x68656c6c6f2d Target=0x68656c6c6f2d ValidatorIndex=12 prefix=monitor"
	require.LogsContain(t, hook, wanted1)
	require.LogsContain(t, hook, wanted2)

}
