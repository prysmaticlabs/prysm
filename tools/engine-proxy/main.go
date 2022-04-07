// Package main provides a proxy middleware for engine API requests between Ethereum
// consensus clients and execution clients accordingly. Allows for configuration of various
// test cases using yaml files as detailed in the README.md of the document.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/sirupsen/logrus"
)

var (
	log               = logrus.WithField("prefix", "engine-proxy")
	port              = flag.Int("port", 8546, "")
	executionEndpoint = flag.String("execution-endpoint", "http://localhost:8545", "")
)

type jsonRPCObject struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      uint64      `json:"id"`
	Result  interface{} `json:"result"`
}

func main() {
	flag.Parse()
	// Handle all HTTP requests through the proxy middleware.
	http.HandleFunc("/", proxyHandler)
	log.Printf("Engine proxy now listening on port %d", *port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

// Proxies requests from a consensus client to an execution client, spoofing requests
// and/or responses as desired. Acts as a middleware useful for testing different merge scenarios.
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	requestBytes, err := parseRequestBytes(r)
	if err != nil {
		log.WithError(err).Error("Could not parse request")
		return
	}

	// TODO: Allow for spoofing requests as well.

	// Create a new proxy request to the execution client.
	url := r.URL
	url.Host = *executionEndpoint
	proxyReq, err := http.NewRequest(r.Method, *executionEndpoint, r.Body)
	if err != nil {
		log.WithError(err).Error("Could create new request")
		return
	}
	// Required proxy headers for forwarding JSON-RPC requests to the execution client.
	proxyReq.Header.Set("Host", r.Host)
	proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	proxyReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	proxyRes, err := client.Do(proxyReq)
	if err != nil {
		log.WithError(err).Error("Could not do client proxy")
		return
	}

	// We optionally spoof the response as desired.
	modifiedResp, err := spoofResponse(requestBytes, proxyRes.Body)
	if err != nil {
		log.Error(err)
		return
	}

	if err = proxyRes.Body.Close(); err != nil {
		log.WithError(err).Error("Could not do client proxy")
		return
	}

	// Set the modified response as the proxy response body.
	proxyRes.Body = ioutil.NopCloser(bytes.NewBuffer(modifiedResp))

	// Pipe the proxy response to the original caller.
	if _, err = io.Copy(w, proxyRes.Body); err != nil {
		log.WithError(err).Error("Could not copy proxy request body")
		return
	}
}

func parseRequestBytes(req *http.Request) ([]byte, error) {
	requestBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	if err = req.Body.Close(); err != nil {
		return nil, err
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(requestBytes))
	return requestBytes, nil
}

// Parses the response body from the execution client and checks if user desires
// to spoof it based on the JSON-RPC method. If so, it returns the modified
// response bytes which will be proxied to the consensus client.
func spoofResponse(requestBytes []byte, responseBody io.Reader) ([]byte, error) {
	responseBytes, err := ioutil.ReadAll(responseBody)
	if err != nil {
		return nil, err
	}

	// If the JSON request is not a JSON-RPC object, return the response as-is.
	jsonRequest, err := unmarshalRPCObject(requestBytes)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "cannot unmarshal array"):
			return responseBytes, nil
		default:
			return nil, err
		}
	}

	// Detect the response, and modify execution payloads as needed.
	jsonResponse, err := unmarshalRPCObject(responseBytes)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "cannot unmarshal array"):
			return responseBytes, nil
		default:
			return nil, err
		}
	}

	// TODO: Allow configurable spoofing via YAML file inputs.
	switch jsonRequest.Method {
	case powchain.ForkchoiceUpdatedMethod:
		forkChoiceResp := &powchain.ForkchoiceUpdatedResponse{}
		if err := extractObjectFromJSONRPC(jsonResponse, forkChoiceResp); err != nil {
			return nil, err
		}
		// Modify the fork choice updated response to point
		// to the zero hash as the latest valid hash.
		forkChoiceResp.Status.LatestValidHash = make([]byte, 32)
		jsonResponse.Result = forkChoiceResp

		log.WithField("method", jsonRequest.Method).Infof("Modified response %v", forkChoiceResp)
		return json.Marshal(jsonResponse)
	default:
		return responseBytes, nil
	}
}

func unmarshalRPCObject(rawBytes []byte) (*jsonRPCObject, error) {
	jsonObj := &jsonRPCObject{}
	if err := json.Unmarshal(rawBytes, jsonObj); err != nil {
		return nil, err
	}
	return jsonObj, nil
}

func extractObjectFromJSONRPC(object *jsonRPCObject, target interface{}) error {
	rawResp, err := json.Marshal(object.Result)
	if err != nil {
		return err
	}
	return json.Unmarshal(rawResp, target)
}
