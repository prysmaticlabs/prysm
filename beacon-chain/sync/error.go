package sync

import (
	"bytes"
	"errors"
	"io"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
)

const genericError = "internal service error"
const rateLimitedError = "rate limited"

var errWrongForkVersion = errors.New("wrong fork version")
var errInvalidEpoch = errors.New("invalid epoch")

var responseCodeSuccess = byte(0x00)
var responseCodeInvalidRequest = byte(0x01)
var responseCodeServerError = byte(0x02)

func (r *Service) generateErrorResponse(code byte, reason string) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{code})
	if _, err := r.p2p.Encoding().EncodeWithLength(buf, []byte(reason)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ReadStatusCode response from a RPC stream.
func ReadStatusCode(stream io.Reader, encoding encoder.NetworkEncoding) (uint8, string, error) {
	b := make([]byte, 1)
	_, err := stream.Read(b)
	if err != nil {
		return 0, "", err
	}

	if b[0] == responseCodeSuccess {
		return 0, "", nil
	}

	msg := make([]byte, 0)
	if err := encoding.DecodeWithLength(stream, &msg); err != nil {
		return 0, "", err
	}

	return b[0], string(msg), nil
}
