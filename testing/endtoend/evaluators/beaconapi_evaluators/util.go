package beaconapi_evaluators

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/params"
	log "github.com/sirupsen/logrus"
)

func doMiddlewareJSONGetRequest(template string, requestPath string, beaconNodeIdx int, dst interface{}, bnType ...string) error {
	var port int
	if len(bnType) > 0 {
		switch bnType[0] {
		case "lighthouse":
			port = params.TestParams.Ports.LighthouseBeaconNodeHTTPPort
		default:
			port = params.TestParams.Ports.PrysmBeaconNodeGatewayPort
		}
	} else {
		port = params.TestParams.Ports.PrysmBeaconNodeGatewayPort
	}

	basePath := fmt.Sprintf(template, port+beaconNodeIdx)
	httpResp, err := http.Get(
		basePath + requestPath,
	)
	if err != nil {
		return err
	}
	if httpResp.StatusCode != http.StatusOK {
		var body interface{}
		if err := json.NewDecoder(httpResp.Body).Decode(&body); err != nil {
			return err
		}
		return fmt.Errorf("request failed with response code: %d with response body %s", httpResp.StatusCode, body)
	}
	return json.NewDecoder(httpResp.Body).Decode(&dst)
}

func doMiddlewareSSZGetRequest(template string, requestPath string, beaconNodeIdx int, bnType ...string) ([]byte, error) {
	client := &http.Client{}
	var port int
	if len(bnType) > 0 {
		switch bnType[0] {
		case "lighthouse":
			port = params.TestParams.Ports.LighthouseBeaconNodeHTTPPort
		default:
			port = params.TestParams.Ports.PrysmBeaconNodeGatewayPort
		}
	} else {
		port = params.TestParams.Ports.PrysmBeaconNodeGatewayPort
	}

	basePath := fmt.Sprintf(template, port+beaconNodeIdx)

	req, err := http.NewRequest("GET", basePath+requestPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != http.StatusOK {
		var body interface{}
		if err := json.NewDecoder(rsp.Body).Decode(&body); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("request failed with response code: %d with response body %s", rsp.StatusCode, body)
	}
	defer closeBody(rsp.Body)
	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func doMiddlewareJSONPostRequest(template string, requestPath string, beaconNodeIdx int, postData, dst interface{}, bnType ...string) error {
	var port int
	if len(bnType) > 0 {
		switch bnType[0] {
		case "lighthouse":
			port = params.TestParams.Ports.LighthouseBeaconNodeHTTPPort
		default:
			port = params.TestParams.Ports.PrysmBeaconNodeGatewayPort
		}
	} else {
		port = params.TestParams.Ports.PrysmBeaconNodeGatewayPort
	}
	basePath := fmt.Sprintf(template, port+beaconNodeIdx)
	b, err := json.Marshal(postData)
	if err != nil {
		return err
	}
	httpResp, err := http.Post(
		basePath+requestPath,
		"application/json",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return err
	}
	if httpResp.StatusCode != http.StatusOK {
		var body interface{}
		if err := json.NewDecoder(httpResp.Body).Decode(&body); err != nil {
			return err
		}
		return fmt.Errorf("request failed with response code: %d with response body %s", httpResp.StatusCode, body)
	}
	return json.NewDecoder(httpResp.Body).Decode(&dst)
}

func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		log.WithError(err).Error("could not close response body")
	}
}
