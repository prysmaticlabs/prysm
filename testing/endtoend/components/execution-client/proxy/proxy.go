// Package proxy provides a proxy middleware for engine API requests between Ethereum
// consensus clients and execution clients accordingly. Allows for configuration of various
// test cases using yaml files as detailed in the README.md of the document.
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"

	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/sirupsen/logrus"
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

type Proxy struct {
	cfg              *config
	address          string
	srv              *http.Server
	interceptor      InterceptorFunc
	backedUpRequests []*http.Request
}

func New(opts ...Option) (*Proxy, error) {
	p := &Proxy{
		cfg: &config{},
	}
	for _, o := range opts {
		if err := o(p); err != nil {
			return nil, err
		}
	}
	if p.cfg.logger != nil {
		p.cfg.logger = logrus.New()
	}
	mux := http.NewServeMux()
	mux.Handle("/", p)
	addr := "127.0.0.1:" + strconv.Itoa(p.cfg.proxyPort)
	srv := &http.Server{
		Handler: mux,
		Addr:    addr,
	}
	p.address = addr
	p.srv = srv
	return p, nil
}

func (p *Proxy) Start(ctx context.Context) error {
	p.srv.BaseContext = func(listener net.Listener) context.Context {
		return ctx
	}
	p.cfg.logger.Infof("Engine proxy now listening on address %s", p.address)
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

// Proxies requests from a consensus client to an execution client, spoofing requests
// and/or responses as desired. Acts as a middleware useful for testing different merge scenarios.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestBytes, err := p.parseRequestBytes(r)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not parse request")
		return
	}
	if p.interceptor != nil && p.interceptor(requestBytes, w, r) {
		return
	}

	for _, rq := range p.backedUpRequests {
		requestB, err := p.parseRequestBytes(rq)
		if err != nil {
			p.cfg.logger.WithError(err).Error("Could not parse request")
			return
		}
		destAddr := "http://" + p.cfg.destinationAddr

		// Create a new proxy request to the execution client.
		url := rq.URL
		url.Host = destAddr
		proxyReq, err := http.NewRequest(rq.Method, destAddr, rq.Body)
		if err != nil {
			p.cfg.logger.WithError(err).Error("Could create new request")
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
			p.cfg.logger.WithError(err).Error("Could not do client proxy")
			return
		}
		buf := bytes.NewBuffer([]byte{})
		// Pipe the proxy response to the original caller.
		if _, err = io.Copy(buf, proxyRes.Body); err != nil {
			p.cfg.logger.WithError(err).Error("Could not copy proxy request body")
			return
		}
		if err = proxyRes.Body.Close(); err != nil {
			p.cfg.logger.WithError(err).Error("Could not do client proxy")
			return
		}
		p.cfg.logger.Infof("Queued Request Response: %s", buf.String())
	}
	p.backedUpRequests = []*http.Request{}

	destAddr := "http://" + p.cfg.destinationAddr

	// Create a new proxy request to the execution client.
	url := r.URL
	url.Host = destAddr
	proxyReq, err := http.NewRequest(r.Method, destAddr, r.Body)
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

	// Pipe the proxy response to the original caller.
	if _, err = io.Copy(w, proxyRes.Body); err != nil {
		p.cfg.logger.WithError(err).Error("Could not copy proxy request body")
		return
	}
	if err = proxyRes.Body.Close(); err != nil {
		p.cfg.logger.WithError(err).Error("Could not do client proxy")
		return
	}
}

func (pn *Proxy) parseRequestBytes(req *http.Request) ([]byte, error) {
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

func (pn *Proxy) checkIfValid(reqBytes []byte) bool {
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

func (pn *Proxy) returnSyncingResponse(reqBytes []byte, w http.ResponseWriter, r *http.Request) {
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
