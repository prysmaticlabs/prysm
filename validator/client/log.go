package client

import (
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"github.com/sirupsen/logrus"
)

type attSubmitted struct {
	data              *ethpb.AttestationData
	attesterIndices   []uint64
	aggregatorIndices []uint64
}

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

func (v *validator) LogNextDutyTimeLeft(slot uint64) error {
	var nextDutySlot uint64
	var role string
	for _, duty := range v.duties.CurrentEpochDuties {
		if duty.AttesterSlot > slot && (nextDutySlot > duty.AttesterSlot || nextDutySlot == 0) {
			nextDutySlot = duty.AttesterSlot
			role = "attester"
		}
		for _, proposerSlot := range duty.ProposerSlots {
			if proposerSlot > slot && (nextDutySlot > proposerSlot || nextDutySlot == 0) {
				nextDutySlot = proposerSlot
				role = "proposer"
			}
		}
	}

	if nextDutySlot == 0 {
		log.WithField("slotInEpoch", slot%params.BeaconConfig().SlotsPerEpoch).Info("No duty until next epoch")
	} else {
		nextDutyTime, err := helpers.SlotToTime(v.genesisTime, nextDutySlot)
		if err != nil {
			return err
		}
		timeLeft := time.Duration(nextDutyTime.Unix() - timeutils.Now().Unix()).Nanoseconds()
		// There is not much value to log if time left is less than one slot.
		if uint64(timeLeft) >= params.BeaconConfig().SecondsPerSlot {
			log.WithFields(
				logrus.Fields{
					"role":        role,
					"currentSlot": slot,
					"dutySlot":    nextDutySlot,
					"slotInEpoch": slot % params.BeaconConfig().SlotsPerEpoch,
					"secondsLeft": timeLeft,
				}).Info("Next duty")
		}
	}

	return nil
}
