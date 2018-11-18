package p2p

import (

	//	ps "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p/config"
	ma "github.com/multiformats/go-multiaddr"
)

// addRelayAddrs adds a relay connection string for each address in Addrs().
func addRelayAddrs(relay string, relayOnly bool) config.AddrsFactory {

	return func(addrs []ma.Multiaddr) []ma.Multiaddr {
		if relay == "" {
			return addrs
		}

		var relayAddrs []ma.Multiaddr

		for _, a := range addrs {
			if a.String() == "/p2p-circuit" {
				continue
			}
			relayAddr, err := ma.NewMultiaddr(relay + "/p2p-circuit" + a.String())
			if err != nil {
				panic(err) // TODO: handle
			}

			relayAddrs = append(relayAddrs, relayAddr)
		}

		if relayOnly {
			return relayAddrs
		}

		return append(addrs, relayAddrs...)
	}
}
