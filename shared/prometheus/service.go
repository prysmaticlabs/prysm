package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "prometheus")

// Service provides Prometheus metrics via the /metrics route. This route will
// show all the metrics registered with the Prometheus DefaultRegisterer.
type Service struct {
	server      *http.Server
	svcRegistry *shared.ServiceRegistry
}

// NewPrometheusService sets up a new instance for a given address host:port.
// An empty host will match with any IP so an address like ":2121" is perfectly acceptable.
func NewPrometheusService(addr string, svcRegistry *shared.ServiceRegistry) *Service {
	s := &Service{svcRegistry: svcRegistry}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", s.healthzHandler)

	s.server = &http.Server{Addr: addr, Handler: mux}

	return s
}

func (s *Service) healthzHandler(w http.ResponseWriter, _ *http.Request) {
	// Call all services in the registry.
	// if any are not OK, write 500
	// print the statuses of all services.

	statuses := s.svcRegistry.Statuses()
	hasError := false
	var buf bytes.Buffer
	for k, v := range statuses {
		var status string
		if v == nil {
			status = "OK"
		} else {
			hasError = true
			status = "ERROR " + v.Error()
		}

		if _, err := buf.WriteString(fmt.Sprintf("%s: %s\n", k, status)); err != nil {
			hasError = true
			status = "ERROR " + err.Error()
		}
	}

	// Write status header
	if hasError {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Write http body
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Errorf("Could not write healthz body %v", err)
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

// Status always returns nil.
// TODO(1207): Add service health checks.
func (s *Service) Status() error {
	return nil
}
