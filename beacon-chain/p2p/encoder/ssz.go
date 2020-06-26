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

// MaxChunkSize allowed for decoding messages.
var MaxChunkSize = params.BeaconNetworkConfig().MaxChunkSize // 1Mib

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
type SszNetworkEncoder struct {
	UseSnappyCompression bool
}

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
	if e.UseSnappyCompression {
		b = snappy.Encode(nil /*dst*/, b)
	}
	return w.Write(b)
}

// EncodeWithLength the proto message to the io.Writer. This encoding prefixes the byte slice with a protobuf varint
// to indicate the size of the message.
func (e SszNetworkEncoder) EncodeWithLength(w io.Writer, msg interface{}) (int, error) {
	if msg == nil {
		return 0, nil
	}
	b, err := e.doEncode(msg)
	if err != nil {
		return 0, err
	}
	// write varint first
	_, err = w.Write(proto.EncodeVarint(uint64(len(b))))
	if err != nil {
		return 0, err
	}
	if e.UseSnappyCompression {
		return writeSnappyBuffer(w, b)
	}
	return w.Write(b)
}

// EncodeWithMaxLength the proto message to the io.Writer. This encoding prefixes the byte slice with a protobuf varint
// to indicate the size of the message. This checks that the encoded message isn't larger than the provided max limit.
func (e SszNetworkEncoder) EncodeWithMaxLength(w io.Writer, msg interface{}, maxSize uint64) (int, error) {
	if msg == nil {
		return 0, nil
	}
	b, err := e.doEncode(msg)
	if err != nil {
		return 0, err
	}
	if uint64(len(b)) > maxSize {
		return 0, fmt.Errorf("size of encoded message is %d which is larger than the provided max limit of %d", len(b), maxSize)
	}
	// write varint first
	_, err = w.Write(proto.EncodeVarint(uint64(len(b))))
	if err != nil {
		return 0, err
	}
	if e.UseSnappyCompression {
		return writeSnappyBuffer(w, b)
	}
	return w.Write(b)
}

func (e SszNetworkEncoder) doDecode(b []byte, to interface{}) error {
	if v, ok := to.(fastssz.Unmarshaler); ok {
		return v.UnmarshalSSZ(b)
	}
	err := ssz.Unmarshal(b, to)
	if err != nil {
		// Check if we are unmarshalling block roots
		// and then lop off the 4 byte offset and try
		// unmarshalling again. This is temporary to
		// avoid too much disruption to onyx nodes.
		// TODO(#6408)
		if _, ok := to.(*[][32]byte); ok {
			return ssz.Unmarshal(b[4:], to)
		}
		return err
	}
	return nil
}

// DecodeGossip decodes the bytes to the protobuf gossip message provided.
func (e SszNetworkEncoder) DecodeGossip(b []byte, to interface{}) error {
	if e.UseSnappyCompression {
		var err error
		b, err = snappy.Decode(nil /*dst*/, b)
		if err != nil {
			return err
		}
	}
	if uint64(len(b)) > MaxGossipSize {
		return errors.Errorf("gossip message exceeds max gossip size: %d bytes > %d bytes", len(b), MaxGossipSize)
	}
	return e.doDecode(b, to)
}

// DecodeWithLength the bytes from io.Reader to the protobuf message provided.
func (e SszNetworkEncoder) DecodeWithLength(r io.Reader, to interface{}) error {
	return e.DecodeWithMaxLength(r, to, MaxChunkSize)
}

// DecodeWithMaxLength the bytes from io.Reader to the protobuf message provided.
// This checks that the decoded message isn't larger than the provided max limit.
func (e SszNetworkEncoder) DecodeWithMaxLength(r io.Reader, to interface{}, maxSize uint64) error {
	if maxSize > MaxChunkSize {
		return fmt.Errorf("maxSize %d exceeds max chunk size %d", maxSize, MaxChunkSize)
	}
	msgLen, err := readVarint(r)
	if err != nil {
		return err
	}
	if msgLen > maxSize {
		return fmt.Errorf("remaining bytes %d goes over the provided max limit of %d", msgLen, maxSize)
	}
	if e.UseSnappyCompression {
		r = newBufferedReader(r)
		defer bufReaderPool.Put(r)
	}
	b := make([]byte, e.MaxLength(int(msgLen)))
	numOfBytes, err := r.Read(b)
	if err != nil {
		return err
	}
	return e.doDecode(b[:numOfBytes], to)
}

// ProtocolSuffix returns the appropriate suffix for protocol IDs.
func (e SszNetworkEncoder) ProtocolSuffix() string {
	if e.UseSnappyCompression {
		return "/ssz_snappy"
	}
	return "/ssz"
}

// MaxLength specifies the maximum possible length of an encoded
// chunk of data.
func (e SszNetworkEncoder) MaxLength(length int) int {
	if e.UseSnappyCompression {
		return snappy.MaxEncodedLen(length)
	}
	return length
}

// Writes a bytes value through a snappy buffered writer.
func writeSnappyBuffer(w io.Writer, b []byte) (int, error) {
	bufWriter := newBufferedWriter(w)
	defer bufWriterPool.Put(bufWriter)
	num, err := bufWriter.Write(b)
	if err != nil {
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
