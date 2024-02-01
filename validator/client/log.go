package client

import (
	"fmt"
	"sync/atomic"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validator")

type submittedAtt struct {
	data              *ethpb.AttestationData
	attesterPubkeys   [][]byte
	aggregatorPubkeys [][]byte
}

// LogSubmittedAtts logs info about submitted attestations.
func (v *validator) LogSubmittedAtts(slot primitives.Slot) {
	v.attLogsLock.Lock()
	defer v.attLogsLock.Unlock()

	if len(v.submittedAtts) == 0 {
		return
	}

	log.Infof("Submitted new attestations for slot %d", slot)
	for _, attLog := range v.submittedAtts {
		attesterPubkeys := make([]string, len(attLog.attesterPubkeys))
		for i, p := range attLog.attesterPubkeys {
			attesterPubkeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(p))
		}
		aggregatorPubkeys := make([]string, len(attLog.aggregatorPubkeys))
		for i, p := range attLog.aggregatorPubkeys {
			aggregatorPubkeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(p))
		}
		log.WithFields(logrus.Fields{
			"committeeIndex":    attLog.data.CommitteeIndex,
			"beaconBlockRoot":   fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.BeaconBlockRoot)),
			"sourceEpoch":       attLog.data.Source.Epoch,
			"sourceRoot":        fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.Source.Root)),
			"targetEpoch":       attLog.data.Target.Epoch,
			"targetRoot":        fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.Target.Root)),
			"attesterPubkeys":   attesterPubkeys,
			"aggregatorPubkeys": aggregatorPubkeys,
		}).Info("Submitted new attestations")
	}

	v.submittedAtts = make(map[[32]byte]*submittedAtt)
}

// LogSyncCommitteeMessagesSubmitted logs info about submitted sync committee messages.
func (v *validator) LogSubmittedSyncCommitteeMessages() {
	log.WithField("messages", v.syncCommitteeStats.totalMessagesSubmitted).Debug("Submitted sync committee messages successfully to beacon node")
	// Reset the amount.
	atomic.StoreUint64(&v.syncCommitteeStats.totalMessagesSubmitted, 0)
}
