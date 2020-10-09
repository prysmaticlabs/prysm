package web

import (
	"context"
	"net/http"
	"time"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/browser"
	"github.com/sirupsen/logrus"
)

var (
	_   = shared.Service(&Server{})
	log = logrus.WithField("prefix", "prysm-web")
)

// Server for the Prysm Web UI.
type Server struct {
	http *http.Server
}

// NewServer creates a server service for the Prysm web UI.
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
		log.WithField("address", s.http.Addr).Info("Starting Prysm web UI")
		if err := s.http.ListenAndServe(); err != nil {
			log.WithError(err).Error("Failed to start validator web server")
		}
	}()
	time.Sleep(time.Second * 1)
	cmd, err := browser.Command("http://" + s.http.Addr)
	if err != nil {
		log.WithError(err).Errorf("Could not open Prysm web UI in browser")
		return
	}
	if err := cmd.Run(); err != nil {
		log.WithError(err).Errorf("Could not open Prysm web UI in browser")
		return
	}
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
