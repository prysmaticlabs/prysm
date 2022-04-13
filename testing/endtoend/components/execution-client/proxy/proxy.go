// Package proxy provides a proxy middleware for engine API requests between Ethereum
// consensus clients and execution clients accordingly. Allows for configuration of various
// test cases using yaml files as detailed in the README.md of the document.
package proxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "engine-api-proxy")

type spoofingConfig struct {
	Requests  []*spoof `yaml:"requests"`
	Responses []*spoof `yaml:"responses"`
}

type spoof struct {
	Method string
	Fields map[string]interface{}
}

type jsonRPCObject struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      uint64        `json:"id"`
	Result  interface{}   `json:"result"`
}

type forkchoiceUpdatedResponse struct {
	Status    *pb.PayloadStatus  `json:"payloadStatus"`
	PayloadId *pb.PayloadIDBytes `json:"payloadId"`
}

type InterceptorFunc func(reqBytes []byte, w http.ResponseWriter, r *http.Request) bool

type ProxyNode struct {
	address          string
	srv              *http.Server
	destAddress      string
	logger           *logrus.Logger
	logEntry         *logrus.Entry
	interceptor      InterceptorFunc
	backedUpRequests []*http.Request
}

func New(proxyPort int, destAddress string) *ProxyNode {
	pn := &ProxyNode{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", pn.proxyHandler())
	addr := "127.0.0.1:" + strconv.Itoa(proxyPort)
	srv := &http.Server{
		Handler: mux,
		Addr:    addr,
	}
	pn.address = addr
	pn.srv = srv
	pn.destAddress = destAddress
	return pn
}

func (pn *ProxyNode) Start(ctx context.Context) error {
	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, "eth1-proxy.log")
	if err != nil {
		return err
	}
	pn.logger = logrus.New()
	pn.logger.SetOutput(stdOutFile)
	pn.logEntry = pn.logger.WithField("prefix", "engine-proxy")
	pn.logEntry.Infof("Engine proxy now listening on address %s", pn.address)
	pn.srv.BaseContext = func(listener net.Listener) context.Context {
		return ctx
	}
	go func() {
		if err := pn.srv.ListenAndServe(); err != nil {
			pn.logEntry.Error(err)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return pn.srv.Shutdown(context.Background())
		}
	}
}

func (pn *ProxyNode) AddInterceptor(icptr InterceptorFunc) {
	pn.interceptor = icptr
}

func (pn *ProxyNode) SyncingInterceptor() InterceptorFunc {
	return func(reqBytes []byte, w http.ResponseWriter, r *http.Request) bool {
		if !pn.checkIfValid(reqBytes) {
			return false
		}
		pn.returnSyncingResponse(reqBytes, w, r)
		return true
	}
}

// Proxies requests from a consensus client to an execution client, spoofing requests
// and/or responses as desired. Acts as a middleware useful for testing different merge scenarios.
func (pn *ProxyNode) proxyHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestBytes, err := pn.parseRequestBytes(r)
		if err != nil {
			pn.logEntry.WithError(err).Error("Could not parse request")
			return
		}
		if pn.interceptor != nil && pn.interceptor(requestBytes, w, r) {
			return
		}

		for _, rq := range pn.backedUpRequests {
			requestB, err := pn.parseRequestBytes(rq)
			if err != nil {
				pn.logEntry.WithError(err).Error("Could not parse request")
				return
			}
			destAddr := "http://" + pn.destAddress

			// Create a new proxy request to the execution client.
			url := rq.URL
			url.Host = destAddr
			proxyReq, err := http.NewRequest(rq.Method, destAddr, rq.Body)
			if err != nil {
				pn.logEntry.WithError(err).Error("Could create new request")
				return
			}

			// Set the modified request as the proxy request body.
			proxyReq.Body = ioutil.NopCloser(bytes.NewBuffer(requestB))

			// Required proxy headers for forwarding JSON-RPC requests to the execution client.
			proxyReq.Header.Set("Host", rq.Host)
			proxyReq.Header.Set("X-Forwarded-For", rq.RemoteAddr)
			proxyReq.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			proxyRes, err := client.Do(proxyReq)
			if err != nil {
				pn.logEntry.WithError(err).Error("Could not do client proxy")
				return
			}
			buf := bytes.NewBuffer([]byte{})
			// Pipe the proxy response to the original caller.
			if _, err = io.Copy(buf, proxyRes.Body); err != nil {
				pn.logEntry.WithError(err).Error("Could not copy proxy request body")
				return
			}
			if err = proxyRes.Body.Close(); err != nil {
				pn.logEntry.WithError(err).Error("Could not do client proxy")
				return
			}
			pn.logEntry.Infof("Queued Request Response: %s", buf.String())
		}
		pn.backedUpRequests = []*http.Request{}

		/*
			var modifiedReq []byte
			if *fuzz {
				modifiedReq, err = fuzzRequest(requestBytes)
			} else {
				// We optionally spoof the request as desired.
				modifiedReq, err = spoofRequest(config, requestBytes)
			}
			if err != nil {
				log.WithError(err).Error("Failed to spoof request")
				return
			}
		*/

		destAddr := "http://" + pn.destAddress

		// Create a new proxy request to the execution client.
		url := r.URL
		url.Host = destAddr
		proxyReq, err := http.NewRequest(r.Method, destAddr, r.Body)
		if err != nil {
			pn.logEntry.WithError(err).Error("Could create new request")
			return
		}

		// Set the modified request as the proxy request body.
		proxyReq.Body = ioutil.NopCloser(bytes.NewBuffer(requestBytes))

		// Required proxy headers for forwarding JSON-RPC requests to the execution client.
		proxyReq.Header.Set("Host", r.Host)
		proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
		proxyReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		proxyRes, err := client.Do(proxyReq)
		if err != nil {
			pn.logEntry.WithError(err).Error("Could not do client proxy")
			return
		}

		/*
			// We optionally spoof the response as desired.
			var modifiedResp []byte
			if *fuzz {
				modifiedResp, err = fuzzResponse(proxyRes.Body)
			} else {
				modifiedResp, err = spoofResponse(config, requestBytes, proxyRes.Body)
			}
			if err != nil {
				log.WithError(err).Error("Failed to spoof response")
				return
			}
		*/

		// Pipe the proxy response to the original caller.
		if _, err = io.Copy(w, proxyRes.Body); err != nil {
			pn.logEntry.WithError(err).Error("Could not copy proxy request body")
			return
		}
		if err = proxyRes.Body.Close(); err != nil {
			pn.logEntry.WithError(err).Error("Could not do client proxy")
			return
		}
	}
}

func (pn *ProxyNode) parseRequestBytes(req *http.Request) ([]byte, error) {
	requestBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	if err = req.Body.Close(); err != nil {
		return nil, err
	}
	pn.logEntry.Infof("%s", string(requestBytes))
	req.Body = ioutil.NopCloser(bytes.NewBuffer(requestBytes))
	return requestBytes, nil
}

func fuzzRequest(requestBytes []byte) ([]byte, error) {
	// If the JSON request is not a JSON-RPC object, return the request as-is.
	jsonRequest, err := unmarshalRPCObject(requestBytes)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "cannot unmarshal array"):
			return requestBytes, nil
		default:
			return nil, err
		}
	}
	if len(jsonRequest.Params) == 0 {
		return requestBytes, nil
	}
	params := make(map[string]interface{})
	if err := extractObjectFromJSONRPC(jsonRequest.Params[0], &params); err != nil {
		return requestBytes, nil
	}
	// 10% chance to fuzz a field.
	fuzzProbability := uint64(5) // TODO: Do not hardcode.
	for k, v := range params {
		// Each field has a probability of of being fuzzed.
		r, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			return nil, err
		}
		switch obj := v.(type) {
		case string:
			if strings.Contains(obj, "0x") && r.Uint64() <= fuzzProbability {
				// Ignore hex strings of odd length.
				if len(obj)%2 != 0 {
					continue
				}
				// Generate some random hex string with same length and set it.
				// TODO: Experiment with length < current field, or junk in general.
				hexBytes, err := hexutil.Decode(obj)
				if err != nil {
					return nil, err
				}
				fuzzedStr, err := randomHex(len(hexBytes))
				if err != nil {
					return nil, err
				}
				log.WithField("method", jsonRequest.Method).Warnf(
					"Fuzzing field %s, modifying value from %s to %s",
					k,
					obj,
					fuzzedStr,
				)
				params[k] = fuzzedStr
			}
		}
	}
	jsonRequest.Params[0] = params
	return json.Marshal(jsonRequest)
}

// Parses the request from thec consensus client and checks if user desires
// to spoof it based on the JSON-RPC method. If so, it returns the modified
// request bytes which will be proxied to the execution client.
func spoofRequest(config *spoofingConfig, requestBytes []byte) ([]byte, error) {
	// If the JSON request is not a JSON-RPC object, return the request as-is.
	jsonRequest, err := unmarshalRPCObject(requestBytes)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "cannot unmarshal array"):
			return requestBytes, nil
		default:
			return nil, err
		}
	}
	if len(jsonRequest.Params) == 0 {
		return requestBytes, nil
	}
	desiredMethodsToSpoof := make(map[string]*spoof)
	for _, spoofReq := range config.Requests {
		desiredMethodsToSpoof[spoofReq.Method] = spoofReq
	}
	// If we don't want to spoof the request, just return the request as-is.
	spoofDetails, ok := desiredMethodsToSpoof[jsonRequest.Method]
	if !ok {
		return requestBytes, nil
	}

	// TODO: Support methods with multiple params.
	params := make(map[string]interface{})
	if err := extractObjectFromJSONRPC(jsonRequest.Params[0], &params); err != nil {
		return nil, err
	}
	for fieldToModify, fieldValue := range spoofDetails.Fields {
		if _, ok := params[fieldToModify]; !ok {
			continue
		}
		params[fieldToModify] = fieldValue
	}
	log.WithField("method", jsonRequest.Method).Infof("Modified request %v", params)
	jsonRequest.Params[0] = params
	return json.Marshal(jsonRequest)
}

func (pn *ProxyNode) checkIfValid(reqBytes []byte) bool {
	jsonRequest, err := unmarshalRPCObject(reqBytes)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "cannot unmarshal array"):
			return false
		default:
			return false
		}
	}
	if strings.Contains(jsonRequest.Method, "engine_forkchoiceUpdatedV1") ||
		strings.Contains(jsonRequest.Method, "engine_newPayloadV1") {
		return true
	}
	return false
}

func (pn *ProxyNode) returnSyncingResponse(reqBytes []byte, w http.ResponseWriter, r *http.Request) {
	jsonRequest, err := unmarshalRPCObject(reqBytes)
	if err != nil {
		return
	}
	switch {
	case strings.Contains(jsonRequest.Method, "engine_forkchoiceUpdatedV1"):
		resp := &forkchoiceUpdatedResponse{
			Status: &pb.PayloadStatus{
				Status: pb.PayloadStatus_SYNCING,
			},
			PayloadId: nil,
		}
		jResp := &jsonRPCObject{
			Method: jsonRequest.Method,
			ID:     jsonRequest.ID,
			Result: resp,
		}
		rawResp, err := json.Marshal(jResp)
		if err != nil {
			return
		}
		_, err = w.Write(rawResp)
		_ = err
		return
	case strings.Contains(jsonRequest.Method, "engine_newPayloadV1"):
		resp := &pb.PayloadStatus{
			Status: pb.PayloadStatus_SYNCING,
		}
		jResp := &jsonRPCObject{
			Method: jsonRequest.Method,
			ID:     jsonRequest.ID,
			Result: resp,
		}
		rawResp, err := json.Marshal(jResp)
		if err != nil {
			return
		}
		_, err = w.Write(rawResp)
		_ = err
		pn.backedUpRequests = append(pn.backedUpRequests, r)
		return
	}
	return
}

func fuzzResponse(responseBody io.Reader) ([]byte, error) {
	responseBytes, err := ioutil.ReadAll(responseBody)
	if err != nil {
		return nil, err
	}
	return responseBytes, nil
}

// Parses the response body from the execution client and checks if user desires
// to spoof it based on the JSON-RPC method. If so, it returns the modified
// response bytes which will be proxied to the consensus client.
func spoofResponse(config *spoofingConfig, requestBytes []byte, responseBody io.Reader) ([]byte, error) {
	responseBytes, err := ioutil.ReadAll(responseBody)
	if err != nil {
		return nil, err
	}
	// If the JSON request is not a JSON-RPC object, return the request as-is.
	jsonRequest, err := unmarshalRPCObject(requestBytes)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "cannot unmarshal array"):
			return responseBytes, nil
		default:
			return nil, err
		}
	}
	jsonResponse, err := unmarshalRPCObject(responseBytes)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "cannot unmarshal array"):
			return responseBytes, nil
		default:
			return nil, err
		}
	}
	desiredMethodsToSpoof := make(map[string]*spoof)
	for _, spoofReq := range config.Responses {
		desiredMethodsToSpoof[spoofReq.Method] = spoofReq
	}
	// If we don't want to spoof the request, just return the request as-is.
	spoofDetails, ok := desiredMethodsToSpoof[jsonRequest.Method]
	if !ok {
		return responseBytes, nil
	}

	// TODO: Support nested objects.
	params := make(map[string]interface{})
	if err := extractObjectFromJSONRPC(jsonResponse.Result, &params); err != nil {
		return nil, err
	}
	for fieldToModify, fieldValue := range spoofDetails.Fields {
		if _, ok := params[fieldToModify]; !ok {
			continue
		}
		params[fieldToModify] = fieldValue
	}
	log.WithField("method", jsonRequest.Method).Infof("Modified response %v", params)
	jsonResponse.Result = params
	return json.Marshal(jsonResponse)
}

func unmarshalRPCObject(rawBytes []byte) (*jsonRPCObject, error) {
	jsonObj := &jsonRPCObject{}
	if err := json.Unmarshal(rawBytes, jsonObj); err != nil {
		return nil, err
	}
	return jsonObj, nil
}

func extractObjectFromJSONRPC(src interface{}, dst interface{}) error {
	rawResp, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(rawResp, dst)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(b), nil
}
