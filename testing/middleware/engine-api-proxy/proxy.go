// Package proxy provides a proxy middleware for engine API requests between Ethereum
// consensus clients and execution clients accordingly. Allows for configuration of various
// test cases using yaml files as detailed in the README.md of the document.
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/sirupsen/logrus"
)

var (
	defaultProxyHost = "127.0.0.1"
	defaultProxyPort = 8545
)

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

// Proxy server that sits as a middleware between an Ethereum consensus client and an execution client,
// allowing us to modify in-flight requests and responses for testing purposes.
type Proxy struct {
	cfg              *config
	address          string
	srv              *http.Server
	interceptor      InterceptorFunc
	backedUpRequests []*http.Request
}

// New creates a proxy server forwarding requests from a consensus client to an execution client.
func New(opts ...Option) (*Proxy, error) {
	defaultInterceptor := func(_ []byte, _ http.ResponseWriter, _ *http.Request) bool {
		return false // No default intercepting of requests.
	}
	p := &Proxy{
		cfg: &config{
			proxyHost: defaultProxyHost,
			proxyPort: defaultProxyPort,
			logger:    logrus.New(),
		},
		interceptor: defaultInterceptor,
	}
	for _, o := range opts {
		if err := o(p); err != nil {
			return nil, err
		}
	}
	mux := http.NewServeMux()
	mux.Handle("/", p)
	addr := fmt.Sprintf("%s:%d", p.cfg.proxyHost, p.cfg.proxyPort)
	srv := &http.Server{
		Handler: mux,
		Addr:    addr,
	}
	p.address = addr
	p.srv = srv
	return p, nil
}

// Start a proxy server.
func (p *Proxy) Start(ctx context.Context) error {
	p.srv.BaseContext = func(listener net.Listener) context.Context {
		return ctx
	}
	p.cfg.logger.WithFields(logrus.Fields{
		"forwardingAddress": p.cfg.destinationUrl.String(),
	}).Infof("Engine proxy now listening on address %s", p.address)
	go func() {
		if err := p.srv.ListenAndServe(); err != nil {
			p.cfg.logger.Error(err)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return p.srv.Shutdown(context.Background())
		}
	}
}

// ServeHTTP requests from a consensus client to an execution client, modifying in-flight requests
// and/or responses as desired. It also processes any backed-up requests.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestBytes, err := parseRequestBytes(r)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not parse request")
		return
	}
	if p.interceptor(requestBytes, w, r) {
		return
	}
	for _, rq := range p.backedUpRequests {
		requestB, err := parseRequestBytes(r)
		if err != nil {
			p.cfg.logger.WithError(err).Error("Could not parse request")
			return
		}
		p.proxyRequest(requestB, rq)
	}
	p.backedUpRequests = []*http.Request{}
	p.proxyRequest(requestBytes, r)
}

// Create a new proxy request to the execution client.
func (p *Proxy) proxyRequest(requestBytes []byte, r *http.Request) {
	proxyReq, err := http.NewRequest(r.Method, p.cfg.destinationUrl.String(), r.Body)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could create new request")
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
		p.cfg.logger.WithError(err).Error("Could not do client proxy")
		return
	}
	defer func() {
		if err = proxyRes.Body.Close(); err != nil {
			p.cfg.logger.WithError(err).Error("Could not do client proxy")
		}
	}()

	// Pipe the proxy response to the original caller.
	buf := bytes.NewBuffer(make([]byte, 0))
	if _, err = io.Copy(buf, proxyRes.Body); err != nil {
		p.cfg.logger.WithError(err).Error("Could not copy proxy request body")
		return
	}
}

func (p *Proxy) returnSyncingResponse(reqBytes []byte, w http.ResponseWriter, r *http.Request) {
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
		p.backedUpRequests = append(p.backedUpRequests, r)
		return
	}
	return
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

func checkIfValid(reqBytes []byte) bool {
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

func unmarshalRPCObject(b []byte) (*jsonRPCObject, error) {
	r := &jsonRPCObject{}
	if err := json.Unmarshal(b, r); err != nil {
		return nil, err
	}
	return r, nil
}
