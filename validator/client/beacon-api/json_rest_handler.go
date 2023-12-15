package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
	"github.com/prysmaticlabs/prysm/v4/network/httputil"
)

type JsonRestHandler interface {
	Get(ctx context.Context, query string, resp interface{}) (*httputil.DefaultJsonError, error)
	Post(ctx context.Context, endpoint string, headers map[string]string, data *bytes.Buffer, resp interface{}) (*httputil.DefaultJsonError, error)
}

type beaconApiJsonRestHandler struct {
	httpClient http.Client
	host       string
}

// Get sends a GET request and decodes the response body as a JSON object into the passed in object.
// If an HTTP error is returned, the body is decoded as a DefaultJsonError JSON object and returned as the first return value.
func (c beaconApiJsonRestHandler) Get(ctx context.Context, endpoint string, resp interface{}) (*httputil.DefaultJsonError, error) {
	if resp == nil {
		return nil, errors.New("resp is nil")
	}

	url := c.host + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request with context")
	}

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform request with HTTP client")
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			return
		}
	}()

	return decodeResp(httpResp, resp)
}

// Post sends a POST request and decodes the response body as a JSON object into the passed in object.
// If an HTTP error is returned, the body is decoded as a DefaultJsonError JSON object and returned as the first return value.
func (c beaconApiJsonRestHandler) Post(
	ctx context.Context,
	apiEndpoint string,
	headers map[string]string,
	data *bytes.Buffer,
	resp interface{},
) (*httputil.DefaultJsonError, error) {
	if data == nil {
		return nil, errors.New("data is nil")
	}

	url := c.host + apiEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request with context")
	}

	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}
	req.Header.Set("Content-Type", api.JsonMediaType)

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform request with HTTP client")
	}
	defer func() {
		if err = httpResp.Body.Close(); err != nil {
			return
		}
	}()

	return decodeResp(httpResp, resp)
}

func decodeResp(httpResp *http.Response, resp interface{}) (*httputil.DefaultJsonError, error) {
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body for %s", httpResp.Request.URL)
	}

	if httpResp.Header.Get("Content-Type") != api.JsonMediaType {
		if httpResp.StatusCode == http.StatusOK {
			return nil, nil
		}
		return &httputil.DefaultJsonError{Code: httpResp.StatusCode, Message: string(body)}, nil
	}

	decoder := json.NewDecoder(bytes.NewBuffer(body))
	if httpResp.StatusCode != http.StatusOK {
		errorJson := &httputil.DefaultJsonError{}
		if err = decoder.Decode(errorJson); err != nil {
			return nil, errors.Wrapf(err, "failed to decode response body into error json for %s", httpResp.Request.URL)
		}
		return errorJson, nil
	}
	// resp is nil for requests that do not return anything.
	if resp != nil {
		if err = decoder.Decode(resp); err != nil {
			return nil, errors.Wrapf(err, "failed to decode response body into json for %s", httpResp.Request.URL)
		}
	}

	return nil, nil
}
