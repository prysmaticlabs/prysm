//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
)

type jsonRestHandler interface {
	GetRestJsonResponse(query string, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error)
}

type beaconApiJsonRestHandler struct {
	httpClient http.Client
	host       string
}

// GetRestJsonResponse sends a GET requests to apiEndpoint and decodes the response body as a JSON object into responseJson.
// If an HTTP error is returned, the body is decoded as a DefaultErrorJson JSON object instead and returned as the first return value.
func (c beaconApiJsonRestHandler) GetRestJsonResponse(apiEndpoint string, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error) {
	if responseJson == nil {
		return nil, errors.New("responseJson is nil")
	}

	url := c.host + apiEndpoint
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query REST API %s", url)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		errorJson := &apimiddleware.DefaultErrorJson{}
		if err := json.NewDecoder(resp.Body).Decode(errorJson); err != nil {
			return nil, errors.Wrapf(err, "failed to decode error json for %s", url)
		}

		return errorJson, errors.Errorf("error %d: %s", errorJson.Code, errorJson.Message)
	}

	if err := json.NewDecoder(resp.Body).Decode(responseJson); err != nil {
		return nil, errors.Wrapf(err, "failed to decode response json for %s", url)
	}

	return nil, nil
}
