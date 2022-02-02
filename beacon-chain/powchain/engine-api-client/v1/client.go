// Package v1 defines an API client for the engine API defined in https://github.com/ethereum/execution-apis.
// This client is used for the Prysm consensus node to connect to execution node as part of
// the Ethereum proof-of-stake machinery.
package v1

import (
	"context"
	"net/url"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
)

const (
	// NewPayloadMethod v1 request string for JSON-RPC.
	NewPayloadMethod = "engine_newPayloadV1"
	// ForkchoiceUpdatedMethod v1 request string for JSON-RPC.
	ForkchoiceUpdatedMethod = "engine_forkchoiceUpdatedV1"
	// GetPayloadMethod v1 request string for JSON-RPC.
	GetPayloadMethod = "engine_getPayloadV1"
)

var (
	// ErrParse corresponds to JSON-RPC code -32700.
	ErrParse = errors.New("invalid JSON was received by the server")
	// ErrInvalidRequest corresponds to JSON-RPC code -32600.
	ErrInvalidRequest = errors.New("JSON sent is not valid request object")
	// ErrMethodNotFound corresponds to JSON-RPC code -32601.
	ErrMethodNotFound = errors.New("method not found")
	// ErrInvalidParams corresponds to JSON-RPC code -32602.
	ErrInvalidParams = errors.New("invalid method parameter(s)")
	// ErrInternal corresponds to JSON-RPC code -32603.
	ErrInternal = errors.New("internal JSON-RPC error")
	// ErrServer corresponds to JSON-RPC code -32000.
	ErrServer = errors.New("client error while processing request")
	// ErrUnknownPayload corresponds to JSON-RPC code -32001.
	ErrUnknownPayload = errors.New("payload does not exist or is not available")
	// ErrUnsupportedScheme for unsupported URL schemes.
	ErrUnsupportedScheme = errors.New("unsupported url scheme, only http(s) and ipc are supported")
)

// ForkchoiceUpdatedResponse is the response kind received by the
// engine_forkchoiceUpdatedV1 endpoint.
type ForkchoiceUpdatedResponse struct {
	Status    *pb.PayloadStatus `json:"status"`
	PayloadId [8]byte           `json:"payloadId"`
}

// Client defines a new engine API client for the Prysm consensus node
// to interact with an Ethereum execution node.
type Client struct {
	cfg *config
	rpc *rpc.Client
}

// New returns a ready, engine API client from an endpoint and configuration options.
// Only http(s) and ipc (inter-process communication) URL schemes are supported.
func New(ctx context.Context, endpoint string, opts ...Option) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	c := &Client{
		cfg: defaultConfig(),
	}
	switch u.Scheme {
	case "http", "https":
		c.rpc, err = rpc.DialHTTPWithClient(endpoint, c.cfg.httpClient)
	case "":
		c.rpc, err = rpc.DialIPC(ctx, endpoint)
	default:
		return nil, errors.Wrapf(ErrUnsupportedScheme, "%q", u.Scheme)
	}
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// NewPayload --
func (c *Client) NewPayload(ctx context.Context, payload *pb.ExecutionPayload) (*pb.PayloadStatus, error) {
	return nil, errors.New("unimplemented")
}

// ForkchoiceUpdated --
func (c *Client) ForkchoiceUpdated(
	ctx context.Context, state *pb.ForkchoiceState, attrs *pb.PayloadAttributes,
) (*ForkchoiceUpdatedResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetPayload --
func (c *Client) GetPayload(ctx context.Context, payloadId [8]byte) (*pb.ExecutionPayload, error) {
	return nil, errors.New("unimplemented")
}
