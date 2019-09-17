package archiver

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

type Service struct {
	beaconDB db.Database
}

type Config struct {
	BeaconDB db.Database
}

func NewArchiverService(ctx context.Context, cfg *Config) *Service {
	return &Service{
		beaconDB: cfg.BeaconDB,
	}
}

func (s *Service) Start() {
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() error {
	return nil
}
