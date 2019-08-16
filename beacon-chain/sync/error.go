package sync

import (
	"bytes"
	"errors"
	"io"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

const genericError = "internal service error"

var errWrongForkVersion = errors.New("wrong fork version")

var responseCodeSuccess = byte(0x00)
var responseCodeInvalidRequest = byte(0x01)
var responseCodeServerError = byte(0x02)

func (r *RegularSync) generateErrorResponse(code byte, reason string) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{code})
	if _, err := r.p2p.Encoding().Encode(buf, &pb.ErrorMessage{ErrorMessage: reason}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (r *RegularSync) readStatusCode(stream io.Reader) (uint8, *pb.ErrorMessage, error) {
	b := make([]byte, 1)
	_, err := stream.Read(b)
	if err != nil {
		return 0, nil, err
	}

	if b[0] == responseCodeSuccess {
		return 0, nil, nil
	}

	msg := &pb.ErrorMessage{}
	if err := r.p2p.Encoding().Decode(stream, msg); err != nil {
		return 0, nil, err
	}

	return uint8(b[0]), msg, nil
}
