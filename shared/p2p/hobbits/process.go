package hobbits

import (
	"context"
	"fmt"
	"net"

	"github.com/renaynay/go-hobbits/encoding"
	log "github.com/sirupsen/logrus"
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

		err := h.sendHello(message, conn)
		if err != nil {
			return errors.Wrap(err, "could not send HELLO response: ")
		}

		return nil
	case GOODBYE:
		err := h.removePeer(conn)
		if err != nil {
			return errors.Wrap(err, "error handling GOODBYE: ")
		}

		log.Trace("GOODBYE successful")
		return nil
	case GET_STATUS:

		err := h.status(message, conn)
		if err != nil {
			return errors.Wrap(err, "could not get status: ")
		}

		return nil
	case GET_BLOCK_HEADERS:
		err := h.blockHeaders(message, conn)
		if err != nil {
			return errors.Wrap(err, "could not retrieve block headers: ")
		}

		return nil
	case BLOCK_HEADERS:
		//TODO
		// log this?

		return nil
	case GET_BLOCK_BODIES: // TODO: this is so messed up
		err := h.blockBodies(message, conn)
		if err != nil {
			return errors.Wrap(err, "could not retrieve block bodies: ")
		}

		return nil
	case BLOCK_BODIES:
		//TODO
		// log this somehow?

		return nil
	case GET_ATTESTATION:
		err := h.attestation(message, conn)
		if err != nil {
			return errors.Wrap(err, "could not retrieve attestation: ")
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
