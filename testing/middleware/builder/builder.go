package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	gMux "github.com/gorilla/mux"
	builderAPI "github.com/prysmaticlabs/prysm/v4/api/client/builder"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/math"
	"github.com/prysmaticlabs/prysm/v4/network"
	"github.com/prysmaticlabs/prysm/v4/network/authorization"
	v1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	statusPath   = "/eth/v1/builder/status"
	registerPath = "/eth/v1/builder/validators"
	headerPath   = "/eth/v1/builder/header/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}"
	blindedPath  = "/eth/v1/builder/blinded_blocks"

	// ForkchoiceUpdatedMethod v1 request string for JSON-RPC.
	ForkchoiceUpdatedMethod = "engine_forkchoiceUpdatedV1"
	// ForkchoiceUpdatedMethodV2 v2 request string for JSON-RPC.
	ForkchoiceUpdatedMethodV2 = "engine_forkchoiceUpdatedV2"
	// GetPayloadMethod v1 request string for JSON-RPC.
	GetPayloadMethod = "engine_getPayloadV1"
	// GetPayloadMethodV2 v2 request string for JSON-RPC.
	GetPayloadMethodV2 = "engine_getPayloadV2"
	// ExchangeTransitionConfigurationMethod v1 request string for JSON-RPC.
)

var (
	defaultBuilderHost = "127.0.0.1"
	defaultBuilderPort = 8551
)

type jsonRPCObject struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      uint64        `json:"id"`
	Result  interface{}   `json:"result"`
}

type ForkchoiceUpdatedResponse struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      uint64        `json:"id"`
	Result  struct {
		Status    *v1.PayloadStatus  `json:"payloadStatus"`
		PayloadId *v1.PayloadIDBytes `json:"payloadId"`
	} `json:"result"`
}

type ExecPayloadResponse struct {
	Version string               `json:"version"`
	Data    *v1.ExecutionPayload `json:"data"`
}

type ExecHeaderResponseCapella struct {
	Version string `json:"version"`
	Data    struct {
		Signature hexutil.Bytes                 `json:"signature"`
		Message   *builderAPI.BuilderBidCapella `json:"message"`
	} `json:"data"`
}

type Builder struct {
	cfg          *config
	address      string
	execClient   *gethRPC.Client
	beaconConn   *grpc.ClientConn
	currId       *v1.PayloadIDBytes
	currPayload  interfaces.ExecutionData
	mux          *gMux.Router
	validatorMap map[string]*eth.ValidatorRegistrationV1
	srv          *http.Server
	lock         sync.RWMutex
}

// New creates a proxy server forwarding requests from a consensus client to an execution client.
func New(opts ...Option) (*Builder, error) {
	p := &Builder{
		cfg: &config{
			builderPort: defaultBuilderPort,
			builderHost: defaultBuilderHost,
			logger:      logrus.New(),
		},
	}
	for _, o := range opts {
		if err := o(p); err != nil {
			return nil, err
		}
	}
	if p.cfg.destinationUrl == nil {
		return nil, errors.New("must provide a destination address for request proxying")
	}
	endpoint := network.HttpEndpoint(p.cfg.destinationUrl.String())
	endpoint.Auth.Method = authorization.Bearer
	endpoint.Auth.Value = p.cfg.secret
	execClient, err := network.NewExecutionRPCClient(context.Background(), endpoint)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/", p)
	router := gMux.NewRouter()
	router.HandleFunc(statusPath, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	router.HandleFunc(registerPath, p.registerValidators)
	router.HandleFunc(headerPath, p.handleHeaderRequest)
	router.HandleFunc(blindedPath, p.handleBlindedBlock)
	addr := fmt.Sprintf("%s:%d", p.cfg.builderHost, p.cfg.builderPort)
	srv := &http.Server{
		Handler:           mux,
		Addr:              addr,
		ReadHeaderTimeout: time.Second,
	}
	conn, err := grpc.DialContext(context.Background(), p.cfg.beaconUrl.String(), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	p.beaconConn = conn
	p.address = addr
	p.srv = srv
	p.execClient = execClient
	p.validatorMap = map[string]*eth.ValidatorRegistrationV1{}
	p.mux = router
	return p, nil
}

// Address for the proxy server.
func (p *Builder) Address() string {
	return p.address
}

// Start a proxy server.
func (p *Builder) Start(ctx context.Context) error {
	p.srv.BaseContext = func(listener net.Listener) context.Context {
		return ctx
	}
	p.cfg.logger.WithFields(logrus.Fields{
		"executionAddress": p.cfg.destinationUrl.String(),
	}).Infof("Builder now listening on address %s", p.address)
	go func() {
		if err := p.srv.ListenAndServe(); err != nil {
			p.cfg.logger.Error(err)
		}
	}()
	for {
		<-ctx.Done()
		return p.srv.Shutdown(context.Background())
	}
}

// ServeHTTP requests from a consensus client to an execution client, modifying in-flight requests
// and/or responses as desired. It also processes any backed-up requests.
func (p *Builder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.cfg.logger.Infof("Received %s request from beacon with url: %s", r.Method, r.URL.Path)
	if p.isBuilderCall(r) {
		p.mux.ServeHTTP(w, r)
		return
	}
	requestBytes, err := parseRequestBytes(r)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not parse request")
		return
	}
	execRes, err := p.sendHttpRequest(r, requestBytes)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not forward request")
		return
	}
	p.cfg.logger.Infof("Received response for %s request with method %s from %s", r.Method, r.Method, p.cfg.destinationUrl.String())

	defer func() {
		if err = execRes.Body.Close(); err != nil {
			p.cfg.logger.WithError(err).Error("Could not do close proxy responseGen body")
		}
	}()

	buf := bytes.NewBuffer([]byte{})
	if _, err = io.Copy(buf, execRes.Body); err != nil {
		p.cfg.logger.WithError(err).Error("Could not copy proxy request body")
		return
	}
	byteResp := bytesutil.SafeCopyBytes(buf.Bytes())
	p.handleEngineCalls(requestBytes, byteResp)
	// Pipe the proxy responseGen to the original caller.
	if _, err = io.Copy(w, buf); err != nil {
		p.cfg.logger.WithError(err).Error("Could not copy proxy request body")
		return
	}
}

func (p *Builder) handleEngineCalls(req []byte, resp []byte) {
	if !isEngineAPICall(req) {
		return
	}
	rpcObj, err := unmarshalRPCObject(req)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not unmarshal rpc object")
		return
	}
	p.cfg.logger.Infof("Received engine call %s", rpcObj.Method)
	switch rpcObj.Method {
	case ForkchoiceUpdatedMethod, ForkchoiceUpdatedMethodV2:
		result := &ForkchoiceUpdatedResponse{}
		err = json.Unmarshal(resp, result)
		if err != nil {
			p.cfg.logger.Errorf("Could not unmarshal fcu: %v", err)
			return
		}
		p.currId = result.Result.PayloadId
		p.cfg.logger.Infof("Received payload id of %#x", result.Result.PayloadId)
	}
}

func (p *Builder) isBuilderCall(req *http.Request) bool {
	return strings.Contains(req.URL.Path, "/eth/v1/builder/")
}

func (p *Builder) registerValidators(w http.ResponseWriter, req *http.Request) {
	registrations := []builderAPI.SignedValidatorRegistration{}
	if err := json.NewDecoder(req.Body).Decode(&registrations); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	for _, r := range registrations {
		msg := r.Message
		p.validatorMap[string(r.Message.Pubkey)] = msg
	}
	// TODO: Verify Signatures from validators
	w.WriteHeader(http.StatusOK)
}

func (p *Builder) handleHeaderRequest(w http.ResponseWriter, req *http.Request) {
	urlParams := gMux.Vars(req)
	pHash := urlParams["parent_hash"]
	if pHash == "" {
		http.Error(w, "no valid parent hash", http.StatusBadRequest)
		return
	}
	reqSlot := urlParams["slot"]
	if reqSlot == "" {
		http.Error(w, "no valid slot provided", http.StatusBadRequest)
		return
	}
	slot, err := strconv.Atoi(reqSlot)
	if err != nil {
		http.Error(w, "invalid slot provided", http.StatusBadRequest)
		return
	}
	ax := types.Slot(slot)
	currEpoch := types.Epoch(ax / params.BeaconConfig().SlotsPerEpoch)
	if currEpoch >= params.BeaconConfig().CapellaForkEpoch {

	}

	b, err := p.retrievePendingBlock()
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not retrieve pending block")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	secKey, err := bls.RandKey()
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not retrieve secret key")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	wObj, err := blocks.WrappedExecutionPayload(b)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not wrap execution payload")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hdr, err := blocks.PayloadToHeader(wObj)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not make payload into header")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	gEth := big.NewInt(int64(params.BeaconConfig().GweiPerEth))
	weiEth := gEth.Mul(gEth, gEth)
	val := builderAPI.Uint256{Int: weiEth}
	wrappedHdr := &builderAPI.ExecutionPayloadHeader{ExecutionPayloadHeader: hdr}
	bid := &builderAPI.BuilderBid{
		Header: wrappedHdr,
		Value:  val,
		Pubkey: secKey.PublicKey().Marshal(),
	}
	sszBid := &eth.BuilderBid{
		Header: hdr,
		Value:  val.SSZBytes(),
		Pubkey: secKey.PublicKey().Marshal(),
	}
	d, err := signing.ComputeDomain(params.BeaconConfig().DomainApplicationBuilder,
		nil, /* fork version */
		nil /* genesis val root */)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not compute the domain")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rt, err := signing.ComputeSigningRoot(sszBid, d)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not compute the signing root")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sig := secKey.Sign(rt[:])
	hdrResp := &builderAPI.ExecHeaderResponse{
		Version: "bellatrix",
		Data: struct {
			Signature hexutil.Bytes          `json:"signature"`
			Message   *builderAPI.BuilderBid `json:"message"`
		}{
			Signature: sig.Marshal(),
			Message:   bid,
		},
	}

	err = json.NewEncoder(w).Encode(hdrResp)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.currPayload = wObj
	w.WriteHeader(http.StatusOK)
}

func (p *Builder) handleHeadeRequestCapella(w http.ResponseWriter) {
	b, err := p.retrievePendingBlockCapella()
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not retrieve pending block")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	secKey, err := bls.RandKey()
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not retrieve secret key")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v := big.NewInt(0).SetBytes(bytesutil.ReverseByteOrder(b.Value))
	wObj, err := blocks.WrappedExecutionPayloadCapella(b.Payload, math.WeiToGwei(v))
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not wrap execution payload")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hdr, err := blocks.PayloadToHeaderCapella(wObj)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not make payload into header")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	val := builderAPI.Uint256{Int: v}
	wrappedHdr := &builderAPI.ExecutionPayloadHeaderCapella{ExecutionPayloadHeaderCapella: hdr}
	bid := &builderAPI.BuilderBidCapella{
		Header: wrappedHdr,
		Value:  val,
		Pubkey: secKey.PublicKey().Marshal(),
	}
	sszBid := &eth.BuilderBidCapella{
		Header: hdr,
		Value:  val.SSZBytes(),
		Pubkey: secKey.PublicKey().Marshal(),
	}
	d, err := signing.ComputeDomain(params.BeaconConfig().DomainApplicationBuilder,
		nil, /* fork version */
		nil /* genesis val root */)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not compute the domain")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rt, err := signing.ComputeSigningRoot(sszBid, d)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not compute the signing root")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sig := secKey.Sign(rt[:])
	hdrResp := &ExecHeaderResponseCapella{
		Version: "capella",
		Data: struct {
			Signature hexutil.Bytes                 `json:"signature"`
			Message   *builderAPI.BuilderBidCapella `json:"message"`
		}{
			Signature: sig.Marshal(),
			Message:   bid,
		},
	}

	err = json.NewEncoder(w).Encode(hdrResp)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.currPayload = wObj
	w.WriteHeader(http.StatusOK)
}

func (p *Builder) handleBlindedBlock(w http.ResponseWriter, req *http.Request) {
	sb := &builderAPI.SignedBlindedBeaconBlockBellatrix{
		SignedBlindedBeaconBlockBellatrix: &eth.SignedBlindedBeaconBlockBellatrix{},
	}
	err := json.NewDecoder(req.Body).Decode(sb)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not decode blinded block")
		// TODO: Allow the method to unmarshal blinded blocks correctly
	}
	if p.currPayload == nil {
		p.cfg.logger.Error("No payload is cached")
		http.Error(w, "payload not found", http.StatusInternalServerError)
		return
	}
	if payload, err := p.currPayload.PbCapella(); err == nil {
		convertedPayload, err := builderAPI.FromProtoCapella(payload)
		if err != nil {
			p.cfg.logger.WithError(err).Error("Could not convert the payload")
			http.Error(w, "payload not found", http.StatusInternalServerError)
			return
		}
		execResp := &builderAPI.ExecPayloadResponseCapella{
			Version: "capella",
			Data:    convertedPayload,
		}
		err = json.NewEncoder(w).Encode(execResp)
		if err != nil {
			p.cfg.logger.WithError(err).Error("Could not encode full payload response")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	bellPayload, err := p.currPayload.PbBellatrix()
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not retrieve the payload")
		http.Error(w, "payload not found", http.StatusInternalServerError)
		return
	}
	convertedPayload, err := builderAPI.FromProto(bellPayload)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not convert the payload")
		http.Error(w, "payload not found", http.StatusInternalServerError)
		return
	}
	execResp := &builderAPI.ExecPayloadResponse{
		Version: "bellatrix",
		Data:    convertedPayload,
	}
	err = json.NewEncoder(w).Encode(execResp)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not encode full payload response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (p *Builder) retrievePendingBlock() (*v1.ExecutionPayload, error) {
	result := &v1.ExecutionPayload{}
	if p.currId == nil {
		return nil, errors.New("no payload id is cached")
	}
	err := p.execClient.CallContext(context.Background(), result, GetPayloadMethod, *p.currId)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Builder) retrievePendingBlockCapella() (*v1.ExecutionPayloadCapellaWithValue, error) {
	result := &v1.ExecutionPayloadCapellaWithValue{}
	if p.currId == nil {
		return nil, errors.New("no payload id is cached")
	}
	err := p.execClient.CallContext(context.Background(), result, GetPayloadMethodV2, *p.currId)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Builder) sendHttpRequest(req *http.Request, requestBytes []byte) (*http.Response, error) {
	proxyReq, err := http.NewRequest(req.Method, p.cfg.destinationUrl.String(), req.Body)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not create new request")
		return nil, err
	}

	// Set the modified request as the proxy request body.
	proxyReq.Body = ioutil.NopCloser(bytes.NewBuffer(requestBytes))

	// Required proxy headers for forwarding JSON-RPC requests to the execution client.
	proxyReq.Header.Set("Host", req.Host)
	proxyReq.Header.Set("X-Forwarded-For", req.RemoteAddr)
	proxyReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	if p.cfg.secret != "" {
		client = network.NewHttpClientWithSecret(p.cfg.secret)
	}
	proxyRes, err := client.Do(proxyReq)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not forward request to destination server")
		return nil, err
	}
	return proxyRes, nil
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

// An estimation of the fees gained by the block proposer, a real
// estimation requires transaction receipts for the actual gas used.
