package multistream

import (
	"errors"
	"io"
)

// ErrNotSupported is the error returned when the muxer does not support
// the protocol specified for the handshake.
var ErrNotSupported = errors.New("protocol not supported")

// SelectProtoOrFail performs the initial multistream handshake
// to inform the muxer of the protocol that will be used to communicate
// on this ReadWriteCloser. It returns an error if, for example,
// the muxer does not know how to handle this protocol.
func SelectProtoOrFail(proto string, rwc io.ReadWriteCloser) error {
	err := handshake(rwc)
	if err != nil {
		return err
	}

	return trySelect(proto, rwc)
}

// SelectOneOf will perform handshakes with the protocols on the given slice
// until it finds one which is supported by the muxer.
func SelectOneOf(protos []string, rwc io.ReadWriteCloser) (string, error) {
	err := handshake(rwc)
	if err != nil {
		return "", err
	}

	for _, p := range protos {
		err := trySelect(p, rwc)
		switch err {
		case nil:
			return p, nil
		case ErrNotSupported:
		default:
			return "", err
		}
	}
	return "", ErrNotSupported
}

func handshake(rwc io.ReadWriteCloser) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- delimWriteBuffered(rwc, []byte(ProtocolID))
	}()

	tok, readErr := ReadNextToken(rwc)
	writeErr := <-errCh

	if writeErr != nil {
		return writeErr
	}
	if readErr != nil {
		return readErr
	}

	if tok != ProtocolID {
		return errors.New("received mismatch in protocol id")
	}
	return nil
}

func trySelect(proto string, rwc io.ReadWriteCloser) error {
	err := delimWriteBuffered(rwc, []byte(proto))
	if err != nil {
		return err
	}

	tok, err := ReadNextToken(rwc)
	if err != nil {
		return err
	}

	switch tok {
	case proto:
		return nil
	case "na":
		return ErrNotSupported
	default:
		return errors.New("unrecognized response: " + tok)
	}
}
