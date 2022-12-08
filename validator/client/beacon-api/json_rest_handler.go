package beacon_api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
)

type jsonRestHandler interface {
	GetRestJsonResponse(query string, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error)
	PostRestJson(apiEndpoint string, headers map[string]string, data *bytes.Buffer, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error)
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

	return decodeJsonResp(resp, responseJson)
}

// PostRestJson sends a POST requests to apiEndpoint and decodes the response body as a JSON object into responseJson. If responseJson
// is nil, nothing is decoded. If an HTTP error is returned, the body is decoded as a DefaultErrorJson JSON object instead and returned
// as the first return value.
func (c beaconApiJsonRestHandler) PostRestJson(apiEndpoint string, headers map[string]string, data *bytes.Buffer, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error) {
	if data == nil {
		return nil, errors.New("POST data is nil")
	}

	url := c.host + apiEndpoint
	req, err := http.NewRequest("POST", url, data)

	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

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
	if resp.StatusCode != http.StatusOK {
		errorJson := &apimiddleware.DefaultErrorJson{}
		if err := json.NewDecoder(resp.Body).Decode(errorJson); err != nil {
			return nil, errors.Wrapf(err, "failed to decode error json for %s", resp.Request.URL)
		}

		return errorJson, errors.Errorf("error %d: %s", errorJson.Code, errorJson.Message)
	}

	if responseJson != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseJson); err != nil {
			return nil, errors.Wrapf(err, "failed to decode response json for %s", resp.Request.URL)
		}
	}

	return nil, nil
}
