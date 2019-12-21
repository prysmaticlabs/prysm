package peers_test

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestStatus(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)
	if p == nil {
		t.Fatalf("p not created")
	}
	if p.MaxBadResponses() != maxBadResponses {
		t.Errorf("maxBadResponses incorrect value: expected %v, received %v", maxBadResponses, p.MaxBadResponses())
	}
}

func TestPeerExplicitAdd(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	id, err := peer.IDB58Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	if err != nil {
		t.Fatalf("Failed to create ID: %v", err)
	}
	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	if err != nil {
		t.Fatalf("Failed to create address: %v", err)
	}
	direction := network.DirInbound
	p.Add(id, address, direction)

	resAddress, err := p.Address(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resAddress != address {
		t.Errorf("Unexpected address: expected %v, received %v", address, resAddress)
	}

	resDirection, err := p.Direction(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resDirection != direction {
		t.Errorf("Unexpected direction: expected %v, received %v", direction, resDirection)
	}

	// Update with another explicit add
	address2, err := ma.NewMultiaddr("/ip4/52.23.23.253/tcp/30000/ipfs/QmfAgkmjiZNZhr2wFN9TwaRgHouMTBT6HELyzE5A3BT2wK/p2p-circuit")
	if err != nil {
		t.Fatalf("Failed to create address: %v", err)
	}
	direction2 := network.DirOutbound
	p.Add(id, address2, direction2)

	resAddress2, err := p.Address(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resAddress2 != address2 {
		t.Errorf("Unexpected address: expected %v, received %v", address2, resAddress2)
	}

	resDirection2, err := p.Direction(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resDirection2 != direction2 {
		t.Errorf("Unexpected direction: expected %v, received %v", direction2, resDirection2)
	}
}

func TestErrUnknownPeer(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	id, err := peer.IDB58Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Address(id)
	if err != peers.ErrPeerUnknown {
		t.Errorf("Unexpected error: expected %v, received %v", peers.ErrPeerUnknown, err)
	}

	_, err = p.Direction(id)
	if err != peers.ErrPeerUnknown {
		t.Errorf("Unexpected error: expected %v, received %v", peers.ErrPeerUnknown, err)
	}

	_, err = p.ChainState(id)
	if err != peers.ErrPeerUnknown {
		t.Errorf("Unexpected error: expected %v, received %v", peers.ErrPeerUnknown, err)
	}

	_, err = p.ConnectionState(id)
	if err != peers.ErrPeerUnknown {
		t.Errorf("Unexpected error: expected %v, received %v", peers.ErrPeerUnknown, err)
	}

	_, err = p.ChainStateLastUpdated(id)
	if err != peers.ErrPeerUnknown {
		t.Errorf("Unexpected error: expected %v, received %v", peers.ErrPeerUnknown, err)
	}

	_, err = p.BadResponses(id)
	if err != peers.ErrPeerUnknown {
		t.Errorf("Unexpected error: expected %v, received %v", peers.ErrPeerUnknown, err)
	}
}

func TestPeerImplicitAdd(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	id, err := peer.IDB58Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	if err != nil {
		t.Fatal(err)
	}

	connectionState := peers.PeerConnecting
	p.SetConnectionState(id, connectionState)

	resConnectionState, err := p.ConnectionState(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resConnectionState != connectionState {
		t.Errorf("Unexpected connection state: expected %v, received %v", connectionState, resConnectionState)
	}
}

func TestPeerChainState(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	id, err := peer.IDB58Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	if err != nil {
		t.Fatal(err)
	}
	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	if err != nil {
		t.Fatalf("Failed to create address: %v", err)
	}
	direction := network.DirInbound
	p.Add(id, address, direction)

	oldChainStartLastUpdated, err := p.ChainStateLastUpdated(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	finalizedEpoch := uint64(123)
	p.SetChainState(id, &pb.Status{FinalizedEpoch: finalizedEpoch})

	resChainState, err := p.ChainState(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resChainState.FinalizedEpoch != finalizedEpoch {
		t.Errorf("Unexpected finalized epoch: expected %v, received %v", finalizedEpoch, resChainState.FinalizedEpoch)
	}

	newChainStartLastUpdated, err := p.ChainStateLastUpdated(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !newChainStartLastUpdated.After(oldChainStartLastUpdated) {
		t.Errorf("Last updated did not increase: old %v new %v", oldChainStartLastUpdated, newChainStartLastUpdated)
	}
}

func TestPeerBadResponses(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	id, err := peer.IDB58Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	if err != nil {
		t.Fatal(err)
	}
	{
		bytes, _ := id.MarshalBinary()
		fmt.Printf("%x\n", bytes)
	}

	if p.IsBad(id) {
		t.Error("Peer marked as bad when should be good")
	}

	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	if err != nil {
		t.Fatalf("Failed to create address: %v", err)
	}
	direction := network.DirInbound
	p.Add(id, address, direction)

	resBadResponses, err := p.BadResponses(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resBadResponses != 0 {
		t.Errorf("Unexpected bad responses: expected 0, received %v", resBadResponses)
	}
	if p.IsBad(id) {
		t.Error("Peer marked as bad when should be good")
	}

	p.IncrementBadResponses(id)
	resBadResponses, err = p.BadResponses(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resBadResponses != 1 {
		t.Errorf("Unexpected bad responses: expected 1, received %v", resBadResponses)
	}
	if p.IsBad(id) {
		t.Error("Peer marked as bad when should be good")
	}

	p.IncrementBadResponses(id)
	resBadResponses, err = p.BadResponses(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resBadResponses != 2 {
		t.Errorf("Unexpected bad responses: expected 2, received %v", resBadResponses)
	}
	if !p.IsBad(id) {
		t.Error("Peer not marked as bad when it should be")
	}

	p.IncrementBadResponses(id)
	resBadResponses, err = p.BadResponses(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resBadResponses != 3 {
		t.Errorf("Unexpected bad responses: expected 3, received %v", resBadResponses)
	}
	if !p.IsBad(id) {
		t.Error("Peer not marked as bad when it should be")
	}
}

func TestPeerConnectionStatuses(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	// Add some peers with different states
	numPeersDisconnected := 11
	for i := 0; i < numPeersDisconnected; i++ {
		addPeer(t, p, peers.PeerDisconnected)
	}
	numPeersConnecting := 7
	for i := 0; i < numPeersConnecting; i++ {
		addPeer(t, p, peers.PeerConnecting)
	}
	numPeersConnected := 43
	for i := 0; i < numPeersConnected; i++ {
		addPeer(t, p, peers.PeerConnected)
	}
	numPeersDisconnecting := 4
	for i := 0; i < numPeersDisconnecting; i++ {
		addPeer(t, p, peers.PeerDisconnecting)
	}

	// Now confirm the states
	if len(p.Disconnected()) != numPeersDisconnected {
		t.Errorf("Unexpected number of disconnected peers: expected %v, received %v", numPeersDisconnected, len(p.Disconnected()))
	}
	if len(p.Connecting()) != numPeersConnecting {
		t.Errorf("Unexpected number of connecting peers: expected %v, received %v", numPeersConnecting, len(p.Connecting()))
	}
	if len(p.Connected()) != numPeersConnected {
		t.Errorf("Unexpected number of connected peers: expected %v, received %v", numPeersConnected, len(p.Connected()))
	}
	if len(p.Disconnecting()) != numPeersDisconnecting {
		t.Errorf("Unexpected number of disconnecting peers: expected %v, received %v", numPeersDisconnecting, len(p.Disconnecting()))
	}
	numPeersActive := numPeersConnecting + numPeersConnected
	if len(p.Active()) != numPeersActive {
		t.Errorf("Unexpected number of active peers: expected %v, received %v", numPeersActive, len(p.Active()))
	}
	numPeersInactive := numPeersDisconnecting + numPeersDisconnected
	if len(p.Inactive()) != numPeersInactive {
		t.Errorf("Unexpected number of inactive peers: expected %v, received %v", numPeersInactive, len(p.Inactive()))
	}
	numPeersAll := numPeersActive + numPeersInactive
	if len(p.All()) != numPeersAll {
		t.Errorf("Unexpected number of peers: expected %v, received %v", numPeersAll, len(p.All()))
	}
}

func TestDecay(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	// Peer 1 has 0 bad responses.
	pid1 := addPeer(t, p, peers.PeerConnected)
	// Peer 2 has 1 bad response.
	pid2 := addPeer(t, p, peers.PeerConnected)
	p.IncrementBadResponses(pid2)
	// Peer 3 has 2 bad response.
	pid3 := addPeer(t, p, peers.PeerConnected)
	p.IncrementBadResponses(pid3)
	p.IncrementBadResponses(pid3)

	// Decay the values
	p.Decay()

	// Ensure the new values are as expected
	badResponses1, _ := p.BadResponses(pid1)
	if badResponses1 != 0 {
		t.Errorf("Unexpected bad responses for peer 0: expected 0, received %v", badResponses1)
	}
	badResponses2, _ := p.BadResponses(pid2)
	if badResponses2 != 0 {
		t.Errorf("Unexpected bad responses for peer 0: expected 0, received %v", badResponses2)
	}
	badResponses3, _ := p.BadResponses(pid3)
	if badResponses3 != 1 {
		t.Errorf("Unexpected bad responses for peer 0: expected 0, received %v", badResponses3)
	}
}

// addPeer is a helper to add a peer with a given connection state)
func addPeer(t *testing.T, p *peers.Status, state peers.PeerConnectionState) peer.ID {
	// Set up some peers with different states
	mhBytes := []byte{0x11, 0x04}
	idBytes := make([]byte, 4)
	rand.Read(idBytes)
	mhBytes = append(mhBytes, idBytes...)
	id, err := peer.IDFromBytes(mhBytes)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	p.Add(id, nil, network.DirUnknown)
	p.SetConnectionState(id, state)
	return id
}
