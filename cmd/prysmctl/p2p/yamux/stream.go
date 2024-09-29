package yamux

import (
	"fmt"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"
	// "github.com/libp2p/go-libp2p/core/network"
)

const (
	streamInit streamState = iota
	streamSYNSent
	streamSYNReceived
	streamEstablished
	streamFinished
)

const (
	halfOpen halfStreamState = iota
	halfClosed
	halfReset
)

type streamState int

type halfStreamState int

type Stream struct {
	sendWindow uint32

	memorySpan MemoryManager

	id      uint32
	session *Session

	recvWindow uint32
	epochStart time.Time

	state                 streamState
	writeState, readState halfStreamState
	stateLock             sync.Mutex

	recvBuf segmentedBuffer

	recvNotifyCh chan struct{}
	sendNotifyCh chan struct{}

	readDeadline, writeDeadline pipeDeadline
}

func newStream(session *Session, id uint32, state streamState, initialWindow uint32, memorySpan MemoryManager) *Stream {
	s := &Stream{
		id:            id,
		session:       session,
		state:         state,
		sendWindow:    initialStreamWindow,
		readDeadline:  makePipeDeadline(),
		writeDeadline: makePipeDeadline(),
		memorySpan:    memorySpan,
		// Initialize the recvBuf with initialStreamWindow, not config.InitialStreamWindowSize.
		// The peer isn't allowed to send more data than initialStreamWindow until we've sent
		// the first window update (which will grant it up to config.InitialStreamWindowSize).
		recvBuf:      newSegmentedBuffer(initialWindow),
		recvWindow:   session.config.InitialStreamWindowSize,
		epochStart:   time.Now(),
		recvNotifyCh: make(chan struct{}, 1),
		sendNotifyCh: make(chan struct{}, 1),
	}
	return s
}

func (s *Stream) forceClose() {
	s.stateLock.Lock()
	if s.readState == halfOpen {
		s.readState = halfReset
	}
	if s.writeState == halfOpen {
		s.writeState = halfReset
	}
	s.state = streamFinished
	s.notifyWaiting()
	s.stateLock.Unlock()

	s.readDeadline.set(time.Time{})
	s.writeDeadline.set(time.Time{})
}

// increments how much data can be sent on the stream
func (s *Stream) incrSendWindow(hdr header, flags uint16) {
	s.processFlags(flags)
	atomic.AddUint32(&s.sendWindow, hdr.Length())
	asyncNotify(s.sendNotifyCh)
}

func (s *Stream) readData(hdr header, flags uint16, conn io.Reader) error {
	s.processFlags(flags)

	// Check that our recv window is not exceeded
	length := hdr.Length()
	if length == 0 {
		return nil
	}

	if err := s.recvBuf.Append(conn, length); err != nil {
		s.session.logger.Printf("[ERR] yamux: Failed to read stream data on stream %d: %v", s.id, err)
		return err
	}

	asyncNotify(s.recvNotifyCh)
	return nil
}

func (s *Stream) sendWindowUpdate(deadline <-chan struct{}) error {
	// Determine the flags if any
	flags := s.sendFlags()

	// Update the receive window.
	needed, delta := s.recvBuf.GrowTo(s.recvWindow, flags != 0)
	if !needed {
		return nil
	}

	now := time.Now()
	if rtt := s.session.getRTT(); flags == 0 && rtt > 0 && now.Sub(s.epochStart) < rtt*4 {
		var recvWindow uint32
		if s.recvWindow > math.MaxUint32/2 {
			recvWindow = min(math.MaxUint32, s.session.config.MaxStreamWindowSize)
		} else {
			recvWindow = min(s.recvWindow*2, s.session.config.MaxStreamWindowSize)
		}
		if recvWindow > s.recvWindow {
			grow := recvWindow - s.recvWindow
			if err := s.memorySpan.ReserveMemory(int(grow), 128); err == nil {
				s.recvWindow = recvWindow
				_, delta = s.recvBuf.GrowTo(s.recvWindow, true)
			}
		}
	}

	s.epochStart = now
	hdr := encode(typeWindowUpdate, flags, s.id, delta)
	return s.session.sendMsg(hdr, nil, deadline)
}

func (s *Stream) notifyWaiting() {
	asyncNotify(s.recvNotifyCh)
	asyncNotify(s.sendNotifyCh)
}

func (s *Stream) processFlags(flags uint16) {
	// Close the stream without holding the state lock
	var closeStream bool
	defer func() {
		if closeStream {
			s.cleanup()
		}
	}()

	if flags&flagACK == flagACK {
		s.stateLock.Lock()
		if s.state == streamSYNSent {
			s.state = streamEstablished
		}
		s.stateLock.Unlock()
		s.session.establishStream(s.id)
	}
	if flags&flagFIN == flagFIN {
		var notify bool
		s.stateLock.Lock()
		if s.readState == halfOpen {
			s.readState = halfClosed
			if s.writeState != halfOpen {
				// We're now fully closed.
				closeStream = true
				s.state = streamFinished
			}
			notify = true
		}
		s.stateLock.Unlock()
		if notify {
			s.notifyWaiting()
		}
	}
	if flags&flagRST == flagRST {
		s.stateLock.Lock()
		if s.readState == halfOpen {
			s.readState = halfReset
		}
		if s.writeState == halfOpen {
			s.writeState = halfReset
		}
		s.state = streamFinished
		s.stateLock.Unlock()
		closeStream = true
		s.notifyWaiting()
	}
}

func (s *Stream) cleanup() {
	s.session.closeStream(s.id)
	s.readDeadline.set(time.Time{})
	s.writeDeadline.set(time.Time{})
}

func (s *Stream) sendFlags() uint16 {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	var flags uint16
	switch s.state {
	case streamInit:
		flags |= flagSYN
		s.state = streamSYNSent
	case streamSYNReceived:
		flags |= flagACK
		s.state = streamEstablished
	}
	return flags
}

func (s *Stream) Close() error {
	_ = s.CloseRead()
	return s.CloseWrite()
}

// ensure no more data can be read from the stream
func (s *Stream) CloseRead() error {
	cleanup := false
	s.stateLock.Lock()
	switch s.readState {
	case halfOpen:
	case halfClosed, halfReset:
		s.stateLock.Unlock()
		return nil
	default:
		panic("invalid state")
	}
	s.readState = halfReset
	cleanup = s.writeState != halfOpen
	if cleanup {
		s.state = streamFinished
	}
	s.stateLock.Unlock()
	s.notifyWaiting()
	if cleanup {
		s.cleanup()
	}
	return nil
}

func (s *Stream) CloseWrite() error {
	s.stateLock.Lock()
	switch s.writeState {
	case halfOpen:
	case halfClosed:
		s.stateLock.Unlock()
		return nil
	case halfReset:
		s.stateLock.Unlock()
		return fmt.Errorf("stream reset")
	default:
		panic("invalid state")
	}
	s.writeState = halfClosed
	cleanup := s.readState != halfOpen
	if cleanup {
		s.state = streamFinished
	}
	s.stateLock.Unlock()
	s.notifyWaiting()

	err := s.sendClose()
	if cleanup {
		s.cleanup()
	}
	return err
}

func (s *Stream) sendClose() error {
	flags := s.sendFlags()
	flags |= flagFIN
	hdr := encode(typeWindowUpdate, flags, s.id, 0)
	return s.session.sendMsg(hdr, nil, nil)
}

func (s *Stream) Reset() error {
	sendReset := false
	s.stateLock.Lock()
	switch s.state {
	case streamFinished:
		s.stateLock.Unlock()
		return nil
	case streamInit:
	case streamSYNSent, streamSYNReceived, streamEstablished:
		sendReset = true
	default:
		panic("unhandled state")
	}

	if s.writeState == halfOpen {
		s.writeState = halfReset
	}
	if s.readState == halfOpen {
		s.readState = halfReset
	}
	s.state = streamFinished
	s.notifyWaiting()
	s.stateLock.Unlock()
	if sendReset {
		_ = s.sendReset()
	}
	s.cleanup()
	return nil
}

func (s *Stream) sendReset() error {
	hdr := encode(typeWindowUpdate, flagRST, s.id, 0)
	return s.session.sendMsg(hdr, nil, nil)
}

func (s *Stream) SetDeadline(t time.Time) error {
	if err := s.SetReadDeadline(t); err != nil {
		return err
	}
	if err := s.SetWriteDeadline(t); err != nil {
		return err
	}
	return nil
}

// SetReadDeadline sets the deadline for future Read calls.
func (s *Stream) SetReadDeadline(t time.Time) error {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	if s.readState == halfOpen {
		s.readDeadline.set(t)
	}
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
func (s *Stream) SetWriteDeadline(t time.Time) error {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	if s.writeState == halfOpen {
		s.writeDeadline.set(t)
	}
	return nil
}

func (s *Stream) Read(b []byte) (n int, err error) {
START:
	s.stateLock.Lock()
	state := s.readState
	s.stateLock.Unlock()

	switch state {
	case halfOpen:
	case halfClosed:
		empty := s.recvBuf.Len() == 0
		if empty {
			return 0, io.EOF
		}
	case halfReset:
		return 0, fmt.Errorf("STREAMERR: stream reset")
	default:
		panic("unknown state")
	}

	if s.recvBuf.Len() == 0 {
		select {
		case <-s.recvNotifyCh:
			goto START
		case <-s.readDeadline.wait():
			return 0, fmt.Errorf("SESSIONERR: timeout")
		}
	}

	n, _ = s.recvBuf.Read(b)

	err = s.sendWindowUpdate(s.readDeadline.wait())
	return n, err
}

func (s *Stream) Write(b []byte) (int, error) {
	var total int
	for total < len(b) {
		n, err := s.write(b[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (s *Stream) write(b []byte) (n int, err error) {
	var flags uint16
	var max uint32
	var hdr header

START:
	s.stateLock.Lock()
	state := s.writeState
	s.stateLock.Unlock()

	switch state {
	case halfOpen:
	case halfClosed:
		return 0, fmt.Errorf("STREAMERR: stream closed for writing")
	case halfReset:
		return 0, fmt.Errorf("STREAMERR: stream reset")
	default:
		panic("unknown state")
	}

	window := atomic.LoadUint32(&s.sendWindow)
	//send window is full
	if window == 0 {
		select {
		case <-s.sendNotifyCh:
			goto START
		case <-s.writeDeadline.wait():
			return 0, fmt.Errorf("timeout")
		}
	}

	flags = s.sendFlags()

	max = min(window, s.session.config.MaxMessageSize-headerSize, uint32(len(b)))
	hdr = encode(typeData, flags, s.id, max)
	if err = s.session.sendMsg(hdr, b[:max], s.writeDeadline.wait()); err != nil {
		return 0, err
	}

	atomic.AddUint32(&s.sendWindow, ^uint32(max-1))

	return int(max), err
}
