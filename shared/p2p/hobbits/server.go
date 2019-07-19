package hobbits

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/renaynay/go-hobbits/encoding"
	log "github.com/sirupsen/logrus"
	ttl "github.com/ReneKroon/ttlcache"
)

func NewHobbitsNode(host string, port int, peers []string, db *db.BeaconDB) HobbitsNode {
	cache := ttl.NewCache()
	cache.Set(string(make([]byte, 32)), true)

	return HobbitsNode{
		NodeId:       strconv.Itoa(rand.Int()),
		Host:         host,
		Port:         port,
		StaticPeers:  peers,
		PeerConns:    make(map[peer.ID]net.Conn),
		feeds:        make(map[reflect.Type]p2p.Feed),
		DB:           db,
		MessageStore: cache,
	}
}

func (h *HobbitsNode) OpenConns() error {
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

func (h *HobbitsNode) Broadcast(ctx context.Context, msg proto.Message) { // TODO this is all messaged up
	for _, peer := range h.PeerConns {
		err := h.Server.SendMessage(peer, encoding.Message())
		if err != nil {
			return errors.Wrap(err, "error broadcasting: ")
		}

		peer.Close() // TODO: do I wanna be closing the conns?
	}
}

// Send builds and sends a message to a Hobbits peer
// It conforms to the p2p composite interface
func (h *HobbitsNode) Send(ctx context.Context, msg proto.Message, peer peer.ID) error {
	var function func(msg proto.Message) (HobbitsMessage, error)

	switch msg.(type) { // investigate the MSG type
	case *pb.BatchedBeaconBlockResponse:
		function = h.blockBodiesResponse
	case *pb.AttestationResponse:
		function = h.attestationResponse
	default:
		return fmt.Errorf("unknown message type %s, could not handle response", reflect.TypeOf(msg).String())
	}

	hobMsg, err := function(msg)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error building %s response", reflect.TypeOf(msg).String()))
	}

	err = h.Server.SendMessage(h.PeerConns[peer], encoding.Message(hobMsg))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error sending %s response", reflect.TypeOf(msg).String()))
	}

	return nil
}

// ReputationManager conforms to the p2p composite interface
// Outside the scope of the Hobbits protocol
func (h *HobbitsNode) Reputation(peer peer.ID, val int) {
}

// Subscriber conforms to the p2p composite interface
func (h *HobbitsNode) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return h.Feed(msg).Subscribe(channel)
}

func (h *HobbitsNode) Status() error {
	return nil
}

func (h *HobbitsNode) Stop() error {
	return nil
}

// Service conforms to the p2p composite interface
type Service shared.Service
