package slasher

import (
	"context"
)

// Receive beacon blocks from some source event feed,
func (s *Service) broadcastAttSlashings(ctx context.Context) {
	sub := s.serviceCfg.AttSlashingsFeed.Subscribe(s.attSlashingsChan)
	defer close(s.attSlashingsChan)
	defer sub.Unsubscribe()
	for {
		select {
		case _ = <-s.attSlashingsChan:
			// TODO hookup to beacon chain here
		case err := <-sub.Err():
			log.WithError(err).Debug("Attester slashing subscriber closed with error")
			return
		case <-ctx.Done():
			return
		}
	}
}

// Receive beacon blocks from some source event feed,
func (s *Service) broadcastBlockSlashings(ctx context.Context) {
	sub := s.serviceCfg.BlockSlashingsFeed.Subscribe(s.blockSlashingsChan)
	defer close(s.blockSlashingsChan)
	defer sub.Unsubscribe()
	for {
		select {
		case _ = <-s.blockSlashingsChan:
			// TODO hookup to beacon chain here
		case err := <-sub.Err():
			log.WithError(err).Debug("Proposer slashing subscriber closed with error")
			return
		case <-ctx.Done():
			return
		}
	}
}
