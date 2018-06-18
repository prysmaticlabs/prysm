package protocol

import (
	"bufio"
	"context"
	"log"

	pb "github.com/ethereum/go-ethereum/sharding/p2p/proto"
	"github.com/gogo/protobuf/proto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

// pattern: /protocol-name/request-or-response-message/version
const pingRequest = "/ping/pingreq/0.0.1"
const pingResponse = "/ping/pingresp/0.0.1"

type PingProtocol struct {
	requests map[string]*pb.PingRequest
	host     host.Host
}

func NewPingProtocol(h host.Host) *PingProtocol {
	p := &PingProtocol{host: h, requests: make(map[string]*pb.PingRequest)}
	h.SetStreamHandler(pingRequest, p.onPingRequest)
	h.SetStreamHandler(pingResponse, p.onPingResponse)
	return p
}

func (p *PingProtocol) onPingRequest(s inet.Stream) {
	data := &pb.PingRequest{}
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(s))
	err := decoder.Decode(data)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("%s: Received ping request from %s. Message: %s", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.Msg)

	// TODO: validate the message signature.
}

func (p *PingProtocol) onPingResponse(s inet.Stream) {

}

func (p *PingProtocol) Ping(ctx context.Context) {
	req := &pb.PingRequest{Msg: "Ping"}

	s, err := p.host.NewStream(ctx, p.host.ID(), pingRequest)
	if err != nil {
		log.Println(err)
		return
	}

	sendProtoMessage(req, s)

}

// helper method - writes a protobuf go data object to a network stream
// data: reference of protobuf go data object to send (not the object itself)
// s: network stream to write the data to
func sendProtoMessage(data proto.Message, s inet.Stream) bool {
	writer := bufio.NewWriter(s)
	enc := protobufCodec.Multicodec(nil).Encoder(writer)
	err := enc.Encode(data)
	if err != nil {
		log.Println(err)
		return false
	}
	writer.Flush()
	return true
}
