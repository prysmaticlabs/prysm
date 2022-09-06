package beaconapi_evaluators

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
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
	if requestPath == "/beacon/blocks/head" {
		responseDump, err := httputil.DumpResponse(httpResp, true)
		if err != nil {
			return err
		}
		fmt.Printf("BeaconBlock: %v", string(responseDump))
	}

	return json.NewDecoder(httpResp.Body).Decode(&dst)
}

func doMiddlewareJSONPostRequestV1(requestPath string, beaconNodeIdx int, postData, dst interface{}, bnType ...string) error {
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
	b, err := json.Marshal(postData)
	if err != nil {
		return err
	}
	basePath := fmt.Sprintf(v1MiddlewarePathTemplate, port+beaconNodeIdx)
	httpResp, err := http.Post(
		basePath+requestPath,
		"application/json",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return err
	}
	return json.NewDecoder(httpResp.Body).Decode(&dst)
}

func buildFieldError(field, expected, actual string) error {
	return fmt.Errorf("value of '%s' was expected to be '%s' but was '%s'", field, expected, actual)
}
