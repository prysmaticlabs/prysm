package hobbits

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
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

type AttestationRequest struct {
	Signature []byte `bson:"signature"`
}

type AttestationResponse struct {
	Attestation pb.Attestation `bson:"attestation"`
}

func (h *HobbitsNode) status(id peer.ID, message HobbitsMessage) error {
	// does something with the status of the other node
	responseBody := Status{
		UserAgent: []byte(fmt.Sprintf("prysm node %d", h.NodeId)),
		Timestamp: uint64(time.Now().Unix()),
	}

	body, err := bson.Marshal(responseBody)
	if err != nil {
		return errors.Wrap(err, "error marshaling response body")
	}

	responseMessage := HobbitsMessage{
		Version:  message.Version,
		Protocol: message.Protocol,
		Header:   message.Header,
		Body:     body,
	}

	err = h.Server.SendMessage(h.PeerConns[id], encoding.Message(responseMessage))
	if err != nil {
		return errors.Wrap(err, "error sending GET_STATUS")
	}

	return nil
}

func (h *HobbitsNode) sendHello(id peer.ID, message HobbitsMessage) error {
	response := h.rpcHello()

	responseBody, err := bson.Marshal(response)

	responseMessage := HobbitsMessage{
		Version:  message.Version,
		Protocol: message.Protocol,
		Header:   message.Header,
		Body:     responseBody,
	}
	log.Trace(responseMessage)

	err = h.Server.SendMessage(h.PeerConns[id], encoding.Message(responseMessage))
	if err != nil {
		log.Trace("error sending a HELLO back") // TODO delete
		return errors.Wrap(err, "error sending HELLO")
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

func (h *HobbitsNode) removePeer(id peer.ID) error {
	peer := h.PeerConns[id]
	delete(h.PeerConns, id)

	err := peer.Close()
	if err != nil {
		return errors.Wrap(err, "error closing connection on peer")
	}
	//
	//index = 0
	//
	//for i, p := range h.StaticPeers {
	//	if reflect.DeepEqual(peer.RemoteAddr().String(), p) {
	//		index = i
	//	}
	//}
	//if index == 0 {
	//	return errors.New("error removing peer from node's static peers")
	//}
	//h.StaticPeers = append(h.StaticPeers[:index], h.StaticPeers[index+1:]...)
	//
	return nil
}

func (h *HobbitsNode) blockHeadersRequest(id peer.ID, message HobbitsMessage) error {


	// var request BlockRequest // TODO: might not need BlockRequest struct, just unmarshal into protobuf
	//err := bson.Unmarshal(message.Body, request)
	//if err != nil {
	//	return errors.Wrap(err, "could not unmarshal block header RPC request: ")
	//}

	return nil
}

func (h *HobbitsNode) blockBodiesRequest(id peer.ID, message HobbitsMessage) error {
	var requestBody BlockRequest
	err := bson.Unmarshal(message.Body, requestBody)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal body of GET_BLOCK_BODY request")
	}

	var bbr pb.BatchedBeaconBlockRequest
	bbr.StartSlot = requestBody.StartSlot

	// Todo: BBR in Send() needs to be replaced with a P2P.Message
	h.Feed(&bbr).Send(bbr)

	//var msg proto.Message
	//msg.

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

func (h *HobbitsNode) attestationRequest(id peer.ID, message HobbitsMessage) error {
	var requestBody AttestationRequest

	err := bson.Unmarshal(message.Body, requestBody)
	if err != nil {
		return errors.Wrap(err, "error unmarshaling body of GET_ATTESTATION request")
	}

	ar := &pb.AttestationRequest{
		Hash: requestBody.Signature,
	}

	h.Feed(ar).Send(p2p.Message{
		Ctx: context.Background(),
		Data: ar,
		Peer: id,
	})

	return nil
}

func (h *HobbitsNode) attestationResponse(msg proto.Message) (HobbitsMessage, error) {
	response := AttestationResponse{
		Attestation: *msg.(*pb.AttestationResponse).Attestation,
	}
	body, err := bson.Marshal(response)
	if err != nil {
		return HobbitsMessage{}, errors.Wrap(err, "error marshaling body for ATTESTATION response")
	}

	head := RPCHeader{
		MethodID: 0x0F,
	}
	header, err := bson.Marshal(head)
	if err != nil {
		return HobbitsMessage{}, errors.Wrap(err, "error marshaling header for ATTESTATION response")
	}

	return HobbitsMessage{
		Version: uint32(3),
		Protocol: encoding.RPC,
		Header: header,
		Body: body,
	}, nil
}