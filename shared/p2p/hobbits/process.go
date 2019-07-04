package hobbits

import (
	"net"
	"reflect"

	"github.com/prysmaticlabs/prysm/bazel-prysm/external/go_sdk/src/context"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/renaynay/go-hobbits/encoding"
	"gopkg.in/mgo.v2/bson"

	"github.com/pkg/errors"
)

func (h *HobbitsNode) processHobbitsMessage(message HobbitsMessage, conn net.Conn) error {
	switch message.Protocol {
	case encoding.RPC:
		err := h.processRPC(message, conn)
		if err != nil {
			return errors.Wrap(err, "error processing an RPC hobbits message")
		}

		return nil
	case encoding.GOSSIP:
		err := h.processGossip(message)
		if err != nil {
			return errors.Wrap(err, "error processing a GOSSIP hobbits message")
		}

		return nil
	}

	return errors.New("protocol unsupported")
}

func (h *HobbitsNode) processRPC(message HobbitsMessage, conn net.Conn) error {
	method, err := h.parseMethodID(message.Header)
	if err != nil {
		return errors.Wrap(err, "could not parse method_id: ")
	}

	switch method {
	case HELLO:
		response := h.rpcHello(message)

		responseBody, err := bson.Marshal(response)

		responseMessage := HobbitsMessage{
			Version:  message.Version,
			Protocol: message.Protocol,
			Header:   message.Header,
			Body:     responseBody,
		}

		err = h.Server.SendMessage(conn, encoding.Message(responseMessage))
		if err != nil {
			return errors.Wrap(err, "error sending hobbits message: ")
		}
	case GOODBYE:
		err := h.removePeer(conn)
		if err != nil {
			return errors.Wrap(err, "error handling GOODBYE: ")
		}
	case GET_STATUS:
		// TODO: retrieve data and call h.Send
	case GET_BLOCK_HEADERS:

		// TODO: retrieve data and call h.Send
	case BLOCK_HEADERS:
		// TODO: call Broadcast?
	case GET_BLOCK_BODIES:
		// TODO: how will this work if batchedBeaconBlockRequest uses finalizedRoot and CanonicalRoot?
	case BLOCK_BODIES:
		// TODO: call Broadcast?
	case GET_ATTESTATION:
		// TODO: retrieve data and call h.Send
	case ATTESTATION:
		// TODO: retrieve data and call h.Send
	}

	return nil
}

func (h *HobbitsNode) rpcHello(message HobbitsMessage) Hello {
	var response Hello

	response.NodeID = h.NodeId

	response.BestRoot = h.DB.HeadStateRoot()

	headState, err := h.DB.HeadState(context.Background())
	if err != nil {
		response.BestSlot = 0
	}

	response.BestSlot = headState.Slot // best slot

	finalizedState, err := h.DB.FinalizedState()
	if err != nil {
		finalizedState = nil
	}

	response.LatestFinalizedEpoch = finalizedState.Slot / 64 // finalized epoch

	hashedFinalizedState, err := hashutil.HashProto(finalizedState) // finalized root
	if err != nil {
		response.LatestFinalizedRoot = [32]byte{}
	}
	response.LatestFinalizedRoot = hashedFinalizedState

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
		return errors.New("error removing peer from node's static peers")
	}

	h.PeerConns = append(h.PeerConns[:index], h.PeerConns[index+1:]...) // TODO: is there a better way to delete
	// TODO: an element from an array by its value?

	return nil
}

func (h *HobbitsNode) processGossip(message HobbitsMessage) error {
	_, err := h.parseTopic(message)
	if err != nil {
		return errors.Wrap(err, "error parsing topic: ")
	}

	err = h.Broadcast(message)
	if err != nil {
		return errors.Wrap(err, "error broadcasting: ")
	}

	return nil
}

func (h *HobbitsNode) parseMethodID(header []byte) (RPCMethod, error) {
	return RPCMethod(0), nil
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
