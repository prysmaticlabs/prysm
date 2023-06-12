package sync

import (
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
)

// Specifies the fixed size context length.
const forkDigestLength = 4

// writes peer's current context for the expected payload to the stream.
func writeContextToStream(objCtx []byte, stream network.Stream) error {
	// The rpc context for our v2 methods is the fork-digest of
	// the relevant payload. We write the associated fork-digest(context)
	// into the stream for the payload.
	rpcCtx, err := expectRpcContext(stream)
	if err != nil {
		return err
	}
	// Exit early if an empty context is expected.
	if !rpcCtx {
		return nil
	}
	_, err = stream.Write(objCtx)
	return err
}

// reads any attached context-bytes to the payload.
func readContextFromStream(stream network.Stream) ([]byte, error) {
	hasCtx, err := expectRpcContext(stream)
	if err != nil {
		return nil, err
	}
	if !hasCtx {
		return []byte{}, nil
	}
	// Read context (fork-digest) from stream
	b := make([]byte, forkDigestLength)
	if _, err := stream.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func expectRpcContext(stream network.Stream) (bool, error) {
	_, _, version, err := p2p.TopicDeconstructor(string(stream.Protocol()))
	if err != nil {
		return false, err
	}
	switch version {
	case p2p.SchemaVersionV1:
		return false, nil
	case p2p.SchemaVersionV2:
		return true, nil
	default:
		return false, errors.New("invalid version of %s registered for topic: %s")
	}
}

// Minimal interface for a stream with a protocol.
type withProtocol interface {
	Protocol() protocol.ID
}

// Validates that the rpc topic matches the provided version.
func validateVersion(version string, stream withProtocol) error {
	_, _, streamVersion, err := p2p.TopicDeconstructor(string(stream.Protocol()))
	if err != nil {
		return err
	}
	if streamVersion != version {
		return errors.Errorf("stream version of %s doesn't match provided version %s", streamVersion, version)
	}
	return nil
}
