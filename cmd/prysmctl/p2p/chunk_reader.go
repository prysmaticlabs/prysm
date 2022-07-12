package p2p

import (
	libp2pcore "github.com/libp2p/go-libp2p-core"
	corenet "github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	log "github.com/sirupsen/logrus"
)

func closeStream(stream corenet.Stream) {
	if err := stream.Close(); err != nil {
		log.Println(err)
	}
}

// ReadChunkedBlock handles each response chunk that is sent by the
// peer and converts it into a beacon block.
func ReadChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p *client, isFirstChunk bool) (interfaces.SignedBeaconBlock, error) {
	// Handle deadlines differently for first chunk
	if isFirstChunk {
		return readFirstChunkedBlock(stream, chain, p2p)
	}

	return readResponseChunk(stream, chain, p2p)
}

// readFirstChunkedBlock reads the first chunked block and applies the appropriate deadlines to
// it.
func readFirstChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p *client) (interfaces.SignedBeaconBlock, error) {
	code, errMsg, err := sync.ReadStatusCode(stream, p2p.Encoding())
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
	err = p2p.Encoding().DecodeWithMaxLength(stream, blk)
	return blk, err
}

// readResponseChunk reads the response from the stream and decodes it into the
// provided message type.
func readResponseChunk(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p *client) (interfaces.SignedBeaconBlock, error) {
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
	err = p2p.Encoding().DecodeWithMaxLength(stream, blk)
	return blk, err
}

func extractBlockDataType(digest []byte, chain blockchain.ChainInfoFetcher) (interfaces.SignedBeaconBlock, error) {
	if len(digest) == 0 {
		bFunc, ok := p2ptypes.BlockMap[bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion)]
		if !ok {
			return nil, errors.New("no block type exists for the genesis fork version.")
		}
		return bFunc()
	}
	if len(digest) != 4 {
		return nil, errors.Errorf("invalid digest returned, wanted a length of %d but received %d", 4, len(digest))
	}
	vRoot := chain.GenesisValidatorsRoot()
	for k, blkFunc := range p2ptypes.BlockMap {
		rDigest, err := signing.ComputeForkDigest(k[:], vRoot[:])
		if err != nil {
			return nil, err
		}
		if rDigest == bytesutil.ToBytes4(digest) {
			return blkFunc()
		}
	}
	return nil, errors.New("no valid digest matched")
}

// reads any attached context-bytes to the payload.
func readContextFromStream(stream corenet.Stream, chain blockchain.ChainInfoFetcher) ([]byte, error) {
	rpcCtx, err := rpcContext(stream, chain)
	if err != nil {
		return nil, err
	}
	if len(rpcCtx) == 0 {
		return []byte{}, nil
	}
	// Read context (fork-digest) from stream
	b := make([]byte, 4)
	if _, err := stream.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// retrieve expected context depending on rpc topic schema version.
func rpcContext(stream corenet.Stream, chain blockchain.ChainInfoFetcher) ([]byte, error) {
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

// reads data from the stream without applying any timeouts.
func readStatusCodeNoDeadline(stream corenet.Stream, encoding encoder.NetworkEncoding) (uint8, string, error) {
	b := make([]byte, 1)
	_, err := stream.Read(b)
	if err != nil {
		return 0, "", err
	}

	if b[0] == responseCodeSuccess {
		return 0, "", nil
	}

	msg := &p2ptypes.ErrorMessage{}
	if err := encoding.DecodeWithMaxLength(stream, msg); err != nil {
		return 0, "", err
	}

	return b[0], string(*msg), nil
}
