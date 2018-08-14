package p2p

import (
	"fmt"
	"math/rand"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/iputils"
)

var port int32 = 9000
var portRange int32 = 100

// buildOptions for the libp2p host.
// TODO: Expand on these options and provide the option configuration via flags.
// Currently, this is a random port and a (seemingly) consistent private key
// identity.
func buildOptions() []libp2p.Option {
	rand.Seed(int64(time.Now().Nanosecond()))
	priv, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 512)
	if err != nil {
		log.Errorf("Failed to generate crypto key pair: %v", err)
	}

	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Errorf("Could not get IPv4 address: %v", err)
	}

	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port+(rand.Int31n(portRange))))
	if err != nil {
		log.Errorf("Failed to p2p listen: %v", err)
	}

	return []libp2p.Option{
		libp2p.ListenAddrs(listen),
		libp2p.Identity(priv),
	}
}
