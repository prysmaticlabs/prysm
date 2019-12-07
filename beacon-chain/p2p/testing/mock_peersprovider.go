package testing

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
)

// MockPeersProvider implements PeersProvider for testing.
type MockPeersProvider struct {
}

// Peers records a broadcast occurred.
func (m *MockPeersProvider) Peers() []*peers.Info {
	res := make([]*peers.Info, 2)
	id0, _ := peer.IDB58Decode("16Uiu2HAkyWZ4Ni1TpvDS8dPxsozmHY85KaiFjodQuV6Tz5tkHVeR")
	ma0, _ := ma.NewMultiaddr("/ip4/213.202.254.180/tcp/13000")
	res[0] = &peers.Info{
		AddrInfo: &peer.AddrInfo{
			ID:    id0,
			Addrs: []ma.Multiaddr{ma0},
		},
		Direction: network.DirInbound,
	}
	id1, _ := peer.IDB58Decode("16Uiu2HAm4HgJ9N1o222xK61o7LSgToYWoAy1wNTJRkh9gLZapVAy")
	ma1, _ := ma.NewMultiaddr("/ip4/52.23.23.253/tcp/30000/ipfs/QmfAgkmjiZNZhr2wFN9TwaRgHouMTBT6HELyzE5A3BT2wK/p2p-circuit")
	res[1] = &peers.Info{
		AddrInfo: &peer.AddrInfo{
			ID:    id1,
			Addrs: []ma.Multiaddr{ma1},
		},
		Direction: network.DirOutbound,
	}
	return res
}
