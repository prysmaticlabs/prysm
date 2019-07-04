package hobbits

import (
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/renaynay/go-hobbits/encoding"
	"github.com/renaynay/go-hobbits/tcp"
)

func NewHobbitsNode(host string, port int, peers []string) HobbitsNode {
	return HobbitsNode{
		NodeId:      strconv.Itoa(rand.Int()),
		Host:        host,
		Port:        port,
		StaticPeers: peers,
		PeerConns:   []net.Conn{},
		feeds:       map[reflect.Type]p2p.Feed{},
		DB: 	//TODO; how tf to initialize the db?,
	}
}

func (h *HobbitsNode) OpenConns() error {
	for _, p := range h.StaticPeers {
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

			h.PeerConns = append(h.PeerConns, conn)

			h.Unlock()
		}()
	}

	return nil
}

func (h *HobbitsNode) Listen() error {
	h.Server = tcp.NewServer(h.Host, h.Port)

	return h.Server.Listen(func(conn net.Conn, message encoding.Message) {
		h.processHobbitsMessage(HobbitsMessage(message), conn)
	})
}

func (h *HobbitsNode) Broadcast(message HobbitsMessage) error {
	for _, peer := range h.PeerConns {
		err := h.Server.SendMessage(peer, encoding.Message(message))
		if err != nil {
			return errors.Wrap(err, "error broadcasting: ")
		}

		peer.Close() // TODO: do I wanna be closing the conns?
	}

	return nil
}
