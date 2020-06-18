package debug

import (
	"context"

	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetPeer returns the data known about the peer defined by the provided peer id.
func (ds *Server) GetPeer(ctx context.Context, peerReq *ethpb.PeerRequest) (*pbrpc.DebugPeerResponse, error) {
	pid, err := peer.Decode(peerReq.PeerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Unable to parse provided peer id: %v", err)
	}
	addr, err := ds.PeersFetcher.Peers().Address(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	dir, err := ds.PeersFetcher.Peers().Direction(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	pbDirection := ethpb.PeerDirection_UNKNOWN
	switch dir {
	case network.DirInbound:
		pbDirection = ethpb.PeerDirection_INBOUND
	case network.DirOutbound:
		pbDirection = ethpb.PeerDirection_OUTBOUND
	}
	connState, err := ds.PeersFetcher.Peers().ConnectionState(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	record, err := ds.PeersFetcher.Peers().ENR(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	enr := ""
	if record != nil {
		enr, err = p2p.SerializeENR(record)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Unable to serialize enr: %v", err)
		}
	}
	peerInfo := &pbrpc.DebugPeerResponse_PeerInfo{
		Metadata:             nil,
		Protocols:            nil,
		FaultCount:           0,
		ProtocolVersion:      "",
		AgentVersion:         "",
		PeerLatency:          0,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	res := &pbrpc.DebugPeerResponse{
		ListeningAddresses:   nil,
		Direction:            0,
		ConnectionState:      0,
		PeerId:               "",
		Enr:                  "",
		PeerInfo:             nil,
		PeerStatus:           nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
}
