package p2p

import (
	"fmt"
	"math/rand"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
)

var port int32 = 9000
var portRange int32 = 100

// buildOptions for the libp2p host.
// TODO: Expand on these options and provide the option configuration via flags.
// Currently, this is a random port and a (seemingly) consistent private key
// identity.
func buildOptions() []libp2p.Option {
	rand.Seed(int64(time.Now().Nanosecond()))
	priv, _, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 512)
	listen, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port+(rand.Int31n(portRange))))

	return []libp2p.Option{
		libp2p.ListenAddrs(listen),
		libp2p.Identity(priv),
	}
}
