package p2p

import (
	ggio "github.com/gogo/protobuf/io"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// setHandshakeHandler to respond to requests for p2p handshake messages.
func setHandshakeHandler(host host.Host, contractAddress string) {
	host.SetStreamHandler(handshakeProtocol, func(stream inet.Stream) {
		defer stream.Close()
		log.Debug("Handling handshake stream")
		w := ggio.NewDelimitedWriter(stream)
		defer w.Close()

		hs := &pb.Handshake{DepositContractAddress: contractAddress}
		if err := w.WriteMsg(hs); err != nil {
			log.WithError(err).Error("Failed to write handshake response")
		}
	})
}
