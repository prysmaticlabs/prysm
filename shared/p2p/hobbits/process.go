package hobbits

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/renaynay/go-hobbits/encoding"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

func (h *HobbitsNode) processHobbitsMessage(id peer.ID, message HobbitsMessage) error {
	switch message.Protocol {
	case encoding.RPC:
		log.Trace("beginning to process the RPC message...")

		err := h.processRPC(id, message)
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

func (h *HobbitsNode) processRPC(id peer.ID, message HobbitsMessage) error { // TODO all of this needs to be put into funcs bc this function is getting disgusting.
	method, err := h.parseMethodID(message.Header)
	if err != nil {
		log.Trace("method id could not be parsed from message header")
		return errors.Wrap(err, "could not parse method_id")
	}

	switch method {
	case HELLO:
		log.Trace("HELLO received")

		err := h.sendHello(id, message)
		if err != nil {
			return errors.Wrap(err, "could not send HELLO response")
		}

		return nil
	case GOODBYE:
		err := h.removePeer(id)
		if err != nil {
			return errors.Wrap(err, "error handling GOODBYE")
		}

		log.Trace("GOODBYE successful")
		return nil
	case GET_STATUS:

		err := h.status(id, message)
		if err != nil {
			return errors.Wrap(err, "could not get status")
		}

		return nil
	case GET_BLOCK_HEADERS:
		err := h.blockHeadersRequest(id, message)
		if err != nil {
			return errors.Wrap(err, "could not retrieve block headers")
		}

		return nil
	case BLOCK_HEADERS:
		//TODO
		// log this?

		return nil
	case GET_BLOCK_BODIES:
		err := h.blockBodyRequest(id, message)
		if err != nil {
			return errors.Wrap(err, "could not retrieve block bodies")
		}

		return nil
	case BLOCK_BODIES:
		//TODO
		// log this somehow?

		return nil
	case GET_ATTESTATION:
		err := h.attestationRequest(id, message)
		if err != nil {
			return errors.Wrap(err, "could not retrieve attestationRequest")
		}

		return nil
	case ATTESTATION:
		//TODO
		// log this somehow?

		return nil
	}

	return nil
}

func (h *HobbitsNode) processGossip(message HobbitsMessage) error {
	log.Trace("processing GOSSIP message")

	// TODO: parse message hash so that it doesn't process already-gossiped messages

	_, err := h.parseTopic(message)
	if err != nil {
		return errors.Wrap(err, "error parsing topic")
	}

	// TODO, does the node log this shit?
	//  maybe the message hash for feedback purposes?



	h.Broadcast(context.Background(), nil) // TODO: marshal into proto.Message

	return nil
}

func (h *HobbitsNode) parseMethodID(header []byte) (RPCMethod, error) {
	fmt.Println("parsing method ID from header...") // TODO delete

	unmarshaledHeader := &RPCHeader{}

	err := bson.Unmarshal(header, unmarshaledHeader)
	if err != nil {
		fmt.Println("could not unmarshal the header of the message") // TODO delete
		return RPCMethod(0), errors.Wrap(err, "could not unmarshal the header of the message")
	}

	return RPCMethod(unmarshaledHeader.MethodID), nil
}

// parseTopic takes care of parsing the topic and updating the node's feeds
func (h *HobbitsNode) parseTopic(message HobbitsMessage) (string, error) {
	header := new(GossipHeader)

	err := bson.Unmarshal(message.Header, header)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling gossip message header")
	}

	return header.topic, nil
}

func (h *HobbitsNode) Feed(msg proto.Message) p2p.Feed {
	t := messageTopic(msg)

	h.Lock()
	defer h.Unlock()

	if h.feeds[t] == nil {
		h.feeds[t] = new(event.Feed)
	}

	return h.feeds[t]
}

func messageTopic(msg proto.Message) reflect.Type {
	return reflect.ValueOf(msg).Elem().Type()
}
