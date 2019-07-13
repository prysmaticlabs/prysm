package hobbits

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"time"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/renaynay/go-hobbits/encoding"
)

func NewHobbitsNode(host string, port int, peers []string, db *db.BeaconDB) HobbitsNode {
	return HobbitsNode{
		NodeId:      strconv.Itoa(rand.Int()),
		Host:        host,
		Port:        port,
		StaticPeers: peers,
		PeerConns:   []net.Conn{},
		feeds:       map[reflect.Type]p2p.Feed{},
		DB:          db,
	}
}


func (h *HobbitsNode)  OpenConns() error {
	for _, p := range h.StaticPeers {
		go func(p string) {
			var conn net.Conn
			var err error

			for i := 0; i <= 10; i++ {
				conn, err = net.DialTimeout("tcp", p, 3*time.Second)
				if err == nil {
					break
				}

				fmt.Println(err)

				time.Sleep(5 * time.Second)
			}

			h.Lock()

			h.PeerConns = append(h.PeerConns, conn)

			h.Unlock()
		}(p)
	}

	return nil
}

func (h *HobbitsNode) Listen() error {
	log.Trace("hobbits node is listening")

	err := h.Server.Listen(func(conn net.Conn, message encoding.Message) {
		err := h.processHobbitsMessage(HobbitsMessage(message), conn)
		if err != nil {
			log.Error(err)
			_ = conn.Close()
		} else {
			log.Trace("a message has been received")
		}
	})

	if err != nil {
		return err
	}
	return nil
}

func (h *HobbitsNode) Broadcast(ctx context.Context, message proto.Message) {

	//
	//for _, peer := range h.PeerConns {
	//	err := h.Server.SendMessage(peer, encoding.Message(message))
	//	if err != nil {
	//		return errors.Wrap(err, "error broadcasting: ")
	//	}
	//
	//	peer.Close() // TODO: do I wanna be closing the conns?
	//}
}

// Send conforms to the p2p composite interface.
func (h *HobbitsNode) Send(ctx context.Context, msg proto.Message, peer peer.ID) error {
	return nil
}

// ReputationManager conforms to the p2p composite interface
func (h *HobbitsNode) Reputation(peer peer.ID, val int) {
}

// Subscriber conforms to the p2p composite interface
func (h *HobbitsNode) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return nil
}

func (h *HobbitsNode) Status() error {
	return nil
}

func (h *HobbitsNode) Stop() error {
	return nil
}
// Service conforms to the p2p composite interface
type Service shared.Service

func (h *HobbitsNode) Feed(msg proto.Message) p2p.Feed {
	//t := messageType(msg)
	//
	//h.mutex.Lock()
	//defer h.mutex.Unlock()
	//if s.feeds[t] == nil {
	//	s.feeds[t] = new(event.Feed)
	//}
	//
	//return s.feeds[t]
}
