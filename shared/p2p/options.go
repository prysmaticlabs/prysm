package p2p

import (
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/iputils"
)

// buildOptions for the libp2p host.
// TODO(287): Expand on these options and provide the option configuration via flags.
// Currently, this is a random port and a (seemingly) consistent private key
// identity.
func buildOptions(port int) []libp2p.Option {
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Errorf("Could not get IPv4 address: %v", err)
	}

	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port))
	if err != nil {
		log.Errorf("Failed to p2p listen: %v", err)
	}

	return []libp2p.Option{
		libp2p.ListenAddrs(listen),
		libp2p.EnableRelay(), // Allows dialing to peers via relay.
	}
}
