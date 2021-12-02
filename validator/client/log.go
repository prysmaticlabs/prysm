package client

import (
	"fmt"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validator")

type attSubmitted struct {
	data              *ethpb.AttestationData
	attesterIndices   []types.ValidatorIndex
	aggregatorIndices []types.ValidatorIndex
}

// LogAttestationsSubmitted logs info about submitted attestations.
func (v *validator) LogAttestationsSubmitted() {
	v.attLogsLock.Lock()
	defer v.attLogsLock.Unlock()

	for _, attLog := range v.attLogs {
		log.WithFields(logrus.Fields{
			"Slot":              attLog.data.Slot,
			"CommitteeIndex":    attLog.data.CommitteeIndex,
			"BeaconBlockRoot":   fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.BeaconBlockRoot)),
			"SourceEpoch":       attLog.data.Source.Epoch,
			"SourceRoot":        fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.Source.Root)),
			"TargetEpoch":       attLog.data.Target.Epoch,
			"TargetRoot":        fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.Target.Root)),
			"AttesterIndices":   attLog.attesterIndices,
			"AggregatorIndices": attLog.aggregatorIndices,
		}).Info("Submitted new attestations")
	}

	v.attLogs = make(map[[32]byte]*attSubmitted)
}

// LogNextDutyTimeLeft logs the next duty info.
func (v *validator) LogNextDutyTimeLeft(slot types.Slot) error {
	if !v.logDutyCountDown {
		return nil
	}
	if v.duties == nil {
		return nil
	}

	var nextDutySlot types.Slot
	attestingCounts := make(map[types.Slot]uint64)
	proposingCounts := make(map[types.Slot]uint64)
	for _, duty := range v.duties.CurrentEpochDuties {
		attestingCounts[duty.AttesterSlot]++

		if duty.AttesterSlot > slot && (nextDutySlot > duty.AttesterSlot || nextDutySlot == 0) {
			nextDutySlot = duty.AttesterSlot
		}
		for _, proposerSlot := range duty.ProposerSlots {
			proposingCounts[proposerSlot]++

			if proposerSlot > slot && (nextDutySlot > proposerSlot || nextDutySlot == 0) {
				nextDutySlot = proposerSlot
			}
		}
	}

	if nextDutySlot == 0 {
		log.WithField("slotInEpoch", slot%params.BeaconConfig().SlotsPerEpoch).Info("No duty until next epoch")
	} else {
		nextDutyTime, err := slots.ToTime(v.genesisTime, nextDutySlot)
		if err != nil {
			return err
		}
		timeLeft := time.Duration(nextDutyTime.Unix() - prysmTime.Now().Unix()).Nanoseconds()
		// There is not much value to log if time left is less than one slot.
		if uint64(timeLeft) >= params.BeaconConfig().SecondsPerSlot {
			log.WithFields(
				logrus.Fields{
					"currentSlot": slot,
					"dutySlot":    nextDutySlot,
					"attesting":   attestingCounts[nextDutySlot],
					"proposing":   proposingCounts[nextDutySlot],
					"slotInEpoch": slot % params.BeaconConfig().SlotsPerEpoch,
					"secondsLeft": timeLeft,
				}).Info("Next duty")
		}
	}

	return nil
}
