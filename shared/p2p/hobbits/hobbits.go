package hobbits

import (
	"net"
	"reflect"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/renaynay/go-hobbits/encoding"
	"github.com/renaynay/go-hobbits/tcp"
	log "github.com/sirupsen/logrus"
)

type HobbitsNode struct {
	sync.Mutex
	NodeId      string
	Host        string
	Port        int
	StaticPeers []string
	PeerConns   map[peer.ID]net.Conn
	feeds       map[reflect.Type]p2p.Feed
	Server      *tcp.Server
	DB          *db.BeaconDB
}

type HobbitsMessage encoding.Message

var topicMapping map[reflect.Type]string // TODO: initialize with a const? How TF do I use this??

type RPCMethod uint8

const (
	HELLO RPCMethod = iota
	GOODBYE
	GET_STATUS
	GET_BLOCK_HEADERS = iota + 8
	BLOCK_HEADERS
	GET_BLOCK_BODIES
	BLOCK_BODIES
	GET_ATTESTATION  //TODO: define in the spec what hex this corresponds to
	ATTESTATION      // TODO: define in the spec what this means
)

// Hobbits toggles a HobbitsNode and requires a host, port and list of peers to which it tries to connect.
func Hobbits(host string, port int, peers []string, db *db.BeaconDB) *HobbitsNode {
	node := NewHobbitsNode(host, port, peers, db)
	node.Server = tcp.NewServer(node.Host, node.Port)

	log.Trace("node has been constructed")

	return &node
}

func (h *HobbitsNode) Start() {
	go h.Listen()

	go h.OpenConns()
}
