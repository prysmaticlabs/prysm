package utils

import (
	log "github.com/sirupsen/logrus"
)

// HandleServiceErrors manages a goroutine that listens for errors broadcast to
// this service's error channel. This serves as a final step for error logging
// and is stopped upon the service shutting down.
func HandleServiceErrors(done <-chan struct{}, errChan <-chan error) {
	for {
		select {
		case <-done:
			return
		case err := <-errChan:
			log.Error(err.Error())
		}
	}
}
