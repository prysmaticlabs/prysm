package sync

import (
	"errors"
	"reflect"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
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
func ReadChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p p2p.P2P, isFirstChunk bool) (interface{}, error) {
	// Handle deadlines differently for first chunk
	if isFirstChunk {
		return readFirstChunkedBlock(stream, chain, p2p)
	}
	blk := &eth.SignedBeaconBlock{}
	if err := readResponseChunk(stream, chain, p2p, blk); err != nil {
		return nil, err
	}
	return blk, nil
}

// readFirstChunkedBlock reads the first chunked block and applies the appropriate deadlines to
// it.
func readFirstChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p p2p.P2P) (interface{}, error) {
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
	blk, err := extractBlockDataType(bytesutil.ToBytes4(rpcCtx), chain)
	if err != nil {
		return nil, err
	}
	err = p2p.Encoding().DecodeWithMaxLength(stream, blk)
	return blk, err
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

func extractBlockDataType(digest [4]byte, chain blockchain.ChainInfoFetcher) (interface{}, error) {
	vRoot := chain.GenesisValidatorRoot()
	for k, t := range types.BlockMap {
		rDigest, err := helpers.ComputeForkDigest(k[:], vRoot[:])
		if err != nil {
			return nil, err
		}
		if rDigest == digest {
			typ := reflect.TypeOf(t)
			if typ.Kind() == reflect.Ptr {
				return reflect.New(typ.Elem()), nil
			}
			return reflect.New(typ), nil
		}

	}
	return nil, errors.New("no valid digest matched")
}
