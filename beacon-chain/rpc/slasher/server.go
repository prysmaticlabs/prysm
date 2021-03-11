// Package debug defines a gRPC server implementation of a debugging service
// which allows for helpful endpoints to debug a beacon node at runtime, this server is
// gated behind the feature flag --enable-debug-rpc-endpoints.
package slasher

// Server defines a server implementation of the gRPC Debug service,
// providing RPC endpoints for runtime debugging of a node, this server is
// gated behind the feature flag --enable-debug-rpc-endpoints.
type Server struct {
}
