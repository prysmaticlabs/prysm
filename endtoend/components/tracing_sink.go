package components

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, e2e.TracingRequestSinkFileName)
	if err != nil {
		log.WithError(err).Error("Failed to create stdout file")
		return
	}
	defer func() {
		if err = stdOutFile.Close(); err != nil {
			log.WithError(err).Error("Failed to close stdout file")
		}
	}()
	ts.server = &http.Server{Addr: ts.endpoint}
	defer func() {
		if err = ts.server.Close(); err != nil {
			log.WithError(err).Error("Failed to close http server")
		}
	}()

	http.HandleFunc("/", func(writer http.ResponseWriter, r *http.Request) {
		reqContent := map[string]interface{}{}
		if err = parseRequest(r, &reqContent); err != nil {
			log.WithError(err).Error("Failed to parse request")
			return
		}
		if err = captureRequest(stdOutFile, reqContent); err != nil {
			log.WithError(err).Error("Failed to capture http request")
		}
	})
	if err := ts.server.ListenAndServe(); err != http.ErrServerClosed {
		log.WithError(err).Error("Failed to serve http")
	}
}

func captureRequest(f io.StringWriter, m map[string]interface{}) error {
	enc, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("%s\n", enc))
	return err
}

func parseRequest(req *http.Request, unmarshalStruct interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if err = req.Body.Close(); err != nil {
		return err
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return json.Unmarshal(body, unmarshalStruct)
}
