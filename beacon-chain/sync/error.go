package sync

import (
	"bytes"
	"errors"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

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
	msg := &types.ErrorMessage{}
	if err := encoding.DecodeWithMaxLength(stream, msg); err != nil {
		return 0, "", err
	}

	return b[0], string(*msg), nil
}

func writeErrorResponseToStream(responseCode byte, reason string, stream libp2pcore.Stream, encoder p2p.EncodingProvider) {
	resp, err := createErrorResponse(responseCode, reason, encoder)
	if err != nil {
		log.WithError(err).Debug("Could not generate a response error")
	} else if _, err := stream.Write(resp); err != nil {
		log.WithError(err).Debugf("Could not write to stream")
	}
}

func createErrorResponse(code byte, reason string, encoder p2p.EncodingProvider) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{code})
	errMsg := types.ErrorMessage(reason)
	if _, err := encoder.Encoding().EncodeWithMaxLength(buf, &errMsg); err != nil {
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

	msg := &types.ErrorMessage{}
	if err := encoding.DecodeWithMaxLength(stream, msg); err != nil {
		return 0, "", err
	}

	return b[0], string(*msg), nil
}

// only returns true for errors that are valid (no resets or expectedEOF errors).
func isValidStreamError(err error) bool {
	return err != nil && !errors.Is(err, mux.ErrReset) && !errors.Is(err, helpers.ErrExpectedEOF)
}

func closeStream(stream network.Stream, log *logrus.Entry) {
	if err := helpers.FullClose(stream); err != nil && err.Error() != mux.ErrReset.Error() {
		log.WithError(err).Debug("Could not reset stream")
	}
}
