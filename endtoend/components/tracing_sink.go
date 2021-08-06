package components

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
)

// TracingSink to capture HTTP requests from opentracing pushes.
type TracingSink struct {
	started  chan struct{}
	endpoint string
	server   *http.Server
}

// TracingSink to capture HTTP requests from opentracing pushes.
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
	gzipFile := gzip.NewWriter(stdOutFile)
	defer func() {
		if err := gzipFile.Flush(); err != nil {
			log.WithError(err).Error("Failed to flush to file")
			return
		}
		if err = stdOutFile.Close(); err != nil {
			log.WithError(err).Error("Failed to close stdout file")
		}
	}()

	http.HandleFunc("/", func(_ http.ResponseWriter, r *http.Request) {
		if err := captureRequest(gzipFile, r); err != nil {
			log.WithError(err).Error("Failed to capture http request")
			return
		}
	})
	if err := ts.server.ListenAndServe(); err != http.ErrServerClosed {
		log.WithError(err).Error("Failed to serve http")
	}
}

func captureRequest(gzipFile io.Writer, r *http.Request) error {
	buf := new(bytes.Buffer)
	if err := r.Write(buf); err != nil {
		return err
	}
	encoded := new(bytes.Buffer)
	base64.StdEncoding.Encode(encoded.Bytes(), buf.Bytes())
	if _, err := encoded.WriteString("\n"); err != nil {
		return err
	}
	if _, err := gzipFile.Write(encoded.Bytes()); err != nil {
		return err
	}
	return nil
}
