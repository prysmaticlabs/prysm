package apimiddleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestWrapAttestationArray(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitAttestationRequestJson{},
		}
		unwrappedAtts := []*attestationJson{{AggregationBits: "1010"}}
		unwrappedAttsJson, err := json.Marshal(unwrappedAtts)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedAttsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapAttestationsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrappedAtts := &submitAttestationRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedAtts))
		require.Equal(t, 1, len(wrappedAtts.Data), "wrong number of wrapped attestations")
		assert.Equal(t, "1010", wrappedAtts.Data[0].AggregationBits)
	})

	t.Run("invalid body", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitAttestationRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapAttestationsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode attestations array"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestPrepareGraffiti(t *testing.T) {
	endpoint := gateway.Endpoint{
		PostRequest: &beaconBlockContainerJson{
			Message: &beaconBlockJson{
				Body: &beaconBlockBodyJson{},
			},
		},
	}

	t.Run("32 bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = string(bytesutil.PadTo([]byte("foo"), 32))

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})

	t.Run("less than 32 bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = "foo"

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})

	t.Run("more than 32 bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = string(bytesutil.PadTo([]byte("foo"), 33))

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})
}

func TestSszRequested(t *testing.T) {
	t.Run("SSZ requested", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{"application/octet-stream"}
		result := sszRequested(request)
		assert.Equal(t, true, result)
	})

	t.Run("multiple content types", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{"application/json", "application/octet-stream"}
		result := sszRequested(request)
		assert.Equal(t, true, result)
	})

	t.Run("no header", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		result := sszRequested(request)
		assert.Equal(t, false, result)
	})

	t.Run("other content type", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{"application/json"}
		result := sszRequested(request)
		assert.Equal(t, false, result)
	})
}

func TestPrepareSszRequestForProxying(t *testing.T) {
	middleware := &gateway.ApiProxyMiddleware{
		GatewayAddress: "http://gateway.example",
	}
	endpoint := gateway.Endpoint{
		Url: "http://foo.example",
	}
	var body bytes.Buffer
	request := httptest.NewRequest("GET", "http://foo.example", &body)

	errJson := prepareSszRequestForProxying(middleware, endpoint, request, "/ssz")
	require.Equal(t, true, errJson == nil)
	assert.Equal(t, "/ssz", request.URL.Path)
}

func TestSerializeMiddlewareResponseIntoSsz(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		ssz, errJson := serializeMiddlewareResponseIntoSsz("Zm9v")
		require.Equal(t, true, errJson == nil)
		assert.DeepEqual(t, []byte("foo"), ssz)
	})

	t.Run("invalid data", func(t *testing.T) {
		_, errJson := serializeMiddlewareResponseIntoSsz("invalid")
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode response body into base64"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWriteSszResponseHeaderAndBody(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		response := &http.Response{
			Header: http.Header{
				"Foo": []string{"foo"},
				"Grpc-Metadata-" + grpcutils.HttpCodeMetadataKey: []string{"204"},
			},
		}
		responseSsz := []byte("ssz")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		errJson := writeSszResponseHeaderAndBody(response, writer, responseSsz, "test.ssz")
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
		assert.Equal(t, "application/octet-stream", v[0])
		v, ok = writer.Header()["Content-Disposition"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "attachment; filename=test.ssz", v[0])
		assert.Equal(t, 204, writer.Code)
	})

	t.Run("no gRPC status code header", func(t *testing.T) {
		response := &http.Response{
			Header:     http.Header{},
			StatusCode: 204,
		}
		responseSsz := []byte("ssz")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		errJson := writeSszResponseHeaderAndBody(response, writer, responseSsz, "test.ssz")
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, 204, writer.Code)
	})

	t.Run("invalid status code", func(t *testing.T) {
		response := &http.Response{
			Header: http.Header{
				"Foo": []string{"foo"},
				"Grpc-Metadata-" + grpcutils.HttpCodeMetadataKey: []string{"invalid"},
			},
		}
		responseSsz := []byte("ssz")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		errJson := writeSszResponseHeaderAndBody(response, writer, responseSsz, "test.ssz")
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not parse status code"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}
