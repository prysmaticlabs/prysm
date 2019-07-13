package hobbits

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	log "github.com/sirupsen/logrus"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/renaynay/go-hobbits/encoding"
	"gopkg.in/mgo.v2/bson"

	"github.com/pkg/errors"
)

func (h *HobbitsNode) processHobbitsMessage(message HobbitsMessage, conn net.Conn) error {
	switch message.Protocol {
	case encoding.RPC:
		log.Trace("beginning to process the RPC message...")

		err := h.processRPC(message, conn)
		if err != nil {
			log.Trace("there was an error processing an RPC hobbits msg ") // TODO DELETE
			return errors.Wrap(err, "error processing an RPC hobbits message")
		}
		return nil
	case encoding.GOSSIP:
		log.Trace("beginning to process the GOSSIP message...")

		err := h.processGossip(message)
		if err != nil {
			return errors.Wrap(err, "error processing a GOSSIP hobbits message")
		}

		return nil
	}

	return errors.New(fmt.Sprintf("protocol unsupported %v", message.Protocol))
}

func (h *HobbitsNode) processRPC(message HobbitsMessage, conn net.Conn) error { // TODO all of this needs to be put into funcs bc this function is getting disgusting.
	method, err := h.parseMethodID(message.Header)
	if err != nil {
		log.Trace("method id could not be parsed from message header")
		return errors.Wrap(err, "could not parse method_id: ")
	}

	switch method {
	case HELLO:
		log.Trace("HELLO received")

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
	case GOODBYE:
		err := h.removePeer(conn)
		if err != nil {
			return errors.Wrap(err, "error handling GOODBYE: ")
		}

		log.Trace("GOODBYE successful")
		return nil
	case GET_STATUS:

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
			Version: message.Version,
			Protocol: message.Protocol,
			Header: message.Header,
			Body: body,
		}

		err = h.Server.SendMessage(conn, encoding.Message(responseMessage))
		if err != nil {
			return errors.Wrap(err, "error sending GET_STATUS: ")
		}
	case GET_BLOCK_HEADERS:
		var request BlockRequest

		err := bson.Unmarshal(message.Body, request)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal block header RPC request: ")
		}

		var index int

	case BLOCK_HEADERS:
		// log this?
	case GET_BLOCK_BODIES: // TODO: this is so messed up
		var requestBody BlockRequest

		err := bson.Unmarshal(message.Body, requestBody)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal block body RPC request: ")
		}

		var request p2p.Message
		request.Data = requestBody


	case BLOCK_BODIES:
		// log this somehow?
	case GET_ATTESTATION:

	case ATTESTATION:
	}

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

func (h *HobbitsNode) processGossip(message HobbitsMessage) error {
	log.Trace("processing GOSSIP message")

	_, err := h.parseTopic(message)
	if err != nil {
		return errors.Wrap(err, "error parsing topic: ")
	}

	h.Broadcast(context.Background(), nil)

	return nil
}

func (h *HobbitsNode) parseMethodID(header []byte) (RPCMethod, error) {
	fmt.Println("parsing method ID from header...") // TODO delete

	unmarshaledHeader := &RPCHeader{}

	err := bson.Unmarshal(header, unmarshaledHeader)
	if err != nil {
		fmt.Println("could not unmarshal the header of the message: ") // TODO delete
		return RPCMethod(0), errors.Wrap(err, "could not unmarshal the header of the message: ")
	}

	return RPCMethod(unmarshaledHeader.MethodID), nil
}

// parseTopic takes care of parsing the topic and updating the node's feeds
func (h *HobbitsNode) parseTopic(message HobbitsMessage) (string, error) {
	header := GossipHeader{}

	err := bson.Unmarshal(message.Header, header)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling gossip message header: ")
	}

	// TODO: checks against topicMapping?
	// TODO: somehow updates h.Feeds?
	return header.topic, nil
}
