package node

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/peers/peerdata"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"go.opencensus.io/trace"
)

var (
	stateConnecting    = ethpb.ConnectionState_CONNECTING.String()
	stateConnected     = ethpb.ConnectionState_CONNECTED.String()
	stateDisconnecting = ethpb.ConnectionState_DISCONNECTING.String()
	stateDisconnected  = ethpb.ConnectionState_DISCONNECTED.String()
	directionInbound   = ethpb.PeerDirection_INBOUND.String()
	directionOutbound  = ethpb.PeerDirection_OUTBOUND.String()
)

// GetSyncStatus requests the beacon node to describe if it's currently syncing or not, and
// if it is, what block it is up to.
func (s *Server) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "node.GetSyncStatus")
	defer span.End()

	isOptimistic, err := s.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		http2.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	headSlot := s.HeadFetcher.HeadSlot()
	response := &SyncStatusResponse{
		Data: &SyncStatusResponseData{
			HeadSlot:     strconv.FormatUint(uint64(headSlot), 10),
			SyncDistance: strconv.FormatUint(uint64(s.GenesisTimeFetcher.CurrentSlot()-headSlot), 10),
			IsSyncing:    s.SyncChecker.Syncing(),
			IsOptimistic: isOptimistic,
			ElOffline:    !s.ExecutionChainInfoFetcher.ExecutionClientConnected(),
		},
	}
	http2.WriteJson(w, response)
}

// GetIdentity retrieves data about the node's network presence.
func (s *Server) GetIdentity(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "node.GetIdentity")
	defer span.End()

	peerId := s.PeerManager.PeerID().Pretty()

	serializedEnr, err := p2p.SerializeENR(s.PeerManager.ENR())
	if err != nil {
		http2.HandleError(w, "Could not obtain enr: "+err.Error(), http.StatusInternalServerError)
		return
	}
	enr := "enr:" + serializedEnr

	sourcep2p := s.PeerManager.Host().Addrs()
	p2pAddresses := make([]string, len(sourcep2p))
	for i := range sourcep2p {
		p2pAddresses[i] = sourcep2p[i].String() + "/p2p/" + peerId
	}

	sourceDisc, err := s.PeerManager.DiscoveryAddresses()
	if err != nil {
		http2.HandleError(w, "Could not obtain discovery address: "+err.Error(), http.StatusInternalServerError)
		return
	}
	discoveryAddresses := make([]string, len(sourceDisc))
	for i := range sourceDisc {
		discoveryAddresses[i] = sourceDisc[i].String()
	}

	meta := &Metadata{
		SeqNumber: strconv.FormatUint(s.MetadataProvider.MetadataSeq(), 10),
		Attnets:   hexutil.Encode(s.MetadataProvider.Metadata().AttnetsBitfield()),
	}

	resp := &GetIdentityResponse{
		Data: &Identity{
			PeerId:             peerId,
			Enr:                enr,
			P2PAddresses:       p2pAddresses,
			DiscoveryAddresses: discoveryAddresses,
			Metadata:           meta,
		},
	}
	http2.WriteJson(w, resp)
}

// GetPeer retrieves data about the given peer.
func (s *Server) GetPeer(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "node.GetPeer")
	defer span.End()

	rawId := r.URL.Query().Get("peer_id")
	if rawId == "" {
		http2.HandleError(w, "peer_id query parameter is required", http.StatusBadRequest)
		return
	}

	peerStatus := s.PeersFetcher.Peers()
	id, err := peer.Decode(rawId)
	if err != nil {
		http2.HandleError(w, "Invalid peer ID: "+err.Error(), http.StatusBadRequest)
		return
	}
	enr, err := peerStatus.ENR(id)
	if err != nil {
		if errors.Is(err, peerdata.ErrPeerUnknown) {
			http2.HandleError(w, "Peer not found: "+err.Error(), http.StatusNotFound)
			return
		}
		http2.HandleError(w, "Could not obtain ENR: "+err.Error(), http.StatusInternalServerError)
		return
	}
	serializedEnr, err := p2p.SerializeENR(enr)
	if err != nil {
		http2.HandleError(w, "Could not obtain ENR: "+err.Error(), http.StatusInternalServerError)
		return
	}
	p2pAddress, err := peerStatus.Address(id)
	if err != nil {
		http2.HandleError(w, "Could not obtain address: "+err.Error(), http.StatusInternalServerError)
		return
	}
	state, err := peerStatus.ConnectionState(id)
	if err != nil {
		http2.HandleError(w, "Could not obtain connection state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	direction, err := peerStatus.Direction(id)
	if err != nil {
		http2.HandleError(w, "Could not obtain direction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if eth.PeerDirection(direction) == eth.PeerDirection_UNKNOWN {
		http2.HandleError(w, "Peer not found", http.StatusNotFound)
		return
	}

	v1ConnState := migration.V1Alpha1ConnectionStateToV1(eth.ConnectionState(state))
	v1PeerDirection, err := migration.V1Alpha1PeerDirectionToV1(eth.PeerDirection(direction))
	if err != nil {
		http2.HandleError(w, "Could not handle peer direction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := &GetPeerResponse{
		Data: &Peer{
			PeerId:    rawId,
			Enr:       "enr:" + serializedEnr,
			Address:   p2pAddress.String(),
			State:     strings.ToLower(v1ConnState.String()),
			Direction: strings.ToLower(v1PeerDirection.String()),
		},
	}
	http2.WriteJson(w, resp)
}

// GetPeers retrieves data about the node's network peers.
func (s *Server) GetPeers(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "node.GetPeers")
	defer span.End()

	states := r.URL.Query()["state"]
	directions := r.URL.Query()["direction"]

	peerStatus := s.PeersFetcher.Peers()
	emptyStateFilter, emptyDirectionFilter := handleEmptyFilters(states, directions)

	if emptyStateFilter && emptyDirectionFilter {
		allIds := peerStatus.All()
		allPeers := make([]*Peer, 0, len(allIds))
		for _, id := range allIds {
			p, err := peerInfo(peerStatus, id)
			if err != nil {
				http2.HandleError(w, "Could not get peer info: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if p == nil {
				continue
			}
			allPeers = append(allPeers, p)
		}
		resp := &GetPeersResponse{Data: allPeers}
		http2.WriteJson(w, resp)
		return
	}

	var stateIds []peer.ID
	if emptyStateFilter {
		stateIds = peerStatus.All()
	} else {
		for _, stateFilter := range states {
			normalized := strings.ToUpper(stateFilter)
			if normalized == stateConnecting {
				ids := peerStatus.Connecting()
				stateIds = append(stateIds, ids...)
				continue
			}
			if normalized == stateConnected {
				ids := peerStatus.Connected()
				stateIds = append(stateIds, ids...)
				continue
			}
			if normalized == stateDisconnecting {
				ids := peerStatus.Disconnecting()
				stateIds = append(stateIds, ids...)
				continue
			}
			if normalized == stateDisconnected {
				ids := peerStatus.Disconnected()
				stateIds = append(stateIds, ids...)
				continue
			}
		}
	}

	var directionIds []peer.ID
	if emptyDirectionFilter {
		directionIds = peerStatus.All()
	} else {
		for _, directionFilter := range directions {
			normalized := strings.ToUpper(directionFilter)
			if normalized == directionInbound {
				ids := peerStatus.Inbound()
				directionIds = append(directionIds, ids...)
				continue
			}
			if normalized == directionOutbound {
				ids := peerStatus.Outbound()
				directionIds = append(directionIds, ids...)
				continue
			}
		}
	}

	var filteredIds []peer.ID
	for _, stateId := range stateIds {
		for _, directionId := range directionIds {
			if stateId.Pretty() == directionId.Pretty() {
				filteredIds = append(filteredIds, stateId)
				break
			}
		}
	}
	filteredPeers := make([]*Peer, 0, len(filteredIds))
	for _, id := range filteredIds {
		p, err := peerInfo(peerStatus, id)
		if err != nil {
			http2.HandleError(w, "Could not get peer info: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if p == nil {
			continue
		}
		filteredPeers = append(filteredPeers, p)
	}

	resp := &GetPeersResponse{Data: filteredPeers}
	http2.WriteJson(w, resp)
}

// GetPeerCount retrieves number of known peers.
func (s *Server) GetPeerCount(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "node.PeerCount")
	defer span.End()

	peerStatus := s.PeersFetcher.Peers()

	resp := &GetPeerCountResponse{
		Data: &PeerCount{
			Disconnected:  strconv.FormatInt(int64(len(peerStatus.Disconnected())), 10),
			Connecting:    strconv.FormatInt(int64(len(peerStatus.Connecting())), 10),
			Connected:     strconv.FormatInt(int64(len(peerStatus.Connected())), 10),
			Disconnecting: strconv.FormatInt(int64(len(peerStatus.Disconnecting())), 10),
		},
	}
	http2.WriteJson(w, resp)
}

// GetVersion requests that the beacon node identify information about its implementation in a
// format similar to a HTTP User-Agent field.
func (*Server) GetVersion(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "node.GetVersion")
	defer span.End()

	v := fmt.Sprintf("Prysm/%s (%s %s)", version.SemanticVersion(), runtime.GOOS, runtime.GOARCH)
	resp := &GetVersionResponse{
		Data: &Version{
			Version: v,
		},
	}
	http2.WriteJson(w, resp)
}

// GetHealth returns node health status in http status codes. Useful for load balancers.
func (s *Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "node.GetHealth")
	defer span.End()

	ok, _, syncingStatus := shared.UintFromQuery(w, r, "syncing_status")
	if !ok || http.StatusText(int(syncingStatus)) == "" {
		http2.HandleError(w, "syncing_status is not a valid HTTP status code", http.StatusBadRequest)
		return
	}

	if s.SyncChecker.Synced() {
		return
	}
	if s.SyncChecker.Syncing() || s.SyncChecker.Initialized() {
		w.WriteHeader(http.StatusPartialContent)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	return
}

func handleEmptyFilters(states []string, directions []string) (emptyState, emptyDirection bool) {
	emptyState = true
	for _, stateFilter := range states {
		normalized := strings.ToUpper(stateFilter)
		filterValid := normalized == stateConnecting || normalized == stateConnected ||
			normalized == stateDisconnecting || normalized == stateDisconnected
		if filterValid {
			emptyState = false
			break
		}
	}

	emptyDirection = true
	for _, directionFilter := range directions {
		normalized := strings.ToUpper(directionFilter)
		filterValid := normalized == directionInbound || normalized == directionOutbound
		if filterValid {
			emptyDirection = false
			break
		}
	}

	return emptyState, emptyDirection
}

func peerInfo(peerStatus *peers.Status, id peer.ID) (*Peer, error) {
	enr, err := peerStatus.ENR(id)
	if err != nil {
		if errors.Is(err, peerdata.ErrPeerUnknown) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "could not obtain ENR")
	}
	var serializedEnr string
	if enr != nil {
		serializedEnr, err = p2p.SerializeENR(enr)
		if err != nil {
			return nil, errors.Wrap(err, "could not serialize ENR")
		}
	}
	address, err := peerStatus.Address(id)
	if err != nil {
		if errors.Is(err, peerdata.ErrPeerUnknown) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "could not obtain address")
	}
	connectionState, err := peerStatus.ConnectionState(id)
	if err != nil {
		if errors.Is(err, peerdata.ErrPeerUnknown) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "could not obtain connection state")
	}
	direction, err := peerStatus.Direction(id)
	if err != nil {
		if errors.Is(err, peerdata.ErrPeerUnknown) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "could not obtain direction")
	}
	if eth.PeerDirection(direction) == eth.PeerDirection_UNKNOWN {
		return nil, nil
	}
	v1ConnState := migration.V1Alpha1ConnectionStateToV1(eth.ConnectionState(connectionState))
	v1PeerDirection, err := migration.V1Alpha1PeerDirectionToV1(eth.PeerDirection(direction))
	if err != nil {
		return nil, errors.Wrapf(err, "could not handle peer direction")
	}
	p := &Peer{
		PeerId:    id.Pretty(),
		State:     strings.ToLower(v1ConnState.String()),
		Direction: strings.ToLower(v1PeerDirection.String()),
	}
	if address != nil {
		p.Address = address.String()
	}
	if serializedEnr != "" {
		p.Enr = "enr:" + serializedEnr
	}

	return p, nil
}
