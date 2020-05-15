// Package debug defines a gRPC server implementation of a debugging service
// which allows for helpful endpoints to debug a beacon node at runtime.
package debug

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Debug service,
// providing RPC endpoints for runtime debugging of a node.
type Server struct {
	GenesisTimeFetcher blockchain.TimeFetcher
	StateGen           *stategen.State
}

// SetLoggingLevel of a beacon node according to a request type,
// either INFO, DEBUG, or TRACE.
func (ds *Server) SetLoggingLevel(ctx context.Context, _ *ptypes.Empty) (*ptypes.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
