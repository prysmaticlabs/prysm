package node

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct{}

func (s Server) GetSyncStatus(ctx context.Context, empty *empty.Empty) (*v2.SyncStatus, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetGenesis(ctx context.Context, empty *empty.Empty) (*v2.Genesis, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetVersion(ctx context.Context, empty *empty.Empty) (*v2.Version, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ListImplementedServices(ctx context.Context, empty *empty.Empty) (*v2.ImplementedServices, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetHost(ctx context.Context, empty *empty.Empty) (*v2.HostData, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetPeer(ctx context.Context, request *v2.PeerRequest) (*v2.Peer, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ListPeers(ctx context.Context, empty *empty.Empty) (*v2.Peers, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}
