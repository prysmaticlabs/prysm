package components

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
)

// TracingSink to capture HTTP requests from opentracing pushes.
type TracingSink struct {
	started  chan struct{}
	endpoint string
	server   *http.Server
}

// NewTracingSink --
func NewTracingSink(endpoint string) *TracingSink {
	return &TracingSink{
		started:  make(chan struct{}, 1),
		endpoint: endpoint,
	}
}

// Start the tracing sink.
func (ts *TracingSink) Start(ctx context.Context) error {
	go ts.initializeSink()
	close(ts.started)
	return nil
}

// Started checks whether a tracing sink is started and ready to be queried.
func (ts *TracingSink) Started() <-chan struct{} {
	return ts.started
}

// Initialize an http handler that writes all requests to a file.
func (ts *TracingSink) initializeSink() {
	ts.server = &http.Server{Addr: ts.endpoint}
	defer func() {
		if err := ts.server.Close(); err != nil {
			log.WithError(err).Error("Failed to close http server")
			return
		}
	}()
	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, e2e.TracingRequestSinkFileName)
	if err != nil {
		log.WithError(err).Error("Failed to create stdout file")
		return
	}
	gzout := gzip.NewWriter(stdOutFile)
	cleanup := func() {
		if err := gzout.Close(); err != nil {
			log.WithError(err).Error("Could not close gzip")
		}
		if err := stdOutFile.Close(); err != nil {
			log.WithError(err).Error("Could not close stdout file")
		}
	}

	http.HandleFunc("/", func(_ http.ResponseWriter, r *http.Request) {
		if err := captureRequest(stdOutFile, r); err != nil {
			log.WithError(err).Error("Failed to capture http request")
			return
		}
	})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cleanup()
		os.Exit(0)
	}()
	if err := ts.server.ListenAndServe(); err != http.ErrServerClosed {
		log.WithError(err).Error("Failed to serve http")
	}
}

// Captures raw requests in base64 encoded form in a line-delimited file.
func captureRequest(f io.Writer, r *http.Request) error {
	buf := bytes.NewBuffer(nil)
	err := r.Write(buf)
	if err != nil {
		return err
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(buf.Bytes())))
	base64.StdEncoding.Encode(encoded, buf.Bytes())
	encoded = append(encoded, []byte("\n")...)
	_, err = f.Write(encoded)
	if err != nil {
		return err
	}
	return nil
}
