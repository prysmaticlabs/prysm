package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
	"github.com/prysmaticlabs/prysm/v4/network/httputil"
)

type JsonRestHandler interface {
	Get(ctx context.Context, query string, resp interface{}) error
	Post(ctx context.Context, endpoint string, headers map[string]string, data *bytes.Buffer, resp interface{}) error
	SwitchBeaconEndpoint(ctx context.Context, beaconApiUrls []string)
}

type beaconApiJsonRestHandler struct {
	httpClient http.Client
	host       func() string
}

// Get sends a GET request and decodes the response body as a JSON object into the passed in object.
// If an HTTP error is returned, the body is decoded as a DefaultJsonError JSON object and returned as the first return value.
func (c beaconApiJsonRestHandler) Get(ctx context.Context, endpoint string, resp interface{}) error {
	if resp == nil {
		return errors.New("resp is nil")
	}

	url := c.host() + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to create request for endpoint %s", url)
	}

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to perform request for endpoint %s", url)
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
) error {
	if data == nil {
		return errors.New("data is nil")
	}
	url := c.host() + apiEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, data)
	if err != nil {
		return errors.Wrapf(err, "failed to create request for endpoint %s", url)
	}

	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}
	req.Header.Set("Content-Type", api.JsonMediaType)

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to perform request for endpoint %s", url)
	}
	defer func() {
		if err = httpResp.Body.Close(); err != nil {
			return
		}
	}()

	return decodeResp(httpResp, resp)
}

func decodeResp(httpResp *http.Response, resp interface{}) error {
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read response body for %s", httpResp.Request.URL)
	}

	if httpResp.Header.Get("Content-Type") != api.JsonMediaType {
		if httpResp.StatusCode == http.StatusOK {
			return nil
		}
		return &httputil.DefaultJsonError{Code: httpResp.StatusCode, Message: string(body)}
	}

	decoder := json.NewDecoder(bytes.NewBuffer(body))
	if httpResp.StatusCode != http.StatusOK {
		errorJson := &httputil.DefaultJsonError{}
		if err = decoder.Decode(errorJson); err != nil {
			return errors.Wrapf(err, "failed to decode response body into error json for %s", httpResp.Request.URL)
		}
		return errorJson
	}
	// resp is nil for requests that do not return anything.
	if resp != nil {
		if err = decoder.Decode(resp); err != nil {
			return errors.Wrapf(err, "failed to decode response body into json for %s", httpResp.Request.URL)
		}
	}

	return nil
}

// SwitchBeaconEndpoint switches to the next available endpoint, this is circular.
func (c beaconApiJsonRestHandler) SwitchBeaconEndpoint(ctx context.Context, beaconApiUrls []string) {
	const endpoint = "/eth/v1/node/health"
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	go func() {
		for {
			select {
			case <-ticker.C:
				// GET request to the health endpoint using the current host
				err := c.Get(ctx, endpoint, nil)
				if err != nil {
					for i, url := range beaconApiUrls {
						if url == c.host() {
							next := (i + 1) % len(beaconApiUrls)
							c.changeHost(beaconApiUrls[next])
							break
						}
					}
				}
			}
		}
	}()
	defer ticker.Stop()
	select {}
}

// changeHost updates the host function in beaconApiJsonRestHandler
func (c beaconApiJsonRestHandler) changeHost(newHost string) {
	c.host = func() string { return newHost }
}
