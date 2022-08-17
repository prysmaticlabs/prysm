package sync

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
)

// Specifies the fixed size context length.
const forkDigestLength = 4

// writes peer's current context for the expected payload to the stream.
func writeContextToStream(objCtx []byte, stream network.Stream, chain blockchain.ForkFetcher) error {
	// The rpc context for our v2 methods is the fork-digest of
	// the relevant payload. We write the associated fork-digest(context)
	// into the stream for the payload.
	rpcCtx, err := rpcContext(stream, chain)
	if err != nil {
		return err
	}
	// Exit early if there is an empty context.
	if len(rpcCtx) == 0 {
		return nil
	}
	// Always choose the object's context when writing to the stream.
	if objCtx != nil {
		rpcCtx = objCtx
	}
	_, err = stream.Write(rpcCtx)
	return err
}

// reads any attached context-bytes to the payload.
func readContextFromStream(stream network.Stream, chain blockchain.ForkFetcher) ([]byte, error) {
	rpcCtx, err := rpcContext(stream, chain)
	if err != nil {
		return nil, err
	}
	if len(rpcCtx) == 0 {
		return []byte{}, nil
	}
	// Read context (fork-digest) from stream
	b := make([]byte, forkDigestLength)
	if _, err := stream.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// retrieve expected context depending on rpc topic schema version.
func rpcContext(stream network.Stream, chain blockchain.ForkFetcher) ([]byte, error) {
	_, _, version, err := p2p.TopicDeconstructor(string(stream.Protocol()))
	if err != nil {
		return nil, err
	}
	switch version {
	case p2p.SchemaVersionV1:
		// Return empty context for a v1 method.
		return []byte{}, nil
	case p2p.SchemaVersionV2:
		currFork := chain.CurrentFork()
		genRoot := chain.GenesisValidatorsRoot()
		digest, err := signing.ComputeForkDigest(currFork.CurrentVersion, genRoot[:])
		if err != nil {
			return nil, err
		}
		return digest[:], nil
	default:
		return nil, errors.New("invalid version of %s registered for topic: %s")
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
