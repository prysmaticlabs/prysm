package archiver

import "github.com/prysmaticlabs/prysm/beacon-chain/db"

type Service struct {
	beaconDB db.Database
}

func (s *Service) Start() {
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() error {
	return nil
}
