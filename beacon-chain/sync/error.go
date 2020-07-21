package sync

import (
	"bytes"
	"errors"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const genericError = "internal service error"
const rateLimitedError = "rate limited"
const stepError = "invalid range or step"

var errWrongForkDigestVersion = errors.New("wrong fork digest version")
var errInvalidEpoch = errors.New("invalid epoch")
var errInvalidFinalizedRoot = errors.New("invalid finalized root")
var errGeneric = errors.New(genericError)

var responseCodeSuccess = byte(0x00)
var responseCodeInvalidRequest = byte(0x01)
var responseCodeServerError = byte(0x02)

func (s *Service) generateErrorResponse(code byte, reason string) ([]byte, error) {
	return createErrorResponse(code, reason, s.p2p)
}

// ReadStatusCode response from a RPC stream.
func ReadStatusCode(stream network.Stream, encoding encoder.NetworkEncoding) (uint8, string, error) {
	// Set ttfb deadline.
	SetStreamReadDeadline(stream, params.BeaconNetworkConfig().TtfbTimeout)
	b := make([]byte, 1)
	_, err := stream.Read(b)
	if err != nil {
		return 0, "", err
	}

	if b[0] == responseCodeSuccess {
		// Set response deadline on a successful response code.
		SetStreamReadDeadline(stream, params.BeaconNetworkConfig().RespTimeout)

		return 0, "", nil
	}

	// Set response deadline, when reading error message.
	SetStreamReadDeadline(stream, params.BeaconNetworkConfig().RespTimeout)
	msg := &pb.ErrorResponse{
		Message: []byte{},
	}
	if err := encoding.DecodeWithMaxLength(stream, msg); err != nil {
		return 0, "", err
	}

	return b[0], string(msg.Message), nil
}

func writeErrorResponseToStream(responseCode byte, reason string, stream libp2pcore.Stream, encoder p2p.EncodingProvider) {
	resp, err := createErrorResponse(responseCode, reason, encoder)
	if err != nil {
		log.WithError(err).Error("Failed to generate a response error")
	} else {
		if _, err := stream.Write(resp); err != nil {
			log.WithError(err).Errorf("Failed to write to stream")
		}
	}
}

func createErrorResponse(code byte, reason string, encoder p2p.EncodingProvider) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{code})
	resp := &pb.ErrorResponse{
		Message: []byte(reason),
	}
	if _, err := encoder.Encoding().EncodeWithMaxLength(buf, resp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// reads data from the stream without applying any timeouts.
func readStatusCodeNoDeadline(stream network.Stream, encoding encoder.NetworkEncoding) (uint8, string, error) {
	b := make([]byte, 1)
	_, err := stream.Read(b)
	if err != nil {
		return 0, "", err
	}

	if b[0] == responseCodeSuccess {
		return 0, "", nil
	}

	msg := &pb.ErrorResponse{
		Message: []byte{},
	}
	if err := encoding.DecodeWithMaxLength(stream, msg); err != nil {
		return 0, "", err
	}

	return b[0], string(msg.Message), nil
}
