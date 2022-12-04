//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
)

type jsonRestHandler interface {
	PostRestJson(apiEndpoint string, headers map[string]string, data *bytes.Buffer, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error)
}

type beaconApiJsonRestHandler struct {
	httpClient http.Client
	host       string
}

func (c beaconApiJsonRestHandler) PostRestJson(apiEndpoint string, headers map[string]string, data *bytes.Buffer, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error) {
	url := c.host + apiEndpoint
	req, err := http.NewRequest("POST", url, data)

	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query REST API %s", url)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
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

	if responseJson != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseJson); err != nil {
			return nil, errors.Wrapf(err, "failed to decode response json for %s", url)
		}
	}

	return nil, nil
}
