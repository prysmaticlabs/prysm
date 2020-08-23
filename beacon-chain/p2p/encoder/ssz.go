package encoder

import (
	"fmt"
	"io"
	"sync"

	fastssz "github.com/ferranbt/fastssz"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var _ = NetworkEncoding(&SszNetworkEncoder{})

// MaxGossipSize allowed for gossip messages.
var MaxGossipSize = params.BeaconNetworkConfig().GossipMaxSize // 1 Mib

// This pool defines the sync pool for our buffered snappy writers, so that they
// can be constantly reused.
var bufWriterPool = new(sync.Pool)

// This pool defines the sync pool for our buffered snappy readers, so that they
// can be constantly reused.
var bufReaderPool = new(sync.Pool)

// SszNetworkEncoder supports p2p networking encoding using SimpleSerialize
// with snappy compression (if enabled).
type SszNetworkEncoder struct{}

func (e SszNetworkEncoder) doEncode(msg interface{}) ([]byte, error) {
	if v, ok := msg.(fastssz.Marshaler); ok {
		return v.MarshalSSZ()
	}
	return ssz.Marshal(msg)
}

// EncodeGossip the proto gossip message to the io.Writer.
func (e SszNetworkEncoder) EncodeGossip(w io.Writer, msg interface{}) (int, error) {
	if msg == nil {
		return 0, nil
	}
	b, err := e.doEncode(msg)
	if err != nil {
		return 0, err
	}
	if uint64(len(b)) > MaxGossipSize {
		return 0, errors.Errorf("gossip message exceeds max gossip size: %d bytes > %d bytes", len(b), MaxGossipSize)
	}
	b = snappy.Encode(nil /*dst*/, b)
	return w.Write(b)
}

// EncodeWithMaxLength the proto message to the io.Writer. This encoding prefixes the byte slice with a protobuf varint
// to indicate the size of the message. This checks that the encoded message isn't larger than the provided max limit.
func (e SszNetworkEncoder) EncodeWithMaxLength(w io.Writer, msg interface{}) (int, error) {
	if msg == nil {
		return 0, nil
	}
	b, err := e.doEncode(msg)
	if err != nil {
		return 0, err
	}
	if uint64(len(b)) > params.BeaconNetworkConfig().MaxChunkSize {
		return 0, fmt.Errorf(
			"size of encoded message is %d which is larger than the provided max limit of %d",
			len(b),
			params.BeaconNetworkConfig().MaxChunkSize,
		)
	}
	// write varint first
	_, err = w.Write(proto.EncodeVarint(uint64(len(b))))
	if err != nil {
		return 0, err
	}
	return writeSnappyBuffer(w, b)
}

func (e SszNetworkEncoder) doDecode(b []byte, to interface{}) error {
	if v, ok := to.(fastssz.Unmarshaler); ok {
		return v.UnmarshalSSZ(b)
	}
	return ssz.Unmarshal(b, to)
}

// DecodeGossip decodes the bytes to the protobuf gossip message provided.
func (e SszNetworkEncoder) DecodeGossip(b []byte, to interface{}) error {
	size, err := snappy.DecodedLen(b)
	if uint64(size) > MaxGossipSize {
		return errors.Errorf("gossip message exceeds max gossip size: %d bytes > %d bytes", size, MaxGossipSize)
	}
	b, err = snappy.Decode(nil /*dst*/, b)
	if err != nil {
		return err
	}
	return e.doDecode(b, to)
}

// DecodeWithMaxLength the bytes from io.Reader to the protobuf message provided.
// This checks that the decoded message isn't larger than the provided max limit.
func (e SszNetworkEncoder) DecodeWithMaxLength(r io.Reader, to interface{}) error {
	msgLen, err := readVarint(r)
	if err != nil {
		return err
	}
	if msgLen > params.BeaconNetworkConfig().MaxChunkSize {
		return fmt.Errorf(
			"remaining bytes %d goes over the provided max limit of %d",
			msgLen,
			params.BeaconNetworkConfig().MaxChunkSize,
		)
	}
	r = newBufferedReader(r)
	defer bufReaderPool.Put(r)
	b := make([]byte, e.MaxLength(int(msgLen)))
	numOfBytes, err := r.Read(b)
	if err != nil {
		return err
	}
	return e.doDecode(b[:numOfBytes], to)
}

// ProtocolSuffix returns the appropriate suffix for protocol IDs.
func (e SszNetworkEncoder) ProtocolSuffix() string {
	return "/ssz_snappy"
}

// MaxLength specifies the maximum possible length of an encoded
// chunk of data.
func (e SszNetworkEncoder) MaxLength(length int) int {
	return snappy.MaxEncodedLen(length)
}

// Writes a bytes value through a snappy buffered writer.
func writeSnappyBuffer(w io.Writer, b []byte) (int, error) {
	bufWriter := newBufferedWriter(w)
	defer bufWriterPool.Put(bufWriter)
	num, err := bufWriter.Write(b)
	if err != nil {
		// Close buf writer in the event of an error.
		if err := bufWriter.Close(); err != nil {
			return 0, err
		}
		return 0, err
	}
	return num, bufWriter.Close()
}

// Instantiates a new instance of the snappy buffered reader
// using our sync pool.
func newBufferedReader(r io.Reader) *snappy.Reader {
	rawReader := bufReaderPool.Get()
	if rawReader == nil {
		return snappy.NewReader(r)
	}
	bufR, ok := rawReader.(*snappy.Reader)
	if !ok {
		return snappy.NewReader(r)
	}
	bufR.Reset(r)
	return bufR
}

// Instantiates a new instance of the snappy buffered writer
// using our sync pool.
func newBufferedWriter(w io.Writer) *snappy.Writer {
	rawBufWriter := bufWriterPool.Get()
	if rawBufWriter == nil {
		return snappy.NewBufferedWriter(w)
	}
	bufW, ok := rawBufWriter.(*snappy.Writer)
	if !ok {
		return snappy.NewBufferedWriter(w)
	}
	bufW.Reset(w)
	return bufW
}
