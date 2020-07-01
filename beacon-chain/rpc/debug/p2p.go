package debug

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetPeer returns the data known about the peer defined by the provided peer id.
func (ds *Server) GetPeer(ctx context.Context, peerReq *ethpb.PeerRequest) (*pbrpc.DebugPeerResponse, error) {
	pid, err := peer.Decode(peerReq.PeerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Unable to parse provided peer id: %v", err)
	}
	return ds.getPeer(pid)
}

// ListPeers returns all peers known to the host node, irregardless of if they are connected/
// disconnected.
func (ds *Server) ListPeers(ctx context.Context, _ *types.Empty) (*pbrpc.DebugPeerResponses, error) {
	responses := []*pbrpc.DebugPeerResponse{}
	for _, pid := range ds.PeersFetcher.Peers().All() {
		resp, err := ds.getPeer(pid)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return &pbrpc.DebugPeerResponses{Responses: responses}, nil
}

func (ds *Server) getPeer(pid peer.ID) (*pbrpc.DebugPeerResponse, error) {
	peers := ds.PeersFetcher.Peers()
	peerStore := ds.PeerManager.Host().Peerstore()
	addr, err := peers.Address(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	dir, err := peers.Direction(pid)
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
	connState, err := peers.ConnectionState(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	record, err := peers.ENR(pid)
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
	metadata, err := peers.Metadata(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	protocols, err := peerStore.GetProtocols(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	resp, err := peers.BadResponses(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}

	rawPversion, err := peerStore.Get(pid, "ProtocolVersion")
	pVersion, ok := rawPversion.(string)
	if err != nil || !ok {
		pVersion = ""
	}
	rawAversion, err := peerStore.Get(pid, "AgentVersion")
	aVersion, ok := rawAversion.(string)
	if err != nil || !ok {
		aVersion = ""
	}
	peerInfo := &pbrpc.DebugPeerResponse_PeerInfo{
		Metadata:        metadata,
		Protocols:       protocols,
		FaultCount:      uint64(resp),
		ProtocolVersion: pVersion,
		AgentVersion:    aVersion,
		PeerLatency:     uint64(peerStore.LatencyEWMA(pid).Milliseconds()),
	}
	addresses := peerStore.Addrs(pid)
	stringAddrs := []string{}
	if addr != nil {
		stringAddrs = append(stringAddrs, addr.String())
	}
	for _, a := range addresses {
		// Do not double count address
		if addr != nil && addr.String() == a.String() {
			continue
		}
		stringAddrs = append(stringAddrs, a.String())
	}
	pStatus, err := peers.ChainState(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	lastUpdated, err := peers.ChainStateLastUpdated(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	unixTime := uint64(0)
	if !lastUpdated.IsZero() {
		unixTime = uint64(lastUpdated.Unix())
	}
	return &pbrpc.DebugPeerResponse{
		ListeningAddresses: stringAddrs,
		Direction:          pbDirection,
		ConnectionState:    ethpb.ConnectionState(connState),
		PeerId:             pid.String(),
		Enr:                enr,
		PeerInfo:           peerInfo,
		PeerStatus:         pStatus,
		LastUpdated:        unixTime,
	}, nil
}
