package debug

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	pbrpc "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// GetForkChoice returns a dump fork choice store.
func (ds *Server) GetForkChoice(ctx context.Context, _ *empty.Empty) (*pbrpc.ForkChoiceResponse, error) {
	return ds.ForkFetcher.ForkChoicer().ForkChoiceDump(ctx)
}
