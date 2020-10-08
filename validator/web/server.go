package web

import (
	"context"
	"net/http"
	"time"

	"github.com/prysmaticlabs/prysm/shared"
	log "github.com/sirupsen/logrus"
)

var _ = shared.Service(&Server{})

type Server struct {
	http *http.Server
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandler)

	return &Server{
		http: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Start the web server.
func (s *Server) Start() {
	go func() {
		log.Info("Running prysm web UI at http://localhost:4242") // TODO: use real URL.
		if err := s.http.ListenAndServe(); err != nil {
			log.WithError(err).Error("Failed to start validator web server")
		}
	}()
}

// Stop the web server gracefully with 1s timeout.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return s.http.Shutdown(ctx)
}

// Status check for web server. Always returns nil.
func (s *Server) Status() error {
	return nil
}
