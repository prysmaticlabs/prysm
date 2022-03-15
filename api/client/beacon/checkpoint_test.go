package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/testing/require"
)

type testRT struct {
	rt func(*http.Request) (*http.Response, error)
}

func (rt *testRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.rt != nil {
		return rt.rt(req)
	}
	return nil, errors.New("RoundTripper not implemented")
}

var _ http.RoundTripper = &testRT{}

func marshalToEnvelope(val interface{}) ([]byte, error) {
	raw, err := json.Marshal(val)
	if err != nil {
		return nil, errors.Wrap(err, "error marsaling value to place in data envelope")
	}
	env := struct {
		Data json.RawMessage `json:"data"`
	}{
		Data: raw,
	}
	return json.Marshal(env)
}

func TestMarshalToEnvelope(t *testing.T) {
	d := struct {
		Version string `json:"version"`
	}{
		Version: "Prysm/v2.0.5 (linux amd64)",
	}
	encoded, err := marshalToEnvelope(d)
	require.NoError(t, err)
	expected := `{"data":{"version":"Prysm/v2.0.5 (linux amd64)"}}`
	require.Equal(t, expected, string(encoded))
}

func TestFallbackVersionCheck(t *testing.T) {
	c := &Client{
		hc:     &http.Client{},
		host:   "localhost:3500",
		scheme: "http",
	}
	c.hc.Transport = &testRT{rt: func(req *http.Request) (*http.Response, error) {
		res := &http.Response{Request: req}
		switch req.URL.Path {
		case get_node_version_path:
			res.StatusCode = http.StatusOK
			b := bytes.NewBuffer(nil)
			d := struct {
				Version string `json:"version"`
			}{
				Version: "Prysm/v2.0.5 (linux amd64)",
			}
			encoded, err := marshalToEnvelope(d)
			require.NoError(t, err)
			b.Write(encoded)
			res.Body = io.NopCloser(b)
		case get_weak_subjectivity_path:
			res.StatusCode = http.StatusNotFound
		}

		return res, nil
	}}

	ctx := context.Background()
	_, err := DownloadOriginData(ctx, c)
	require.ErrorIs(t, err, ErrUnsupportedPrysmCheckpointVersion)
}
