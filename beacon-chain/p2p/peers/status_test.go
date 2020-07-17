package peers_test

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStatus(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})
	require.NotNil(t, p, "p not created")
	assert.Equal(t, maxBadResponses, p.Scorer().BadResponsesThreshold(), "maxBadResponses incorrect value")
}

func TestPeerExplicitAdd(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	id, err := peer.Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	require.NoError(t, err, "Failed to create ID")
	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	require.NoError(t, err, "Failed to create address")
	direction := network.DirInbound
	p.Add(new(enr.Record), id, address, direction)

	resAddress, err := p.Address(id)
	require.NoError(t, err)
	assert.Equal(t, address, resAddress, "Unexpected address")

	resDirection, err := p.Direction(id)
	require.NoError(t, err)
	assert.Equal(t, direction, resDirection, "Unexpected direction")

	// Update with another explicit add
	address2, err := ma.NewMultiaddr("/ip4/52.23.23.253/tcp/30000/ipfs/QmfAgkmjiZNZhr2wFN9TwaRgHouMTBT6HELyzE5A3BT2wK/p2p-circuit")
	require.NoError(t, err)
	direction2 := network.DirOutbound
	p.Add(new(enr.Record), id, address2, direction2)

	resAddress2, err := p.Address(id)
	require.NoError(t, err)
	assert.Equal(t, address2, resAddress2, "Unexpected address")

	resDirection2, err := p.Direction(id)
	require.NoError(t, err)
	assert.Equal(t, direction2, resDirection2, "Unexpected direction")
}

func TestPeerNoENR(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	id, err := peer.Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	require.NoError(t, err, "Failed to create ID")
	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	require.NoError(t, err, "Failed to create address")
	direction := network.DirInbound
	p.Add(nil, id, address, direction)

	retrievedENR, err := p.ENR(id)
	require.NoError(t, err, "Could not retrieve chainstate")
	var nilENR *enr.Record
	assert.Equal(t, nilENR, retrievedENR, "Wanted a nil enr to be saved")
}

func TestPeerNoOverwriteENR(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	id, err := peer.Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	require.NoError(t, err, "Failed to create ID")
	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	require.NoError(t, err, "Failed to create address")
	direction := network.DirInbound
	record := new(enr.Record)
	record.Set(enr.WithEntry("test", []byte{'a'}))
	p.Add(record, id, address, direction)
	// try to overwrite
	p.Add(nil, id, address, direction)

	retrievedENR, err := p.ENR(id)
	require.NoError(t, err, "Could not retrieve chainstate")
	require.NotNil(t, retrievedENR, "Wanted a non-nil enr")
}

func TestErrUnknownPeer(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	id, err := peer.Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	require.NoError(t, err)

	_, err = p.Address(id)
	assert.ErrorContains(t, peers.ErrPeerUnknown.Error(), err)

	_, err = p.Direction(id)
	assert.ErrorContains(t, peers.ErrPeerUnknown.Error(), err)

	_, err = p.ChainState(id)
	assert.ErrorContains(t, peers.ErrPeerUnknown.Error(), err)

	_, err = p.ConnectionState(id)
	assert.ErrorContains(t, peers.ErrPeerUnknown.Error(), err)

	_, err = p.ChainStateLastUpdated(id)
	assert.ErrorContains(t, peers.ErrPeerUnknown.Error(), err)

	_, err = p.Scorer().BadResponses(id)
	assert.ErrorContains(t, peers.ErrPeerUnknown.Error(), err)
}

func TestPeerCommitteeIndices(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	id, err := peer.Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	require.NoError(t, err, "Failed to create ID")
	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	require.NoError(t, err, "Failed to create address")
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
	require.NoError(t, err, "Could not retrieve committee indices")
	assert.DeepEqual(t, wantedIndices, indices)
}

func TestPeerSubscribedToSubnet(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

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
	assert.Equal(t, 1, len(peers), "Unexpected num of peers")
	assert.Equal(t, expectedPeer, peers[0])

	peers = p.SubscribedToSubnet(8)
	assert.Equal(t, 1, len(peers), "Unexpected num of peers")
	assert.Equal(t, expectedPeer, peers[0])

	peers = p.SubscribedToSubnet(9)
	assert.Equal(t, 1, len(peers), "Unexpected num of peers")
	assert.Equal(t, expectedPeer, peers[0])
}

func TestPeerImplicitAdd(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	id, err := peer.Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	require.NoError(t, err)

	connectionState := peers.PeerConnecting
	p.SetConnectionState(id, connectionState)

	resConnectionState, err := p.ConnectionState(id)
	require.NoError(t, err)

	assert.Equal(t, connectionState, resConnectionState, "Unexpected connection state")
}

func TestPeerChainState(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	id, err := peer.Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	require.NoError(t, err)
	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	require.NoError(t, err, "Failed to create address")
	direction := network.DirInbound
	p.Add(new(enr.Record), id, address, direction)

	oldChainStartLastUpdated, err := p.ChainStateLastUpdated(id)
	require.NoError(t, err)

	finalizedEpoch := uint64(123)
	p.SetChainState(id, &pb.Status{FinalizedEpoch: finalizedEpoch})

	resChainState, err := p.ChainState(id)
	require.NoError(t, err)
	assert.Equal(t, finalizedEpoch, resChainState.FinalizedEpoch, "Unexpected finalized epoch")

	newChainStartLastUpdated, err := p.ChainStateLastUpdated(id)
	require.NoError(t, err)
	if !newChainStartLastUpdated.After(oldChainStartLastUpdated) {
		t.Errorf("Last updated did not increase: old %v new %v", oldChainStartLastUpdated, newChainStartLastUpdated)
	}
}

func TestPeerBadResponses(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	id, err := peer.Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	require.NoError(t, err)
	{
		_, err := id.MarshalBinary()
		require.NoError(t, err)
	}

	assert.Equal(t, false, p.IsBad(id), "Peer marked as bad when should be good")

	address, err := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	require.NoError(t, err, "Failed to create address")
	direction := network.DirInbound
	p.Add(new(enr.Record), id, address, direction)

	resBadResponses, err := p.Scorer().BadResponses(id)
	require.NoError(t, err)
	assert.Equal(t, 0, resBadResponses, "Unexpected bad responses")
	assert.Equal(t, false, p.IsBad(id), "Peer marked as bad when should be good")

	p.Scorer().IncrementBadResponses(id)
	resBadResponses, err = p.Scorer().BadResponses(id)
	require.NoError(t, err)
	assert.Equal(t, 1, resBadResponses, "Unexpected bad responses")
	assert.Equal(t, false, p.IsBad(id), "Peer marked as bad when should be good")

	p.Scorer().IncrementBadResponses(id)
	resBadResponses, err = p.Scorer().BadResponses(id)
	require.NoError(t, err)
	assert.Equal(t, 2, resBadResponses, "Unexpected bad responses")
	assert.Equal(t, true, p.IsBad(id), "Peer not marked as bad when it should be")

	p.Scorer().IncrementBadResponses(id)
	resBadResponses, err = p.Scorer().BadResponses(id)
	require.NoError(t, err)
	assert.Equal(t, 3, resBadResponses, "Unexpected bad responses")
	assert.Equal(t, true, p.IsBad(id), "Peer not marked as bad when it should be")
}

func TestAddMetaData(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

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
	require.NoError(t, err)
	assert.Equal(t, newMetaData.SeqNumber, md.SeqNumber, "Unexpected sequence number")
}

func TestPeerConnectionStatuses(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

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
	assert.Equal(t, numPeersDisconnected, len(p.Disconnected()), "Unexpected number of disconnected peers")
	assert.Equal(t, numPeersConnecting, len(p.Connecting()), "Unexpected number of connecting peers")
	assert.Equal(t, numPeersConnected, len(p.Connected()), "Unexpected number of connected peers")
	assert.Equal(t, numPeersDisconnecting, len(p.Disconnecting()), "Unexpected number of disconnecting peers")
	numPeersActive := numPeersConnecting + numPeersConnected
	assert.Equal(t, numPeersActive, len(p.Active()), "Unexpected number of active peers")
	numPeersInactive := numPeersDisconnecting + numPeersDisconnected
	assert.Equal(t, numPeersInactive, len(p.Inactive()), "Unexpected number of inactive peers")
	numPeersAll := numPeersActive + numPeersInactive
	assert.Equal(t, numPeersAll, len(p.All()), "Unexpected number of peers")
}

func TestPrune(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	for i := 0; i < p.MaxPeerLimit()+100; i++ {
		if i%7 == 0 {
			// Peer added as disconnected.
			_ = addPeer(t, p, peers.PeerDisconnected)
		}
		// Peer added to peer handler.
		_ = addPeer(t, p, peers.PeerConnected)
	}

	disPeers := p.Disconnected()
	firstPID := disPeers[0]
	secondPID := disPeers[1]
	thirdPID := disPeers[2]

	// Make first peer a bad peer
	p.Scorer().IncrementBadResponses(firstPID)
	p.Scorer().IncrementBadResponses(firstPID)

	// Add bad response for p2.
	p.Scorer().IncrementBadResponses(secondPID)

	// Prune peers
	p.Prune()

	// Bad peer is expected to still be kept in handler.
	badRes, err := p.Scorer().BadResponses(firstPID)
	assert.NoError(t, err, "error is supposed to be  nil")
	assert.Equal(t, 2, badRes, "Did not get expected amount")

	// Not so good peer is pruned away so that we can reduce the
	// total size of the handler.
	badRes, err = p.Scorer().BadResponses(secondPID)
	assert.NotNil(t, err, "error is supposed to be not nil")

	// Last peer has been removed.
	badRes, err = p.Scorer().BadResponses(thirdPID)
	assert.NotNil(t, err, "error is supposed to be not nil")
}

func TestTrimmedOrderedPeers(t *testing.T) {
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: 1,
		},
	})

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
	assert.Equal(t, expectedTarget, target, "Incorrect target epoch retrieved")
	assert.Equal(t, maxPeers, len(pids), "Incorrect number of peers retrieved")

	// Expect the returned list to be ordered by finalized epoch and trimmed to max peers.
	assert.Equal(t, pid3, pids[0], "Incorrect first peer")
	assert.Equal(t, pid2, pids[1], "Incorrect second peer")
	assert.Equal(t, pid1, pids[2], "Incorrect third peer")
}

func TestBestPeer(t *testing.T) {
	maxBadResponses := 2
	expectedFinEpoch := uint64(4)
	expectedRoot := [32]byte{'t', 'e', 's', 't'}
	junkRoot := [32]byte{'j', 'u', 'n', 'k'}
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

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
	assert.Equal(t, expectedFinEpoch, retEpoch, "Incorrect Finalized epoch retrieved")
}

func TestBestFinalized_returnsMaxValue(t *testing.T) {
	maxBadResponses := 2
	maxPeers := 10
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})

	for i := 0; i <= maxPeers+100; i++ {
		p.Add(new(enr.Record), peer.ID(i), nil, network.DirOutbound)
		p.SetConnectionState(peer.ID(i), peers.PeerConnected)
		p.SetChainState(peer.ID(i), &pb.Status{
			FinalizedEpoch: 10,
		})
	}

	_, pids := p.BestFinalized(maxPeers, 0)
	assert.Equal(t, maxPeers, len(pids), "Wrong number of peers returned")
}

func TestStatus_CurrentEpoch(t *testing.T) {
	maxBadResponses := 2
	p := peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: maxBadResponses,
		},
	})
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

	assert.Equal(t, uint64(5), p.HighestEpoch(), "Expected current epoch to be 5")
}

// addPeer is a helper to add a peer with a given connection state)
func addPeer(t *testing.T, p *peers.Status, state peers.PeerConnectionState) peer.ID {
	// Set up some peers with different states
	mhBytes := []byte{0x11, 0x04}
	idBytes := make([]byte, 4)
	_, err := rand.Read(idBytes)
	require.NoError(t, err)
	mhBytes = append(mhBytes, idBytes...)
	id, err := peer.IDFromBytes(mhBytes)
	require.NoError(t, err)
	p.Add(new(enr.Record), id, nil, network.DirUnknown)
	p.SetConnectionState(id, state)
	p.SetMetadata(id, &pb.MetaData{
		SeqNumber: 0,
		Attnets:   bitfield.NewBitvector64(),
	})
	return id
}
