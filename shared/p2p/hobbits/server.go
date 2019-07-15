package hobbits

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/renaynay/go-hobbits/encoding"
	log "github.com/sirupsen/logrus"
)

func NewHobbitsNode(host string, port int, peers []string, db *db.BeaconDB) HobbitsNode {
	return HobbitsNode{
		NodeId:      strconv.Itoa(rand.Int()),
		Host:        host,
		Port:        port,
		StaticPeers: peers,
		PeerConns:   make(map[peer.ID]net.Conn),
		feeds:       make(map[reflect.Type]p2p.Feed),
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

			h.PeerConns[peer.ID(p)] = conn

			h.Unlock()
		}(p)
	}

	return nil
}

func (h *HobbitsNode) Listen() error {
	log.Trace("hobbits node is listening")

	return h.Server.Listen(func(conn net.Conn, message encoding.Message) {
		id := peer.ID(conn.RemoteAddr().String())
		_, ok := h.PeerConns[id]
		if !ok {
			h.PeerConns[id] = conn
		}

		err := h.processHobbitsMessage(id, HobbitsMessage(message))
		if err != nil {
			log.Error(err)
			_ = conn.Close()
			delete(h.PeerConns, id)
			return
		}

		log.Trace("a message has been received")
	})
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
	conn := h.PeerConns[peer]  // get the conn for the peer

	switch msg.(type) { // investigate the MSG type
	case *pb.BatchedBeaconBlockResponse:
		hobMsg, err := h.blockBodiesResponse(msg)
		if err != nil {
			return errors.Wrap(err, "error building BLOCK_BODIES response")
		}

		err = h.Server.SendMessage(h.PeerConns[peer], encoding.Message(hobMsg))
		if err != nil {
			return errors.Wrap(err, "error sending BLOCK_BODIES response")
		}
	case *pb.AttestationResponse:
		hobMsg, err := h.attestationResponse(msg)
		if err != nil {
			return errors.Wrap(err, "error building ATTESTATION response")
		}

		err = h.Server.SendMessage(h.PeerConns[peer], encoding.Message(hobMsg))
		if err != nil {
			return errors.Wrap(err, "error sending ATTESTATION response")
		}
	}

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


