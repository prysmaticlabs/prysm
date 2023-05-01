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
	"strings"
	"sync"
	"time"

	gethEngine "github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	gMux "github.com/gorilla/mux"
	builderAPI "github.com/prysmaticlabs/prysm/v4/api/client/builder"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
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

type ExecHeaderResponse struct {
	Version string `json:"version"`
	Data    struct {
		Signature hexutil.Bytes `json:"signature"`
		Message   *BuilderBid   `json:"message"`
	} `json:"data"`
}

type BuilderBid struct {
	Header *gethEngine.ExecutableData `json:"header"`
	Value  builderAPI.Uint256         `json:"value"`
	Pubkey hexutil.Bytes              `json:"pubkey"`
}

type Builder struct {
	cfg          *config
	address      string
	execClient   *gethRPC.Client
	beaconConn   *grpc.ClientConn
	currId       *v1.PayloadIDBytes
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
		"forwardingAddress": p.cfg.destinationUrl.String(),
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
	if p.isBuilderCall(r) {
		p.mux.ServeHTTP(w, r)
		return
	}
	requestBytes, err := parseRequestBytes(r)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not parse request")
		return
	}
	p.handleEngineCalls(requestBytes)
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

	// Pipe the proxy responseGen to the original caller.
	if _, err = io.Copy(w, execRes.Body); err != nil {
		p.cfg.logger.WithError(err).Error("Could not copy proxy request body")
		return
	}
}

func (p *Builder) handleEngineCalls(req []byte) {
	if !isEngineAPICall(req) {
		return
	}
	rpcObj, err := unmarshalRPCObject(req)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not unmarshal rpc object")
		return
	}
	_ = rpcObj
	// do things
}

func (p *Builder) isBuilderCall(req *http.Request) bool {
	return strings.Contains(req.URL.Path, "/eth/v1/builder/")
}

func (p *Builder) handleBuilderCalls(w http.ResponseWriter, req *http.Request) {
	switch {
	case strings.Contains(req.URL.Path, "/eth/v1/builder/validators"):
		p.registerValidators(w, req)
	case strings.Contains(req.URL.Path, "/eth/v1/builder/header"):
		p.handleHeaderRequest(w, req)
	case strings.Contains(req.URL.Path, "/eth/v1/builder/blinded_blocks"):
	case strings.Contains(req.URL.Path, "/eth/v1/builder/status"):
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "unknown request url", http.StatusBadRequest)
	}
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
	time.Unix().Add().Before()
	b, err := p.retrievePendingBlock()
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not retrieve pending block")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fees := calculateFees(b)
	execV2 := gethEngine.BlockToExecutableData(b, fees)
	secKey, err := bls.RandKey()
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not retrieve secret key")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	marshalled, err := json.Marshal(execV2.ExecutionPayload)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not marshal execution payload")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	obj := &v1.ExecutionPayload{}
	err = json.Unmarshal(marshalled, obj)
	if err != nil {
		p.cfg.logger.WithError(err).Error("Could not unmarshal execution payload")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	wObj, err := blocks.WrappedExecutionPayload(obj)
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
	bid := &BuilderBid{
		Header: execV2.ExecutionPayload,
		Value:  builderAPI.Uint256{Int: fees},
		Pubkey: secKey.PublicKey().Marshal(),
	}
	sszBid := &eth.BuilderBid{
		Header: hdr,
		Value:  builderAPI.Uint256{Int: fees}.SSZBytes(),
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
	hdrResp := &ExecHeaderResponse{
		Version: "bellatrix",
		Data: struct {
			Signature hexutil.Bytes `json:"signature"`
			Message   *BuilderBid   `json:"message"`
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
	w.WriteHeader(http.StatusOK)
}

func (p *Builder) handleBlindedBlock(w http.ResponseWriter, req *http.Request) {
	sb := &builderAPI.SignedBlindedBeaconBlockBellatrix{}
	err := json.NewDecoder(req.Body).Decode(sb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	execResp := &builderAPI.ExecPayloadResponse{
		Version: "capella",
		Data:    builderAPI.ExecutionPayload{},
	}
	err = json.NewEncoder(w).Encode(execResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (p *Builder) retrievePendingBlock() (*gethtypes.Block, error) {
	return ethclient.NewClient(p.execClient).BlockByNumber(context.Background(), big.NewInt(-1))
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
func calculateFees(block *gethtypes.Block) *big.Int {
	feesWei := new(big.Int)
	for _, tx := range block.Transactions() {
		minerFee, err := tx.EffectiveGasTip(block.BaseFee())
		if err != nil {
			continue
		}
		feesWei.Add(feesWei, new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), minerFee))
	}
	return feesWei
}
