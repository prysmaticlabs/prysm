package p2p

import (
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
)

var port = 9001

func buildOptions() []libp2p.Option {
	priv, _, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 512)
	listen, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))

	return []libp2p.Option{
		libp2p.ListenAddrs(listen),
		libp2p.Identity(priv),
	}
}
