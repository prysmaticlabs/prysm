package p2p

import (
	"context"
	"testing"

	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	tu "github.com/libp2p/go-testutil"
	ma "github.com/multiformats/go-multiaddr"
)

func hostWithConnMgr(t *testing.T) host.Host {
	h, err := libp2p.New(context.Background(), optionConnectionManager(5))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// Test libp2p connection for connection manager
type tconn struct {
	inet.Conn

	pid peer.ID
}

func (t *tconn) RemotePeer() peer.ID {
	return t.pid
}

func (_ *tconn) RemoteMultiaddr() ma.Multiaddr {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/1234")
	if err != nil {
		panic("cannot create multiaddr")
	}
	return addr
}

func TestReputation(t *testing.T) {
	h := hostWithConnMgr(t)

	s := &Server{
		host: h,
	}

	pid := tu.RandPeerIDFatal(t)

	h.ConnManager().Notifee().Connected(h.Network(), &tconn{pid: pid})

	s.Reputation(pid, 5)
	if h.ConnManager().GetTagInfo(pid).Value != 5 {
		t.Fatal("Expected value 5")
	}

	s.Reputation(pid, -10)
	if h.ConnManager().GetTagInfo(pid).Value != -5 {
		t.Fatal("Expected value -5")
	}

	s.Reputation(pid, -10)
	if h.ConnManager().GetTagInfo(pid).Value != -15 {
		t.Fatal("Expected value -15")
	}

	s.Reputation(pid, 100)
	if h.ConnManager().GetTagInfo(pid).Value != 85 {
		t.Fatal("Expected value 85")
	}

}
