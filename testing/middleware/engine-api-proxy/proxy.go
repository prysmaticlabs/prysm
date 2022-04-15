// Package proxy provides a proxy middleware for engine API requests between Ethereum
// consensus clients and execution clients accordingly. Allows for customizing
// in-flight requests or responses using custom triggers. Useful for end-to-end testing.
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
	"sync"

	"github.com/pkg/errors"
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

type interceptorConfig struct {
	response interface{}
	trigger  func() bool
}

// Proxy server that sits as a middleware between an Ethereum consensus client and an execution client,
// allowing us to modify in-flight requests and responses for testing purposes.
type Proxy struct {
	cfg              *config
	address          string
	srv              *http.Server
	lock             sync.RWMutex
	interceptors     map[string]*interceptorConfig
	backedUpRequests []*http.Request
}

// New creates a proxy server forwarding requests from a consensus client to an execution client.
func New(opts ...Option) (*Proxy, error) {
	p := &Proxy{
		cfg: &config{
			proxyHost: defaultProxyHost,
			proxyPort: defaultProxyPort,
			logger:    logrus.New(),
		},
		interceptors: make(map[string]*interceptorConfig),
	}
	for _, o := range opts {
		if err := o(p); err != nil {
			return nil, err
		}
	}
	if p.cfg.destinationUrl == nil {
		return nil, errors.New("must provide a destination address for request proxying")
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

// Address for the proxy server.
func (p *Proxy) Address() string {
	return p.address
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
	// Check if we need to intercept the request with a custom response.
	hasIntercepted, err := p.interceptIfNeeded(requestBytes, w)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not intercept request")
		return
	}
	if hasIntercepted {
		return
	}
	// If we are not intercepting the request, we proxy as normal.
	p.proxyRequest(requestBytes, w, r)
}

// AddRequestInterceptor for a desired json-rpc method by specifying a custom response
// and a function that checks if the interceptor should be triggered.
func (p *Proxy) AddRequestInterceptor(rpcMethodName string, response interface{}, trigger func() bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.interceptors[rpcMethodName] = &interceptorConfig{
		response,
		trigger,
	}
}

// Checks if there is a custom interceptor hook on the request, check if it can be
// triggered, and then write the custom response to the writer.
func (p *Proxy) interceptIfNeeded(requestBytes []byte, w http.ResponseWriter) (hasIntercepted bool, err error) {
	if !isEngineAPICall(requestBytes) {
		return
	}
	var jreq *jsonRPCObject
	jreq, err = unmarshalRPCObject(requestBytes)
	if err != nil {
		return
	}
	p.lock.RLock()
	defer p.lock.RUnlock()
	interceptor, shouldIntercept := p.interceptors[jreq.Method]
	if !shouldIntercept {
		return
	}
	if !interceptor.trigger() {
		return
	}
	jResp := &jsonRPCObject{
		Method: jreq.Method,
		ID:     jreq.ID,
		Result: interceptor.response,
	}
	if err = json.NewEncoder(w).Encode(jResp); err != nil {
		return
	}
	hasIntercepted = true
	return
}

// Create a new proxy request to the execution client.
func (p *Proxy) proxyRequest(requestBytes []byte, w http.ResponseWriter, r *http.Request) {
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
		p.cfg.logger.WithError(err).Error("Could not forward request to destination server")
		return
	}
	defer func() {
		if err = proxyRes.Body.Close(); err != nil {
			p.cfg.logger.WithError(err).Error("Could not do close proxy response body")
		}
	}()

	// Pipe the proxy response to the original caller.
	if _, err = io.Copy(w, proxyRes.Body); err != nil {
		p.cfg.logger.WithError(err).Error("Could not copy proxy request body")
		return
	}
}

// Peek into the bytes of an HTTP request's body.
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

// Checks whether the JSON-RPC request is for the Ethereum engine API.
func isEngineAPICall(reqBytes []byte) bool {
	jsonRequest, err := unmarshalRPCObject(reqBytes)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "cannot unmarshal array"):
			return false
		default:
			return false
		}
	}
	return strings.Contains(jsonRequest.Method, "engine_")
}

func unmarshalRPCObject(b []byte) (*jsonRPCObject, error) {
	r := &jsonRPCObject{}
	if err := json.Unmarshal(b, r); err != nil {
		return nil, err
	}
	return r, nil
}
