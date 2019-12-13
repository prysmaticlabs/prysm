package service

import (
	"time"

	ptypes "github.com/gogo/protobuf/types"
)

func (s *Service) validatorUpdater() {
	tick := time.Tick(2000 * time.Millisecond)
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
					log.Infof("Finalized epoch %v", ch.FinalizedEpoch)
				}
				continue
			}
			log.Info("No chain head was returned by beacon chain.")
		}

	}

}
