package geninit

import (
	"context"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"go.opencensus.io/trace"
	"time"

	"github.com/prysmaticlabs/prysm/runtime"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	ctx context.Context
	powWaiter ClockWaiter
	genesisSetter ClockSetter
	powFetcher powchain.ChainStartFetcher
	d db.HeadAccessDatabase
}
var _ runtime.Service = &Service{}

type ServiceOption func(*Service)

func New(ctx context.Context, pw ClockWaiter, gs ClockSetter, f powchain.ChainStartFetcher, d db.HeadAccessDatabase, opts ...ServiceOption) (*Service, error) {
	s := &Service{
		ctx: ctx,
		powWaiter: pw,
		genesisSetter: gs,
		powFetcher: f,
		d: d,
	}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

func (s *Service) Start() {
	go s.run()
}

func (s *Service) run() {
	c, err := s.powWaiter.WaitForClock(s.ctx)
	if err != nil {
		log.WithError(err).Error("timeout waiting for genesis timestamp")
	}
	if err = s.saveGenesis(c); err != nil {
		log.Fatalf("Could not initialize beacon chain, %s", err.Error())
	}
	s.genesisSetter.SetGenesisClock(c)
}

func (s *Service) saveGenesis(c Clock) error {
	ctx, span := trace.StartSpan(s.ctx, "beacon-chain.geninit.Service.saveGenesis")
	defer span.End()
	eth1 := s.powFetcher.ChainStartEth1Data()
	pst := s.powFetcher.PreGenesisState()
	st, err := transition.OptimizedGenesisBeaconState(uint64(c.GenesisTime().Unix()), pst, eth1)
	if err != nil {
		return err
	}
	if err := s.d.SaveGenesisData(ctx, st); err != nil {
		return errors.Wrap(err, "db error, could not save genesis data")
	}
	log.Info("Initialized beacon chain genesis state")
	// Clear out all pre-genesis data now that the state is initialized.
	s.powFetcher.ClearPreGenesisData()
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() error {
	return nil
}

type GenesisReady struct {
	time time.Time
}
