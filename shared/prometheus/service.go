package prometheus

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "prometheus")

// Service provides Prometheus metrics via the /metrics route. This route will
// show all the metrics registered with the Prometheus DefaultRegisterer.
type Service struct {
	server *http.Server
}

// NewPrometheusService sets up a new instance for a given address host:port.
// An empty host will match with any IP so an address like ":2121" is perfectly acceptable.
func NewPrometheusService(addr string) *Service {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &Service{
		server: &http.Server{Addr: addr, Handler: mux},
	}
}

// Start the prometheus service.
func (s *Service) Start() {
	log.WithField("endpoint", s.server.Addr).Info("Starting service")
	go func() {
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Errorf("Could not listen to host:port :%s: %v", s.server.Addr, err)
		}
	}()
}

// Stop the service gracefully.
func (s *Service) Stop() error {
	log.Info("Stopping service")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
