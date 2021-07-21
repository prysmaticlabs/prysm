package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus/hooks/test"
)

type testRequestContainer struct {
	TestString    string
	TestHexString string `hex:"true"`
}

func defaultRequestContainer() *testRequestContainer {
	return &testRequestContainer{
		TestString:    "test string",
		TestHexString: "0x666F6F", // hex encoding of "foo"
	}
}

type testResponseContainer struct {
	TestString string
	TestHex    string `hex:"true"`
	TestEnum   string `enum:"true"`
	TestTime   string `time:"true"`
}

func defaultResponseContainer() *testResponseContainer {
	return &testResponseContainer{
		TestString: "test string",
		TestHex:    "Zm9v", // base64 encoding of "foo"
		TestEnum:   "Test Enum",
		TestTime:   "2006-01-02T15:04:05Z",
	}
}

type testErrorJson struct {
	Message     string
	Code        int
	CustomField string
}

// StatusCode returns the error's underlying error code.
func (e *testErrorJson) StatusCode() int {
	return e.Code
}

// Msg returns the error's underlying message.
func (e *testErrorJson) Msg() string {
	return e.Message
}

// SetCode sets the error's underlying error code.
func (e *testErrorJson) SetCode(code int) {
	e.Code = code
}

func TestDeserializeRequestBodyIntoContainer(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var bodyJson bytes.Buffer
		err := json.NewEncoder(&bodyJson).Encode(defaultRequestContainer())
		require.NoError(t, err)

		container := &testRequestContainer{}
		errJson := DeserializeRequestBodyIntoContainer(&bodyJson, container)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "test string", container.TestString)
	})

	t.Run("error", func(t *testing.T) {
		var bodyJson bytes.Buffer
		bodyJson.Write([]byte("foo"))
		errJson := DeserializeRequestBodyIntoContainer(&bodyJson, &testRequestContainer{})
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode request body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestProcessRequestContainerFields(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		container := defaultRequestContainer()

		errJson := ProcessRequestContainerFields(container)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "Zm9v", container.TestHexString)
	})

	t.Run("error", func(t *testing.T) {
		errJson := ProcessRequestContainerFields("foo")
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not process request data"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestSetRequestBodyToRequestContainer(t *testing.T) {
	var body bytes.Buffer
	request := httptest.NewRequest("GET", "http://foo.example", &body)

	errJson := SetRequestBodyToRequestContainer(defaultRequestContainer(), request)
	require.Equal(t, true, errJson == nil)
	container := &testRequestContainer{}
	require.NoError(t, json.NewDecoder(request.Body).Decode(container))
	assert.Equal(t, "test string", container.TestString)
	contentLengthHeader, ok := request.Header["Content-Length"]
	require.Equal(t, true, ok)
	require.Equal(t, 1, len(contentLengthHeader), "wrong number of header values")
	assert.Equal(t, "55", contentLengthHeader[0])
	assert.Equal(t, int64(55), request.ContentLength)
}

func TestPrepareRequestForProxying(t *testing.T) {
	middleware := &ApiProxyMiddleware{
		GatewayAddress: "http://gateway.example",
	}
	// We will set some params to make the request more interesting.
	endpoint := Endpoint{
		Path:               "/{url_param}",
		RequestURLLiterals: []string{"url_param"},
		RequestQueryParams: []QueryParam{{Name: "query_param"}},
	}
	var body bytes.Buffer
	request := httptest.NewRequest("GET", "http://foo.example?query_param=bar", &body)

	errJson := middleware.PrepareRequestForProxying(endpoint, request)
	require.Equal(t, true, errJson == nil)
	assert.Equal(t, "http", request.URL.Scheme)
	assert.Equal(t, middleware.GatewayAddress, request.URL.Host)
	assert.Equal(t, "", request.RequestURI)
}

func TestReadGrpcResponseBody(t *testing.T) {
	var b bytes.Buffer
	b.Write([]byte("foo"))

	body, jsonErr := ReadGrpcResponseBody(&b)
	require.Equal(t, true, jsonErr == nil)
	assert.Equal(t, "foo", string(body))
}

func TestDeserializeGrpcResponseBodyIntoErrorJson(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		e := &testErrorJson{
			Message: "foo",
			Code:    500,
		}
		body, err := json.Marshal(e)
		require.NoError(t, err)

		eToDeserialize := &testErrorJson{}
		errJson := DeserializeGrpcResponseBodyIntoErrorJson(eToDeserialize, body)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "foo", eToDeserialize.Msg())
		assert.Equal(t, 500, eToDeserialize.StatusCode())
	})

	t.Run("error", func(t *testing.T) {
		errJson := DeserializeGrpcResponseBodyIntoErrorJson(nil, nil)
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not unmarshal error"))
	})
}

func TestHandleGrpcResponseError(t *testing.T) {
	response := &http.Response{
		StatusCode: 400,
		Header: http.Header{
			"Foo": []string{"foo"},
			"Bar": []string{"bar"},
		},
	}
	writer := httptest.NewRecorder()
	errJson := &testErrorJson{
		Message: "foo",
		Code:    500,
	}

	HandleGrpcResponseError(errJson, response, writer)
	v, ok := writer.Header()["Foo"]
	require.Equal(t, true, ok, "header not found")
	require.Equal(t, 1, len(v), "wrong number of header values")
	assert.Equal(t, "foo", v[0])
	v, ok = writer.Header()["Bar"]
	require.Equal(t, true, ok, "header not found")
	require.Equal(t, 1, len(v), "wrong number of header values")
	assert.Equal(t, "bar", v[0])
	assert.Equal(t, 400, errJson.StatusCode())
}

func TestGrpcResponseIsStatusCodeOnly(t *testing.T) {
	var body bytes.Buffer

	t.Run("status_code_only", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", &body)
		result := GrpcResponseIsStatusCodeOnly(request, nil)
		assert.Equal(t, true, result)
	})

	t.Run("different_method", func(t *testing.T) {
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		result := GrpcResponseIsStatusCodeOnly(request, nil)
		assert.Equal(t, false, result)
	})

	t.Run("non_empty_response", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", &body)
		result := GrpcResponseIsStatusCodeOnly(request, &testRequestContainer{})
		assert.Equal(t, false, result)
	})
}

func TestDeserializeGrpcResponseBodyIntoContainer(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		body, err := json.Marshal(defaultRequestContainer())
		require.NoError(t, err)

		container := &testRequestContainer{}
		errJson := DeserializeGrpcResponseBodyIntoContainer(body, container)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "test string", container.TestString)
	})

	t.Run("error", func(t *testing.T) {
		var bodyJson bytes.Buffer
		bodyJson.Write([]byte("foo"))
		errJson := DeserializeGrpcResponseBodyIntoContainer(bodyJson.Bytes(), &testRequestContainer{})
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not unmarshal response"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestProcessMiddlewareResponseFields(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		container := defaultResponseContainer()

		errJson := ProcessMiddlewareResponseFields(container)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "0x666f6f", container.TestHex)
		assert.Equal(t, "test enum", container.TestEnum)
		assert.Equal(t, "1136214245", container.TestTime)
	})

	t.Run("error", func(t *testing.T) {
		errJson := ProcessMiddlewareResponseFields("foo")
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not process response data"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestSerializeMiddlewareResponseIntoJson(t *testing.T) {
	container := defaultResponseContainer()
	j, errJson := SerializeMiddlewareResponseIntoJson(container)
	assert.Equal(t, true, errJson == nil)
	cToDeserialize := &testResponseContainer{}
	require.NoError(t, json.Unmarshal(j, cToDeserialize))
	assert.Equal(t, "test string", cToDeserialize.TestString)
}

func TestWriteMiddlewareResponseHeadersAndBody(t *testing.T) {
	var body bytes.Buffer

	t.Run("GET", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", &body)
		response := &http.Response{
			Header: http.Header{
				"Foo": []string{"foo"},
				"Grpc-Metadata-" + grpcutils.HttpCodeMetadataKey: []string{"204"},
			},
		}
		container := defaultResponseContainer()
		responseJson, err := json.Marshal(container)
		require.NoError(t, err)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		errJson := WriteMiddlewareResponseHeadersAndBody(request, response, responseJson, writer)
		require.Equal(t, true, errJson == nil)
		v, ok := writer.Header()["Foo"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "foo", v[0])
		v, ok = writer.Header()["Content-Length"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "102", v[0])
		assert.Equal(t, 204, writer.Code)
		assert.DeepEqual(t, responseJson, writer.Body.Bytes())
	})

	t.Run("GET_no_grpc_status_code_header", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", &body)
		response := &http.Response{
			Header:     http.Header{},
			StatusCode: 204,
		}
		container := defaultResponseContainer()
		responseJson, err := json.Marshal(container)
		require.NoError(t, err)
		writer := httptest.NewRecorder()

		errJson := WriteMiddlewareResponseHeadersAndBody(request, response, responseJson, writer)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, 204, writer.Code)
	})

	t.Run("GET_invalid_status_code", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", &body)
		response := &http.Response{
			Header: http.Header{},
		}

		// Set invalid status code.
		response.Header["Grpc-Metadata-"+grpcutils.HttpCodeMetadataKey] = []string{"invalid"}

		container := defaultResponseContainer()
		responseJson, err := json.Marshal(container)
		require.NoError(t, err)
		writer := httptest.NewRecorder()

		errJson := WriteMiddlewareResponseHeadersAndBody(request, response, responseJson, writer)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not parse status code"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})

	t.Run("POST", func(t *testing.T) {
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		response := &http.Response{
			Header:     http.Header{},
			StatusCode: 204,
		}
		container := defaultResponseContainer()
		responseJson, err := json.Marshal(container)
		require.NoError(t, err)
		writer := httptest.NewRecorder()

		errJson := WriteMiddlewareResponseHeadersAndBody(request, response, responseJson, writer)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, 204, writer.Code)
	})

	t.Run("POST_with_response_body", func(t *testing.T) {
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		response := &http.Response{
			Header:     http.Header{},
			StatusCode: 204,
		}
		container := defaultResponseContainer()
		responseJson, err := json.Marshal(container)
		require.NoError(t, err)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		errJson := WriteMiddlewareResponseHeadersAndBody(request, response, responseJson, writer)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, 204, writer.Code)
		assert.DeepEqual(t, responseJson, writer.Body.Bytes())
	})
}

func TestWriteError(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		responseHeader := http.Header{
			"Grpc-Metadata-" + grpcutils.CustomErrorMetadataKey: []string{"{\"CustomField\":\"bar\"}"},
		}
		errJson := &testErrorJson{
			Message: "foo",
			Code:    500,
		}
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		WriteError(writer, errJson, responseHeader)
		v, ok := writer.Header()["Content-Length"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "48", v[0])
		v, ok = writer.Header()["Content-Type"]
		require.Equal(t, true, ok, "header not found")
		require.Equal(t, 1, len(v), "wrong number of header values")
		assert.Equal(t, "application/json", v[0])
		assert.Equal(t, 500, writer.Code)
		eDeserialize := &testErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), eDeserialize))
		assert.Equal(t, "foo", eDeserialize.Message)
		assert.Equal(t, 500, eDeserialize.Code)
		assert.Equal(t, "bar", eDeserialize.CustomField)
	})

	t.Run("invalid_custom_error_header", func(t *testing.T) {
		logHook := test.NewGlobal()

		responseHeader := http.Header{
			"Grpc-Metadata-" + grpcutils.CustomErrorMetadataKey: []string{"invalid"},
		}

		WriteError(httptest.NewRecorder(), &testErrorJson{}, responseHeader)
		assert.LogsContain(t, logHook, "Could not unmarshal custom error message")
	})
}
