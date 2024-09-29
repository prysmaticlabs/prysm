package yamux

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
	// TODO: dummy implementation
)

type MemoryManager interface {
	ReserveMemory(size int, prio uint8) error
	ReleaseMemory(size int)
	Done()
}

type nullMemoryManagerImpl struct {
	mu sync.Mutex
}

func (n *nullMemoryManagerImpl) ReserveMemory(size int, prio uint8) error { return nil }
func (n *nullMemoryManagerImpl) ReleaseMemory(size int)                   {}
func (n *nullMemoryManagerImpl) Done()                                    {}

var nullMemoryManager = &nullMemoryManagerImpl{}

type Session struct {
	rtt          int64
	remoteGoAway int32
	localGoAway  int32
	nextStreamID uint32

	config           *Config
	logger           *log.Logger
	conn             net.Conn
	reader           io.Reader
	newMemoryManager func() (MemoryManager, error)

	pingLock   sync.Mutex
	pingID     uint32
	activePing *ping

	numIncomingStreams uint32

	//PERF: sync.Map?
	streams    map[uint32]*Stream
	inflight   map[uint32]struct{}
	streamLock sync.Mutex

	synCh    chan struct{}
	acceptCh chan *Stream
	sendCh   chan []byte
	pongCh   chan uint32
	pingCh   chan uint32

	//PERF: lazy initialization?
	recvDoneCh chan struct{}
	sendDoneCh chan struct{}

	client bool

	shutdown     bool
	shutdownErr  error
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex

	keepaliveLock   sync.Mutex
	keepaliveTimer  *time.Timer
	keepaliveActive bool
}

func newSession(config *Config, conn net.Conn, client bool, readBuf int, newMemoryManager func() (MemoryManager, error)) *Session {
	var reader io.Reader = conn
	if readBuf > 0 {
		reader = bufio.NewReaderSize(reader, readBuf)
	}
	if newMemoryManager == nil {
		newMemoryManager = func() (MemoryManager, error) { return nullMemoryManager, nil }
	}
	s := &Session{
		config:           config,
		client:           client,
		logger:           log.New(config.LogOutput, "", log.LstdFlags),
		conn:             conn,
		reader:           reader,
		streams:          make(map[uint32]*Stream),
		inflight:         make(map[uint32]struct{}),
		synCh:            make(chan struct{}, config.AcceptBacklog),
		acceptCh:         make(chan *Stream, config.AcceptBacklog),
		sendCh:           make(chan []byte, 64),
		pongCh:           make(chan uint32, config.PingBacklog),
		pingCh:           make(chan uint32),
		recvDoneCh:       make(chan struct{}),
		sendDoneCh:       make(chan struct{}),
		shutdownCh:       make(chan struct{}),
		newMemoryManager: newMemoryManager,
	}
	if client {
		s.nextStreamID = 1
	} else {
		s.nextStreamID = 2
	}
	if config.EnableKeepAlive {
		s.startKeepalive()
	}
	go s.recv()
	go s.send()
	go s.startMeasureRTT()
	return s
}

// close the session and all streams
func (s *Session) Close() error {
	s.shutdownLock.Lock()
	defer s.shutdownLock.Unlock()

	if s.shutdown {
		return nil
	}
	s.shutdown = true
	if s.shutdownErr == nil {
		s.shutdownErr = errors.New("SESSIONERR: session shutdown")
	}
	close(s.shutdownCh)
	s.conn.Close()
	s.stopKeepalive()
	<-s.recvDoneCh
	<-s.sendDoneCh

	s.streamLock.Lock()
	defer s.streamLock.Unlock()
	for id, stream := range s.streams {
		stream.forceClose()
		delete(s.streams, id)
		stream.memorySpan.Done()
	}
	return nil
}

func (s *Session) AcceptStream() (*Stream, error) {
	for {
		select {
		case stream := <-s.acceptCh:
			if err := stream.sendWindowUpdate(nil); err != nil {
				s.logger.Printf("[WARN] error sending window update before accepting: %s", err)
				continue
			}
			return stream, nil
		case <-s.shutdownCh:
			return nil, s.shutdownErr
		}
	}
}

// reads the stream message and creates new streams
func (s *Session) handleStreamMessage(hdr header) error {
	id := hdr.StreamID()
	flags := hdr.Flags()
	if flags&flagSYN == flagSYN {
		if err := s.incomingStream(id); err != nil {
			return err
		}
	}

	s.streamLock.Lock()
	stream := s.streams[id]
	s.streamLock.Unlock()

	if stream == nil {
		if hdr.MsgType() == typeData && hdr.Length() > 0 {
			if _, err := io.CopyN(io.Discard, s.reader, int64(hdr.Length())); err != nil {
				return nil
			}
		}
		return nil
	}

	if hdr.MsgType() == typeWindowUpdate {
		stream.incrSendWindow(hdr, flags)
		return nil
	}

	if err := stream.readData(hdr, flags, s.reader); err != nil {
		if sendErr := s.sendMsg(s.goAway(goAwayProtoErr), nil, nil); sendErr != nil {
			s.logger.Printf("[WARN] yamux: failed to send go away: %v", sendErr)
		}
		return err
	}
	return nil
}

// handlePing is invoked for a typePing frame
func (s *Session) handlePing(hdr header) error {
	flags := hdr.Flags()
	pingID := hdr.Length()

	if flags&flagSYN == flagSYN {
		select {
		case s.pongCh <- pingID:
		default:
			s.logger.Printf("[WARN] yamux: dropped ping reply")
		}
		return nil
	}

	// Handle a response
	s.pingLock.Lock()
	if s.activePing != nil && s.activePing.id == pingID {
		select {
		case s.activePing.pingResponse <- struct{}{}:
		default:
		}
	}
	s.pingLock.Unlock()
	return nil
}

// handleGoAway is invokde for a typeGoAway frame
func (s *Session) handleGoAway(hdr header) error {
	code := hdr.Length()
	switch code {
	case goAwayNormal:
		atomic.SwapInt32(&s.remoteGoAway, 1)
	case goAwayProtoErr:
		s.logger.Printf("[ERR] yamux: received protocol error go away")
		return fmt.Errorf("SESSIONERR: yamux protocol error")
	case goAwayInternalErr:
		s.logger.Printf("[ERR] yamux: received internal error go away")
		return fmt.Errorf("SESSIONERR: remote yamux internal error")
	default:
		s.logger.Printf("[ERR] yamux: received unexpected go away")
		return fmt.Errorf("SESSIONERR: unexpected go away received")
	}
	return nil
}

func (s *Session) goAway(reason uint32) header {
	atomic.SwapInt32(&s.localGoAway, 1)
	hdr := encode(typeGoAway, 0, 0, reason)
	return hdr
}

// initialize incoming stream
func (s *Session) incomingStream(id uint32) error {
	if s.client != (id%2 == 0) {
		s.logger.Printf("[ERR] yamux: both endpoints are clients")
		return fmt.Errorf("SESSIONERR: both yamux endpoints are clients")
	}

	// localGoAway state --> stream is shutting down and no new streams should be initiated
	if atomic.LoadInt32(&s.localGoAway) == 1 {
		// reset the stream
		hdr := encode(typeWindowUpdate, flagRST, id, 0)
		return s.sendMsg(hdr, nil, nil)
	}

	span, err := s.newMemoryManager()
	if err != nil {
		return fmt.Errorf("SESSIONERR: failed to create resource span: %w", err)
	}
	if err := span.ReserveMemory(initialStreamWindow, 255); err != nil {
		return err
	}
	stream := newStream(s, id, streamSYNReceived, initialStreamWindow, span)

	s.streamLock.Lock()
	defer s.streamLock.Unlock()

	if _, ok := s.streams[id]; ok {
		s.logger.Printf("[ERR] yamux: duplicate stream declared")
		if sendErr := s.sendMsg(s.goAway(goAwayProtoErr), nil, nil); sendErr != nil {
			s.logger.Printf("[WARN] yamux: failed to send go away: %v", sendErr)
		}
		span.Done()
		return errors.New("duplicate stream initiated")
	}

	if s.numIncomingStreams >= s.config.MaxIncomingStreams {
		s.logger.Printf("[WARN] yamux: MaxIncomingStreams exceeded, forcing stream reset")
		defer span.Done()
		hdr := encode(typeWindowUpdate, flagRST, id, 0)
		return s.sendMsg(hdr, nil, nil)
	}

	s.numIncomingStreams++
	s.streams[id] = stream

	select {
	case s.acceptCh <- stream:
		return nil
	default:
		defer span.Done()
		s.logger.Printf("[WARN] yamux: backlog exceeded, forcing stream reset")
		s.deleteStream(id)
		hdr := encode(typeWindowUpdate, flagRST, id, 0)
		return s.sendMsg(hdr, nil, nil)
	}
}

func (s *Session) deleteStream(id uint32) {
	str, ok := s.streams[id]
	if !ok {
		return
	}
	if s.client == (id%2 == 0) {
		if s.numIncomingStreams == 0 {
			s.logger.Printf("[ERR] yamux: numIncomingStreams underflow")
			// prevent the creation of any new streams
			s.numIncomingStreams = math.MaxUint32
		} else {
			s.numIncomingStreams--
		}
	}
	delete(s.streams, id)
	str.memorySpan.Done()
}

// calculates rtt for the ping
func (s *Session) Ping() (dur time.Duration, err error) {
	s.pingLock.Lock()
	if s.activePing != nil {
		activePing := s.activePing
		s.pingLock.Unlock()
		return activePing.wait()
	}

	activePing := newPing(s.pingID)
	s.pingID++
	s.activePing = activePing
	s.pingLock.Unlock()

	defer func() {
		activePing.finish(dur, err)
		s.pingLock.Lock()
		s.activePing = nil
		s.pingLock.Unlock()
	}()

	timer := time.NewTimer(s.config.ConnectionWriteTimeout)
	defer timer.Stop()
	select {
	case s.pingCh <- activePing.id:
	case <-timer.C:
		return 0, errors.New("PINGERR: i/o deadline reached")
	case <-s.shutdownCh:
		return 0, s.shutdownErr
	}

	start := time.Now()

	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(s.config.ConnectionWriteTimeout)
	select {
	case <-activePing.pingResponse:
	case <-timer.C:
		return 0, errors.New("PINGERR: i/o deadline reached")
	case <-s.shutdownCh:
		return 0, s.shutdownErr
	}

	// Compute the RTT
	return time.Since(start), nil
}

func (s *Session) OpenStream(ctx context.Context) (*Stream, error) {
	if s.IsClosed() {
		return nil, s.shutdownErr
	}
	if atomic.LoadInt32(&s.remoteGoAway) == 1 {
		return nil, fmt.Errorf("SESSIONERR: remote end is not accepting connections")
	}

	select {
	case s.synCh <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.shutdownCh:
		return nil, s.shutdownErr
	}

	span, err := s.newMemoryManager()
	if err != nil {
		return nil, fmt.Errorf("SESSIONERR: failed to create resource scope span: %w", err)
	}
	if err := span.ReserveMemory(initialStreamWindow, 255); err != nil {
		return nil, err
	}

GET_ID:
	// Get an ID, and check for stream exhaustion
	id := atomic.LoadUint32(&s.nextStreamID)
	if id >= math.MaxUint32-1 {
		span.Done()
		return nil, fmt.Errorf("SESSIONERR: streams exhauseted")
	}
	if !atomic.CompareAndSwapUint32(&s.nextStreamID, id, id+2) {
		goto GET_ID
	}

	// Register the stream
	stream := newStream(s, id, streamInit, initialStreamWindow, span)
	s.streamLock.Lock()
	s.streams[id] = stream
	s.inflight[id] = struct{}{}
	s.streamLock.Unlock()

	// Send the window update to create
	if err := stream.sendWindowUpdate(ctx.Done()); err != nil {
		defer span.Done()
		select {
		case <-s.synCh:
		default:
			s.logger.Printf("[ERR] yamux: aborted stream open without inflight syn semaphore")
		}
		return nil, err
	}
	return stream, nil
}

func (s *Session) IsClosed() bool {
	select {
	case <-s.shutdownCh:
		return true
	default:
		return false
	}
}
