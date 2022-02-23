package v1

import "github.com/pkg/errors"

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
	// ErrMismatchTerminalBlockHash when the terminal block hash value received via
	// the API mismatches Prysm's configuration value.
	ErrMismatchTerminalBlockHash = errors.New("terminal block hash mismatch")
	// ErrMismatchTerminalTotalDiff when the terminal total difficulty value received via
	// the API mismatches Prysm's configuration value.
	ErrMismatchTerminalTotalDiff = errors.New("terminal total difficulty mismatch")
)
