package yamux

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	//TODO
	pool "github.com/libp2p/go-buffer-pool"
)

const (
	sizeOfVersion  = 1
	sizeOfType     = 1
	sizeOfFlags    = 2
	sizeOfStreamID = 4
	sizeOfLength   = 4
	headerSize     = sizeOfVersion + sizeOfType + sizeOfFlags +
		sizeOfStreamID + sizeOfLength
)

const (
	initialStreamWindow = 256 * 1024
	// maxStreamWindow     = 16 * 1024 * 1024
)

const (
	flagSYN uint16 = 1 << iota
	flagACK
	flagFIN
	flagRST
)

const (
	typeData uint8 = iota
	typeWindowUpdate
	typeGoAway
	typePing
)

const (
	goAwayNormal uint32 = iota
	goAwayProtoErr
	goAwayInternalErr
)

const (
	protoVersion uint8 = 0
)

type header [headerSize]byte

type segmentedBuffer struct {
	cap     uint32
	len     uint32
	bm      sync.Mutex
	readPos int
	bPos    int
	b       [][]byte
}

func (h header) Version() uint8 {
	return h[0]
}

func (h header) MsgType() uint8 {
	return h[1]
}

func (h header) Flags() uint16 {
	return binary.BigEndian.Uint16(h[2:4])
}

func (h header) StreamID() uint32 {
	return binary.BigEndian.Uint32(h[4:8])
}

func (h header) Length() uint32 {
	return binary.BigEndian.Uint32(h[8:12])
}

func (h header) String() string {
	return fmt.Sprintf("Vsn:%d Type:%d Flags:%d StreamID:%d Length:%d",
		h.Version(), h.MsgType(), h.Flags(), h.StreamID(), h.Length())
}

func encode(msgType uint8, flags uint16, streamID uint32, length uint32) header {
	var h header
	h[0] = protoVersion
	h[1] = msgType
	binary.BigEndian.PutUint16(h[2:4], flags)
	binary.BigEndian.PutUint32(h[4:8], streamID)
	binary.BigEndian.PutUint32(h[8:12], length)
	return h
}

func newSegmentedBuffer(initialCapacity uint32) segmentedBuffer {
	return segmentedBuffer{cap: initialCapacity, b: make([][]byte, 0, 16)}
}

func asyncNotify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (s *segmentedBuffer) GrowTo(max uint32, force bool) (bool, uint32) {
	s.bm.Lock()
	defer s.bm.Unlock()

	currentWindow := s.cap + s.len
	if currentWindow >= max {
		return force, 0
	}
	delta := max - currentWindow

	if delta < (max/2) && !force {
		return false, 0
	}

	s.cap += delta
	return true, delta
}

func (s *segmentedBuffer) checkOverflow(l uint32) error {
	s.bm.Lock()
	defer s.bm.Unlock()
	if s.cap < l {
		return fmt.Errorf("receive window exceeded (remain: %d, recv: %d)", s.cap, l)
	}
	return nil
}

func (s *segmentedBuffer) Append(input io.Reader, length uint32) error {
	if err := s.checkOverflow(length); err != nil {
		return err
	}

	dst := pool.Get(int(length))
	n, err := io.ReadFull(input, dst)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	s.bm.Lock()
	defer s.bm.Unlock()
	if n > 0 {
		s.len += uint32(n)
		s.cap -= uint32(n)
		// s.b has no available space at the end, but has space at the beginning
		if len(s.b) == cap(s.b) && s.bPos > 0 {
			if s.bPos == len(s.b) {
				// the buffer is empty, so just move pos
				s.bPos = 0
				s.b = s.b[:0]
			} else if s.bPos > cap(s.b)/4 {
				// at least 1/4 of buffer is empty, so shift data to the left to free space at the end
				copied := copy(s.b, s.b[s.bPos:])
				// clear references to copied data
				for i := copied; i < len(s.b); i++ {
					s.b[i] = nil
				}
				s.b = s.b[:copied]
				s.bPos = 0
			}
		}
		s.b = append(s.b, dst[0:n])
	}
	return err
}

func (s *segmentedBuffer) Len() uint32 {
	s.bm.Lock()
	defer s.bm.Unlock()
	return s.len
}

func (s *segmentedBuffer) Read(b []byte) (int, error) {
	s.bm.Lock()
	defer s.bm.Unlock()
	if s.bPos == len(s.b) {
		return 0, io.EOF
	}
	data := s.b[s.bPos][s.readPos:]
	n := copy(b, data)
	if n == len(data) {
		pool.Put(s.b[s.bPos])
		s.b[s.bPos] = nil
		s.bPos++
		s.readPos = 0
	} else {
		s.readPos += n
	}
	if n > 0 {
		s.len -= uint32(n)
	}
	return n, nil
}
