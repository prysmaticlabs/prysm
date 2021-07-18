package sync

import (
	ssz "github.com/ferranbt/fastssz"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
)

// chunkWriter writes the given message as a chunked response to the given network
// stream.
// response_chunk  ::= <result> | <context-bytes> | <encoding-dependent-header> | <encoded-payload>
func (s *Service) chunkWriter(stream libp2pcore.Stream, msg ssz.Marshaler) error {
	SetStreamWriteDeadline(stream, defaultWriteDuration)
	return WriteChunk(stream, s.cfg.Chain, s.cfg.P2P.Encoding(), msg)
}

// WriteChunk object to stream.
// response_chunk  ::= <result> | <context-bytes> | <encoding-dependent-header> | <encoded-payload>
func WriteChunk(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, encoding encoder.NetworkEncoding, msg ssz.Marshaler) error {
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
func ReadChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p p2p.P2P, isFirstChunk bool) (interfaces.SignedBeaconBlock, error) {
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
func readFirstChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p p2p.P2P) (interfaces.SignedBeaconBlock, error) {
	code, errMsg, err := ReadStatusCode(stream, p2p.Encoding())
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, errors.New(errMsg)
	}
	rpcCtx, err := readContextFromStream(stream, chain)
	if err != nil {
		return nil, err
	}
	blk, err := extractBlockDataType(rpcCtx, chain)
	if err != nil {
		return nil, err
	}
	// This may not work, double check tests.
	err = p2p.Encoding().DecodeWithMaxLength(stream, blk)
	return wrapper.WrappedPhase0SignedBeaconBlock(blk), err
}

// readResponseChunk reads the response from the stream and decodes it into the
// provided message type.
func readResponseChunk(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p p2p.P2P) (interfaces.SignedBeaconBlock, error) {
	SetStreamReadDeadline(stream, respTimeout)
	code, errMsg, err := readStatusCodeNoDeadline(stream, p2p.Encoding())
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, errors.New(errMsg)
	}
	// No-op for now with the rpc context.
	rpcCtx, err := readContextFromStream(stream, chain)
	if err != nil {
		return nil, err
	}
	blk, err := extractBlockDataType(rpcCtx, chain)
	if err != nil {
		return nil, err
	}
	// This may not work, double check tests.
	err = p2p.Encoding().DecodeWithMaxLength(stream, blk)
	return blk, err
}

func extractBlockDataType(digest []byte, chain blockchain.ChainInfoFetcher) (interfaces.SignedBeaconBlock, error) {
	if len(digest) == 0 {
		bFunc, ok := types.BlockMap[bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion)]
		if !ok {
			return nil, errors.New("no block type exists for the genesis fork version.")
		}
		return bFunc(), nil
	}
	if len(digest) != digestLength {
		return nil, errors.Errorf("invalid digest returned, wanted a length of %d but received %d", digestLength, len(digest))
	}
	vRoot := chain.GenesisValidatorRoot()
	for k, blkFunc := range types.BlockMap {
		rDigest, err := helpers.ComputeForkDigest(k[:], vRoot[:])
		if err != nil {
			return nil, err
		}
		if rDigest == bytesutil.ToBytes4(digest) {
			return blkFunc(), nil
		}
	}
	return nil, errors.New("no valid digest matched")
}
