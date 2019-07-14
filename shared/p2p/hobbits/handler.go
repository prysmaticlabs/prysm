package hobbits

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/renaynay/go-hobbits/encoding"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

type RPCHeader struct {
	MethodID uint8 `bson:"method_id"`
}

type Hello struct {
	NodeID               string   `bson:"node_id"`
	LatestFinalizedRoot  [32]byte `bson:"latest_finalized_root"`
	LatestFinalizedEpoch uint64   `bson:"latest_finalized_epoch"`
	BestRoot             [32]byte `bson:"best_root"`
	BestSlot             uint64   `bson:"best_slot"`
}

type GossipHeader struct {
	topic string `bson:"topic"`
}

type Status struct {
	UserAgent []byte `bson:"user_agent"`
	Timestamp uint64 `bson:"timestamp"`
}

type BlockRequest struct {
	StartRoot [32]byte `bson:"start_root"`
	StartSlot uint64   `bson:"start_slot"`
	Max       uint64   `bson:"max"`
	Skip      uint64   `bson:"skip"`
	Direction uint8    `bson:"direction"`
}

func (h *HobbitsNode) status(message HobbitsMessage, conn net.Conn) error {
	// does something with the status of the other node
	responseBody := Status{
		UserAgent: []byte(fmt.Sprintf("prysm node %d", h.NodeId)),
		Timestamp: uint64(time.Now().Unix()),
	}

	body, err := bson.Marshal(responseBody)
	if err != nil {
		return errors.Wrap(err, "error marshaling response body: ")
	}

	responseMessage := HobbitsMessage{
		Version:  message.Version,
		Protocol: message.Protocol,
		Header:   message.Header,
		Body:     body,
	}

	err = h.Server.SendMessage(conn, encoding.Message(responseMessage))
	if err != nil {
		return errors.Wrap(err, "error sending GET_STATUS: ")
	}

	return nil
}

func (h *HobbitsNode) sendHello(message HobbitsMessage, conn net.Conn) error {
	response := h.rpcHello()

	responseBody, err := bson.Marshal(response)

	responseMessage := HobbitsMessage{
		Version:  message.Version,
		Protocol: message.Protocol,
		Header:   message.Header,
		Body:     responseBody,
	}
	log.Trace(responseMessage)

	err = h.Server.SendMessage(conn, encoding.Message(responseMessage))
	if err != nil {
		log.Trace("error sending a HELLO back") // TODO delete
		return errors.Wrap(err, "error sending HELLO: ")
	}

	log.Trace("sending HELLO...")
	return nil
}

func (h *HobbitsNode) rpcHello() Hello {
	var response Hello

	response.NodeID = h.NodeId
	response.BestRoot = h.DB.HeadStateRoot()

	headState, err := h.DB.HeadState(context.Background())
	if err != nil {
		log.Printf("error getting HeadState data from db: %s", err.Error())
	} else {
		response.BestSlot = headState.Slot // best slot
	}

	finalizedState, err := h.DB.FinalizedState()
	if err != nil {
		log.Printf("error getting FinalizedState data from db: %s", err.Error())
	} else {
		response.LatestFinalizedEpoch = finalizedState.Slot / 64 // finalized epoch

		hashedFinalizedState, err := hashutil.HashProto(finalizedState) // finalized root
		if err != nil {
			log.Printf("error hashing FinalizedState: %s", err.Error())
		} else {
			response.LatestFinalizedRoot = hashedFinalizedState
		}
	}

	return response
}

func (h *HobbitsNode) removePeer(peer net.Conn) error {
	index := 0

	for i, p := range h.PeerConns {
		if reflect.DeepEqual(peer, p) {
			index = i
		}
	}
	if index == 0 {
		return errors.New("error removing peer from node's open connections")
	}
	h.PeerConns = append(h.PeerConns[:index], h.PeerConns[index+1:]...)
	err := peer.Close()
	if err != nil {
		return errors.Wrap(err, "error closing connection on peer")
	}

	index = 0

	for i, p := range h.StaticPeers {
		if reflect.DeepEqual(peer.RemoteAddr().String(), p) {
			index = i
		}
	}
	if index == 0 {
		return errors.New("error removing peer from node's static peers")
	}
	h.StaticPeers = append(h.StaticPeers[:index], h.StaticPeers[index+1:]...)

	return nil
}

func (h *HobbitsNode) blockHeaders(message HobbitsMessage, conn net.Conn) error {
	// var request BlockRequest // TODO: might not need BlockRequest struct, just unmarshal into protobuf
	//err := bson.Unmarshal(message.Body, request)
	//if err != nil {
	//	return errors.Wrap(err, "could not unmarshal block header RPC request: ")
	//}

	return nil
}

func (h *HobbitsNode) blockBodies(message HobbitsMessage, conn net.Conn) error {
	//var requestBody BlockRequest
	//
	//err := bson.Unmarshal(message.Body, requestBody)
	//if err != nil {
	//	return errors.Wrap(err, "could not unmarshal block body RPC request: ")
	//}
	//
	//var request p2p.Message
	//request.Data = requestBody


	return nil
}

func (h *HobbitsNode) attestation(message HobbitsMessage, conn net.Conn) error {

	return nil
}
