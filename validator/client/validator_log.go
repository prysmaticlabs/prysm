package client

import (
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
)

type attSubmitted struct {
	data    *ethpb.AttestationData
	indices []uint64
}

func (v *validator) LogAttestationsSubmitted() {
	v.attLogsLock.Lock()
	defer v.attLogsLock.Unlock()

	for _, attLog := range v.attLogs {
		log.WithFields(logrus.Fields{
			"Slot":             attLog.data.Slot,
			"CommitteeIndex":   attLog.data.CommitteeIndex,
			"BeaconBlockRoot":  fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.BeaconBlockRoot)),
			"SourceEpoch":      attLog.data.Source.Epoch,
			"SourceRoot":       fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.Source.Root)),
			"TargetEpoch":      attLog.data.Target.Epoch,
			"TargetRoot":       fmt.Sprintf("%#x", bytesutil.Trunc(attLog.data.Target.Root)),
			"ValidatorIndices": attLog.indices,
		}).Info("Submitted new unaggregated attestations")
	}

	v.attLogs = make(map[[32]byte]*attSubmitted)
}
