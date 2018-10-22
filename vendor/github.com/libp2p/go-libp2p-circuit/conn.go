package relay

import (
	"fmt"
	"net"

	inet "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

type Conn struct {
	inet.Stream
	remote pstore.PeerInfo
}

type NetAddr struct {
	Relay  string
	Remote string
}

func (n *NetAddr) Network() string {
	return "libp2p-circuit-relay"
}

func (n *NetAddr) String() string {
	return fmt.Sprintf("relay[%s-%s]", n.Remote, n.Relay)
}

func (c *Conn) RemoteAddr() net.Addr {
	return &NetAddr{
		Relay:  c.Conn().RemotePeer().Pretty(),
		Remote: c.remote.ID.Pretty(),
	}
}

// TODO: is it okay to cast c.Conn().RemotePeer() into a multiaddr? might be "user input"
func (c *Conn) RemoteMultiaddr() ma.Multiaddr {
	proto := ma.ProtocolWithCode(ma.P_P2P).Name
	peerid := c.Conn().RemotePeer().Pretty()
	p2paddr := ma.StringCast(fmt.Sprintf("/%s/%s", proto, peerid))

	circaddr := ma.Cast(ma.CodeToVarint(P_CIRCUIT))
	return p2paddr.Encapsulate(circaddr)
}

func (c *Conn) LocalMultiaddr() ma.Multiaddr {
	return c.Conn().LocalMultiaddr()
}

func (c *Conn) LocalAddr() net.Addr {
	na, err := manet.ToNetAddr(c.Conn().LocalMultiaddr())
	if err != nil {
		log.Error("failed to convert local multiaddr to net addr:", err)
		return nil
	}
	return na
}
