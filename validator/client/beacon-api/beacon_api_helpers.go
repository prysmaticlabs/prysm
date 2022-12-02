//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"regexp"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
)

func validRoot(root string) bool {
	matchesRegex, err := regexp.MatchString("^0x[a-fA-F0-9]{64}$", root)
	if err != nil {
		return false
	}
	return matchesRegex
}

func buildURL(hostPort string, path string, queryParams ...neturl.Values) string {
	url := fmt.Sprintf("%s/%s", hostPort, path)

	if len(queryParams) == 0 {
		return url
	}

	return fmt.Sprintf("%s?%s", url, queryParams[0].Encode())
}

func getRestJsonResponse(httpClient http.Client, query string, responseJson interface{}) (*apimiddleware.DefaultErrorJson, error) {
	resp, err := httpClient.Get(query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query REST API %s", query)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		errorJson := &apimiddleware.DefaultErrorJson{}
		if err := json.NewDecoder(resp.Body).Decode(errorJson); err != nil {
			return nil, errors.Wrapf(err, "failed to decode error json for %s", query)
		}

		return errorJson, errors.Errorf("error %d: %s", errorJson.Code, errorJson.Message)
	}

	if err := json.NewDecoder(resp.Body).Decode(responseJson); err != nil {
		return nil, errors.Wrapf(err, "failed to decode response json for %s", query)
	}

	return nil, nil
}
