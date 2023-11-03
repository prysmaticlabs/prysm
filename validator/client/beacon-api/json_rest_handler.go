package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
)

type jsonRestHandler interface {
	GetRestJsonResponse(ctx context.Context, query string, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error)
	PostRestJson(ctx context.Context, apiEndpoint string, headers map[string]string, data *bytes.Buffer, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error)
}

type beaconApiJsonRestHandler struct {
	httpClient http.Client
	host       string
}

// GetRestJsonResponse sends a GET requests to apiEndpoint and decodes the response body as a JSON object into responseJson.
// If an HTTP error is returned, the body is decoded as a DefaultErrorJson JSON object instead and returned as the first return value.
// TODO: GetRestJsonResponse and PostRestJson have converged to the point of being nearly identical, but with some inconsistencies
// (like responseJson is being checked for nil one but not the other). We should merge them into a single method
// with variadic functional options for headers and data.
func (c beaconApiJsonRestHandler) GetRestJsonResponse(ctx context.Context, apiEndpoint string, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error) {
	if responseJson == nil {
		return nil, errors.New("responseJson is nil")
	}

	url := c.host + apiEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request with context")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query REST API %s", url)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			return
		}
	}()

	return decodeJsonResp(resp, responseJson)
}

// PostRestJson sends a POST requests to apiEndpoint and decodes the response body as a JSON object into responseJson. If responseJson
// is nil, nothing is decoded. If an HTTP error is returned, the body is decoded as a DefaultErrorJson JSON object instead and returned
// as the first return value.
func (c beaconApiJsonRestHandler) PostRestJson(ctx context.Context, apiEndpoint string, headers map[string]string, data *bytes.Buffer, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error) {
	if data == nil {
		return nil, errors.New("POST data is nil")
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send POST data to REST endpoint %s", url)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			return
		}
	}()

	return decodeJsonResp(resp, responseJson)
}

func decodeJsonResp(resp *http.Response, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error) {
	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		errorJson := &apimiddleware.DefaultErrorJson{}
		if err := decoder.Decode(errorJson); err != nil {
			if resp.StatusCode == http.StatusNotFound {
				errorJson = &apimiddleware.DefaultErrorJson{Code: http.StatusNotFound, Message: "Resource not found"}
			} else {
				remaining, readErr := io.ReadAll(decoder.Buffered())
				if readErr == nil {
					log.Debugf("Undecoded value: %s", string(remaining))
				}
				return nil, errors.Wrapf(err, "failed to decode error json for %s", resp.Request.URL)
			}
		}
		return errorJson, errors.Errorf("error %d: %s", errorJson.Code, errorJson.Message)
	}

	if responseJson != nil {
		if err := decoder.Decode(responseJson); err != nil {
			remaining, readErr := io.ReadAll(decoder.Buffered())
			if readErr == nil {
				log.Debugf("Undecoded value: %s", string(remaining))
			}
			return nil, errors.Wrapf(err, "failed to decode response json for %s", resp.Request.URL)
		}
	}

	return nil, nil
}
