package sync

import (
	"errors"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	block2 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
)

// chunkWriter writes the given message as a chunked response to the given network
// stream.
// response_chunk  ::= <result> | <context-bytes> | <encoding-dependent-header> | <encoded-payload>
func (s *Service) chunkWriter(stream libp2pcore.Stream, msg interface{}) error {
	SetStreamWriteDeadline(stream, defaultWriteDuration)
	return WriteChunk(stream, s.cfg.Chain, s.cfg.P2P.Encoding(), msg)
}

// WriteChunk object to stream.
// response_chunk  ::= <result> | <context-bytes> | <encoding-dependent-header> | <encoded-payload>
func WriteChunk(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, encoding encoder.NetworkEncoding, msg interface{}) error {
	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		return err
	}
	if err := writeContextToStream(stream, chain); err != nil {
		return err
	}
	_, err := encoding.EncodeWithMaxLength(stream, msg)
	return err
}

// ReadChunkedBlock handles each response chunk that is sent by the
// peer and converts it into a beacon block.
func ReadChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p p2p.P2P, isFirstChunk bool) (block2.SignedBeaconBlock, error) {
	// Handle deadlines differently for first chunk
	if isFirstChunk {
		return readFirstChunkedBlock(stream, chain, p2p)
	}
	blk := &eth.SignedBeaconBlock{}
	if err := readResponseChunk(stream, chain, p2p, blk); err != nil {
		return nil, err
	}
	return wrapper.WrappedPhase0SignedBeaconBlock(blk), nil
}

// readFirstChunkedBlock reads the first chunked block and applies the appropriate deadlines to
// it.
func readFirstChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p p2p.P2P) (block2.SignedBeaconBlock, error) {
	blk := &eth.SignedBeaconBlock{}
	code, errMsg, err := ReadStatusCode(stream, p2p.Encoding())
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, errors.New(errMsg)
	}
	// No-op for now with the rpc context.
	_, err = readContextFromStream(stream, chain)
	if err != nil {
		return nil, err
	}
	err = p2p.Encoding().DecodeWithMaxLength(stream, blk)
	return wrapper.WrappedPhase0SignedBeaconBlock(blk), err
}

// readResponseChunk reads the response from the stream and decodes it into the
// provided message type.
func readResponseChunk(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p p2p.P2P, to interface{}) error {
	SetStreamReadDeadline(stream, respTimeout)
	code, errMsg, err := readStatusCodeNoDeadline(stream, p2p.Encoding())
	if err != nil {
		return err
	}
	if code != 0 {
		return errors.New(errMsg)
	}
	// No-op for now with the rpc context.
	_, err = readContextFromStream(stream, chain)
	if err != nil {
		return err
	}
	return p2p.Encoding().DecodeWithMaxLength(stream, to)
}
