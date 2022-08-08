package geninit

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/runtime"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	ctx context.Context
	powWaiter ClockWaiter
	genesisSetter ClockSetter
}
var _ runtime.Service = &Service{}

type ServiceOption func(*Service)

func New(ctx context.Context, pw ClockWaiter, gs ClockSetter, opts ...ServiceOption) (*Service, error) {
	s := &Service{
		ctx: ctx,
		powWaiter: pw,
		genesisSetter: gs,
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
	s.genesisSetter.SetGenesisClock(c)
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
