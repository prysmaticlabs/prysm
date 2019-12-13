package service

import (
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"

	ptypes "github.com/gogo/protobuf/types"
)

// finalisedChangeUpdater this is a stub for the comming PRs #3133
// Store validator index to public key map Validate attestation signature.
func (s *Service) finalisedChangeUpdater() {
	secondsPerSlot := params.BeaconConfig().SecondsPerSlot
	d := time.Duration(secondsPerSlot) * time.Second
	tick := time.Tick(d)
	var finalizedEpoch uint64
	for {
		select {
		case <-tick:
			ch, err := s.beaconClient.GetChainHead(s.context, &ptypes.Empty{})
			if err != nil {
				log.Error(err)
				break
			}
			if ch != nil {
				if ch.FinalizedEpoch > finalizedEpoch {
					log.Infof("Finalized epoch %d", ch.FinalizedEpoch)
				}
				continue
			}
			log.Error("No chain head was returned by beacon chain.")
		}

	}

}
