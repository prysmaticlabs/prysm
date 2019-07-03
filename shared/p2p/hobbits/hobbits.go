package hobbits

import (
	"net"
	"reflect"
	"sync"

	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/renaynay/go-hobbits/encoding"
	"github.com/renaynay/go-hobbits/tcp"
	"github.com/renaynay/prysm/beacon-chain/db"
)

type HobbitsNode struct {
	sync.Mutex
	host        string
	port        int
	staticPeers []string
	peerConns   []net.Conn
	feeds       map[reflect.Type]p2p.Feed
	server      *tcp.Server
	db          *db.BeaconDB
}

type HobbitsMessage encoding.Message

type RPCMethod int

const (
	HELLO RPCMethod = iota
	GOODBYE
	GET_STATUS
	GET_BLOCK_HEADERS = iota + 62
	BLOCK_HEADERS
	GET_BLOCK_BODIES
	BLOCK_BODIES
	GET_ATTESTATION  //TODO: define in the spec what hex this corresponds to
	ATTESTATION      // TODO: define in the spec what this means
)

var topicMapping map[reflect.Type]string // TODO: initialize with a const?

type GossipHeader struct {
	topic string `bson:"topic"`
}

type RPC struct {
	// TODO: make an RPC Body to catch the method_id... looks like the header and body are smashed
	// TODO: in the spec
}

type Hello struct {
	networkID            uint8    `bson:"network_id"`
	chainID              uint8    `bson:"chain_id"`
	latestFinalizedRoot  [32]byte `bson:"latest_finalized_root"`
	latestFinalizedEpoch uint64   `bson:"latest_finalized_epoch"`
	bestRoot             [32]byte `bson:"best_root"`
	bestSlot             uint64   `bson:"best_slot"`
}
