package peers_test

import (
	"crypto/rand"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/go-bitfield"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	p.Add(new(enr.Record), id, address, direction)

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
	p.Add(new(enr.Record), id, address2, direction2)

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

func TestPeerNoENR(t *testing.T) {
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
	p.Add(nil, id, address, direction)

	retrievedENR, err := p.ENR(id)
	if err != nil {
		t.Fatalf("Could not retrieve chainstate: %v", err)
	}
	if retrievedENR != nil {
		t.Error("Wanted a nil enr to be saved")
	}
}

func TestPeerNoOverwriteENR(t *testing.T) {
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
	record := new(enr.Record)
	record.Set(enr.WithEntry("test", []byte{'a'}))
	p.Add(record, id, address, direction)
	// try to overwrite
	p.Add(nil, id, address, direction)

	retrievedENR, err := p.ENR(id)
	if err != nil {
		t.Fatalf("Could not retrieve chainstate: %v", err)
	}
	if retrievedENR == nil {
		t.Error("Wanted a non-nil enr")
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

func TestPeerCommitteeIndices(t *testing.T) {
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
	record := new(enr.Record)
	record.Set(enr.WithEntry("test", []byte{'a'}))
	p.Add(record, id, address, direction)
	bitV := bitfield.NewBitvector64()
	for i := 0; i < 64; i++ {
		if i == 2 || i == 8 || i == 9 {
			bitV.SetBitAt(uint64(i), true)
		}
	}
	p.SetMetadata(id, &pb.MetaData{
		SeqNumber: 2,
		Attnets:   bitV,
	})

	wantedIndices := []uint64{2, 8, 9}

	indices, err := p.CommitteeIndices(id)
	if err != nil {
		t.Fatalf("Could not retrieve committee indices: %v", err)
	}

	if !reflect.DeepEqual(indices, wantedIndices) {
		t.Errorf("Wanted indices of %v but got %v", wantedIndices, indices)
	}
}

func TestPeerSubscribedToSubnet(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	// Add some peers with different states
	numPeers := 2
	for i := 0; i < numPeers; i++ {
		addPeer(t, p, peers.PeerConnected)
	}
	expectedPeer := p.All()[1]
	bitV := bitfield.NewBitvector64()
	for i := 0; i < 64; i++ {
		if i == 2 || i == 8 || i == 9 {
			bitV.SetBitAt(uint64(i), true)
		}
	}
	p.SetMetadata(expectedPeer, &pb.MetaData{
		SeqNumber: 2,
		Attnets:   bitV,
	})
	numPeers = 3
	for i := 0; i < numPeers; i++ {
		addPeer(t, p, peers.PeerDisconnected)
	}
	peers := p.SubscribedToSubnet(2)
	if len(peers) != 1 {
		t.Errorf("Expected num of peers to be %d but got %d", 1, len(peers))
	}
	if peers[0] != expectedPeer {
		t.Errorf("Expected peer of %s but got %s", expectedPeer, peers[0])
	}

	peers = p.SubscribedToSubnet(8)
	if len(peers) != 1 {
		t.Errorf("Expected num of peers to be %d but got %d", 1, len(peers))
	}
	if peers[0] != expectedPeer {
		t.Errorf("Expected peer of %s but got %s", expectedPeer, peers[0])
	}

	peers = p.SubscribedToSubnet(9)
	if len(peers) != 1 {
		t.Errorf("Expected num of peers to be %d but got %d", 1, len(peers))
	}
	if peers[0] != expectedPeer {
		t.Errorf("Expected peer of %s but got %s", expectedPeer, peers[0])
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
	p.Add(new(enr.Record), id, address, direction)

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
		bytes, err := id.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
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
	p.Add(new(enr.Record), id, address, direction)

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

func TestAddMetaData(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)

	// Add some peers with different states
	numPeers := 5
	for i := 0; i < numPeers; i++ {
		addPeer(t, p, peers.PeerConnected)
	}
	newPeer := p.All()[2]

	newMetaData := &pb.MetaData{
		SeqNumber: 8,
		Attnets:   bitfield.NewBitvector64(),
	}
	p.SetMetadata(newPeer, newMetaData)

	md, err := p.Metadata(newPeer)
	if err != nil {
		t.Fatal(err)
	}
	if md.SeqNumber != newMetaData.SeqNumber {
		t.Errorf("Wanted sequence number of %d but got %d", newMetaData.SeqNumber, md.SeqNumber)
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
	badResponses1, err := p.BadResponses(pid1)
	if err != nil {
		t.Fatal(err)
	}
	if badResponses1 != 0 {
		t.Errorf("Unexpected bad responses for peer 0: expected 0, received %v", badResponses1)
	}
	badResponses2, err := p.BadResponses(pid2)
	if err != nil {
		t.Fatal(err)
	}
	if badResponses2 != 0 {
		t.Errorf("Unexpected bad responses for peer 0: expected 0, received %v", badResponses2)
	}
	badResponses3, err := p.BadResponses(pid3)
	if err != nil {
		t.Fatal(err)
	}
	if badResponses3 != 1 {
		t.Errorf("Unexpected bad responses for peer 0: expected 0, received %v", badResponses3)
	}
}

func TestTrimmedOrderedPeers(t *testing.T) {
	p := peers.NewStatus(1)

	expectedTarget := uint64(2)
	maxPeers := 3
	mockroot2 := [32]byte{}
	mockroot3 := [32]byte{}
	mockroot4 := [32]byte{}
	mockroot5 := [32]byte{}
	copy(mockroot2[:], "two")
	copy(mockroot3[:], "three")
	copy(mockroot4[:], "four")
	copy(mockroot5[:], "five")
	// Peer 1
	pid1 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid1, &pb.Status{
		FinalizedEpoch: 3,
		FinalizedRoot:  mockroot3[:],
	})
	// Peer 2
	pid2 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid2, &pb.Status{
		FinalizedEpoch: 4,
		FinalizedRoot:  mockroot4[:],
	})
	// Peer 3
	pid3 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid3, &pb.Status{
		FinalizedEpoch: 5,
		FinalizedRoot:  mockroot5[:],
	})
	// Peer 4
	pid4 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid4, &pb.Status{
		FinalizedEpoch: 2,
		FinalizedRoot:  mockroot2[:],
	})
	// Peer 5
	pid5 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid5, &pb.Status{
		FinalizedEpoch: 2,
		FinalizedRoot:  mockroot2[:],
	})

	target, pids := p.BestFinalized(maxPeers, 0)
	if target != expectedTarget {
		t.Errorf("Incorrect target epoch retrieved; wanted %v but got %v", expectedTarget, target)
	}
	if len(pids) != maxPeers {
		t.Errorf("Incorrect number of peers retrieved; wanted %v but got %v", maxPeers, len(pids))
	}

	// Expect the returned list to be ordered by finalized epoch and trimmed to max peers.
	if pids[0] != pid3 {
		t.Errorf("Incorrect first peer; wanted %v but got %v", pid3, pids[0])
	}
	if pids[1] != pid2 {
		t.Errorf("Incorrect second peer; wanted %v but got %v", pid2, pids[1])
	}
	if pids[2] != pid1 {
		t.Errorf("Incorrect third peer; wanted %v but got %v", pid1, pids[2])
	}
}

func TestBestPeer(t *testing.T) {
	maxBadResponses := 2
	expectedFinEpoch := uint64(4)
	expectedRoot := [32]byte{'t', 'e', 's', 't'}
	junkRoot := [32]byte{'j', 'u', 'n', 'k'}
	p := peers.NewStatus(maxBadResponses)

	// Peer 1
	pid1 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid1, &pb.Status{
		FinalizedEpoch: expectedFinEpoch,
		FinalizedRoot:  expectedRoot[:],
	})
	// Peer 2
	pid2 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid2, &pb.Status{
		FinalizedEpoch: expectedFinEpoch,
		FinalizedRoot:  expectedRoot[:],
	})
	// Peer 3
	pid3 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid3, &pb.Status{
		FinalizedEpoch: 3,
		FinalizedRoot:  junkRoot[:],
	})
	// Peer 4
	pid4 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid4, &pb.Status{
		FinalizedEpoch: expectedFinEpoch,
		FinalizedRoot:  expectedRoot[:],
	})
	// Peer 5
	pid5 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid5, &pb.Status{
		FinalizedEpoch: expectedFinEpoch,
		FinalizedRoot:  expectedRoot[:],
	})
	// Peer 6
	pid6 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid6, &pb.Status{
		FinalizedEpoch: 3,
		FinalizedRoot:  junkRoot[:],
	})
	retEpoch, _ := p.BestFinalized(15, 0)
	if retEpoch != expectedFinEpoch {
		t.Errorf("Incorrect Finalized epoch retrieved; wanted %v but got %v", expectedFinEpoch, retEpoch)
	}
}

func TestBestFinalized_returnsMaxValue(t *testing.T) {
	maxBadResponses := 2
	maxPeers := 10
	p := peers.NewStatus(maxBadResponses)

	for i := 0; i <= maxPeers+100; i++ {
		p.Add(new(enr.Record), peer.ID(i), nil, network.DirOutbound)
		p.SetConnectionState(peer.ID(i), peers.PeerConnected)
		p.SetChainState(peer.ID(i), &pb.Status{
			FinalizedEpoch: 10,
		})
	}

	_, pids := p.BestFinalized(maxPeers, 0)
	if len(pids) != maxPeers {
		t.Fatalf("returned wrong number of peers, wanted %d, got %d", maxPeers, len(pids))
	}
}

func TestStatus_CurrentEpoch(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(maxBadResponses)
	// Peer 1
	pid1 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid1, &pb.Status{
		HeadSlot: params.BeaconConfig().SlotsPerEpoch * 4,
	})
	// Peer 2
	pid2 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid2, &pb.Status{
		HeadSlot: params.BeaconConfig().SlotsPerEpoch * 5,
	})
	// Peer 3
	pid3 := addPeer(t, p, peers.PeerConnected)
	p.SetChainState(pid3, &pb.Status{
		HeadSlot: params.BeaconConfig().SlotsPerEpoch * 4,
	})

	if p.HighestEpoch() != 5 {
		t.Fatalf("Expected current epoch to be 5, got %d", p.HighestEpoch())
	}
}

// addPeer is a helper to add a peer with a given connection state)
func addPeer(t *testing.T, p *peers.Status, state peers.PeerConnectionState) peer.ID {
	// Set up some peers with different states
	mhBytes := []byte{0x11, 0x04}
	idBytes := make([]byte, 4)
	if _, err := rand.Read(idBytes); err != nil {
		t.Fatal(err)
	}
	mhBytes = append(mhBytes, idBytes...)
	id, err := peer.IDFromBytes(mhBytes)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	p.Add(new(enr.Record), id, nil, network.DirUnknown)
	p.SetConnectionState(id, state)
	p.SetMetadata(id, &pb.MetaData{
		SeqNumber: 0,
		Attnets:   bitfield.NewBitvector64(),
	})
	return id
}
