package hobbits

import (
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/renaynay/go-hobbits/encoding"
	"github.com/renaynay/go-hobbits/tcp"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

func NewHobbitsNode(host string, port int, peers []string) HobbitsNode {
	return HobbitsNode{
		host:        host,
		port:        port,
		staticPeers: peers,
		peerConns:   []net.Conn{},
		feeds:       map[reflect.Type]p2p.Feed{},
	}
}

func (h *HobbitsNode) OpenConns() error {
	for _, p := range h.staticPeers {
		p := p

		go func() {
			var conn net.Conn
			var err error

			for i := 0; i <= 10; i++ {
				conn, err = net.DialTimeout("tcp", p, 3*time.Second)
				if err == nil {
					break
				}

				fmt.Println(err)

				time.Sleep(5*time.Second)
			}

			h.Lock()

			h.peerConns = append(h.peerConns, conn)

			h.Unlock()
		}()
	}

	return nil
}

func (h *HobbitsNode) Listen() error {
	h.server = tcp.NewServer(h.host, h.port)

	return h.server.Listen(func(conn net.Conn, message encoding.Message) {
		h.processHobbitsMessage(HobbitsMessage(message), conn)
	})
}

func (h *HobbitsNode) Broadcast(message HobbitsMessage) error {
	for _, peer := range h.peerConns {
		err := h.server.SendMessage(peer, encoding.Message(message))
		if err != nil {
			return errors.Wrap(err, "error broadcasting: ")
		}

		peer.Close() // TODO: do I wanna be closing the conns?
	}

	return nil
}
