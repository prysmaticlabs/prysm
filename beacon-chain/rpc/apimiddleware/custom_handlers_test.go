package apimiddleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/api/grpc"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/events"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/r3labs/sse"
)

type testSSZResponseJson struct {
	Version string `json:"version"`
	Data    string `json:"data"`
}

func (t testSSZResponseJson) SSZVersion() string {
	return t.Version
}

func (t testSSZResponseJson) SSZData() string {
	return t.Data
}

func TestSSZRequested(t *testing.T) {
	t.Run("ssz_requested", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{octetStreamMediaType}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("ssz_content_type_first", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s,%s", octetStreamMediaType, jsonMediaType)}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("ssz_content_type_preferred_1", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s;q=0.9,%s", jsonMediaType, octetStreamMediaType)}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("ssz_content_type_preferred_2", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s;q=0.95,%s;q=0.9", octetStreamMediaType, jsonMediaType)}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("other_content_type_preferred", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s,%s;q=0.9", jsonMediaType, octetStreamMediaType)}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("other_params", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s,%s;q=0.9,otherparam=xyz", jsonMediaType, octetStreamMediaType)}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("no_header", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("empty_header", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("empty_header_value", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{""}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("other_content_type", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{"application/other"}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("garbage", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{"This is Sparta!!!"}
		result, err := sszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})
}

func TestPrepareSSZRequestForProxying(t *testing.T) {
	middleware := &apimiddleware.ApiProxyMiddleware{
		GatewayAddress: "http://apimiddleware.example",
	}
	endpoint := apimiddleware.Endpoint{
		Path: "http://foo.example",
	}
	var body bytes.Buffer
	request := httptest.NewRequest("GET", "http://foo.example", &body)

	errJson := prepareSSZRequestForProxying(middleware, endpoint, request)
	require.Equal(t, true, errJson == nil)
	assert.Equal(t, "/internal/ssz", request.URL.Path)
}

func TestPreparePostedSszData(t *testing.T) {
	var body bytes.Buffer
	body.Write([]byte("body"))
	request := httptest.NewRequest("POST", "http://foo.example", &body)

	preparePostedSSZData(request)
	assert.Equal(t, int64(19), request.ContentLength)
	assert.Equal(t, jsonMediaType, request.Header.Get("Content-Type"))
}

func TestSerializeMiddlewareResponseIntoSSZ(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		j := testSSZResponseJson{
			Version: "Version",
			Data:    "Zm9v",
		}
		v, ssz, errJson := serializeMiddlewareResponseIntoSSZ(j)
		require.Equal(t, true, errJson == nil)
		assert.DeepEqual(t, []byte("foo"), ssz)
		assert.Equal(t, "version", v)
	})

	t.Run("invalid_data", func(t *testing.T) {
		j := testSSZResponseJson{
			Version: "Version",
			Data:    "invalid",
		}
		_, _, errJson := serializeMiddlewareResponseIntoSSZ(j)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode response body into base64"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWriteSSZResponseHeaderAndBody(t *testing.T) {
	responseSsz := []byte("ssz")
	version := "version"
	fileName := "test.ssz"

	t.Run("ok", func(t *testing.T) {
		response := &http.Response{
			Header: http.Header{
				"Foo": []string{"foo"},
				"Grpc-Metadata-" + grpc.HttpCodeMetadataKey: []string{"204"},
			},
		}

		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		errJson := writeSSZResponseHeaderAndBody(response, writer, responseSsz, version, fileName)
		require.Equal(t, true, errJson == nil)
		v, ok := writer.Header()["Foo"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "foo", v[0])
		v, ok = writer.Header()["Content-Length"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "3", v[0])
		v, ok = writer.Header()["Content-Type"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, octetStreamMediaType, v[0])
		v, ok = writer.Header()["Content-Disposition"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "attachment; filename=test.ssz", v[0])
		v, ok = writer.Header()[versionHeader]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "version", v[0])
		assert.Equal(t, 204, writer.Code)
	})

	t.Run("no_grpc_status_code_header", func(t *testing.T) {
		response := &http.Response{
			Header:     http.Header{},
			StatusCode: 204,
		}
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		errJson := writeSSZResponseHeaderAndBody(response, writer, responseSsz, version, fileName)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, 204, writer.Code)
	})

	t.Run("invalid_status_code", func(t *testing.T) {
		response := &http.Response{
			Header: http.Header{
				"Foo": []string{"foo"},
				"Grpc-Metadata-" + grpc.HttpCodeMetadataKey: []string{"invalid"},
			},
		}
		responseSsz := []byte("ssz")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		errJson := writeSSZResponseHeaderAndBody(response, writer, responseSsz, version, fileName)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not parse status code"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestReceiveEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *sse.Event)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	req := httptest.NewRequest("GET", "http://foo.example", &bytes.Buffer{})
	req = req.WithContext(ctx)

	go func() {
		base64Val := "Zm9v"
		data := &eventFinalizedCheckpointJson{
			Block: base64Val,
			State: base64Val,
			Epoch: "1",
		}
		bData, err := json.Marshal(data)
		require.NoError(t, err)
		msg := &sse.Event{
			Data:  bData,
			Event: []byte(events.FinalizedCheckpointTopic),
		}
		ch <- msg
		time.Sleep(time.Second)
		cancel()
	}()

	errJson := receiveEvents(ch, w, req)
	assert.Equal(t, true, errJson == nil)
}

func TestReceiveEvents_EventNotSupported(t *testing.T) {
	ch := make(chan *sse.Event)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	req := httptest.NewRequest("GET", "http://foo.example", &bytes.Buffer{})

	go func() {
		msg := &sse.Event{
			Data:  []byte("foo"),
			Event: []byte("not_supported"),
		}
		ch <- msg
	}()

	errJson := receiveEvents(ch, w, req)
	require.NotNil(t, errJson)
	assert.Equal(t, "Event type 'not_supported' not supported", errJson.Msg())
}

func TestReceiveEvents_TrailingSpace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *sse.Event)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	req := httptest.NewRequest("GET", "http://foo.example", &bytes.Buffer{})
	req = req.WithContext(ctx)

	go func() {
		base64Val := "Zm9v"
		data := &eventFinalizedCheckpointJson{
			Block: base64Val,
			State: base64Val,
			Epoch: "1",
		}
		bData, err := json.Marshal(data)
		require.NoError(t, err)
		msg := &sse.Event{
			Data:  bData,
			Event: []byte("finalized_checkpoint "),
		}
		ch <- msg
		time.Sleep(time.Second)
		cancel()
	}()

	errJson := receiveEvents(ch, w, req)
	assert.Equal(t, true, errJson == nil)
	assert.Equal(t, `event: finalized_checkpoint
data: {"block":"0x666f6f","state":"0x666f6f","epoch":"1","execution_optimistic":false}

`, w.Body.String())
}

func TestWriteEvent(t *testing.T) {
	base64Val := "Zm9v"
	data := &eventFinalizedCheckpointJson{
		Block: base64Val,
		State: base64Val,
		Epoch: "1",
	}
	bData, err := json.Marshal(data)
	require.NoError(t, err)
	msg := &sse.Event{
		Data:  bData,
		Event: []byte("test_event"),
	}
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	errJson := writeEvent(msg, w, &eventFinalizedCheckpointJson{})
	require.Equal(t, true, errJson == nil)
	written := w.Body.String()
	assert.Equal(t, "event: test_event\ndata: {\"block\":\"0x666f6f\",\"state\":\"0x666f6f\",\"epoch\":\"1\",\"execution_optimistic\":false}\n\n", written)
}
