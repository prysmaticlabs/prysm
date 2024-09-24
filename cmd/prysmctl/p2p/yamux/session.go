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
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	// TODO: dummy implementation
	pool "github.com/libp2p/go-buffer-pool"
)

type MemoryManager interface {
	ReserveMemory(size int, prio uint8) error
	ReleaseMemory(size int)
	Done()
}

type nullMemoryManagerImpl struct {
	// mu sync.Mutex
}

func (n nullMemoryManagerImpl) ReserveMemory(size int, prio uint8) error { return nil }
func (n nullMemoryManagerImpl) ReleaseMemory(size int)                   {}
func (n nullMemoryManagerImpl) Done()                                    {}

var nullMemoryManager = &nullMemoryManagerImpl{}

type Session struct {
	rtt int64 // nanoseconds

	// remote side does not want futher connections.
	remoteGoAway int32

	// stop accepting futher connections.
	localGoAway int32

	// next stream to be sent
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
	streams            map[uint32]*Stream
	inflight           map[uint32]struct{}
	streamLock         sync.Mutex

	// synCh acts like a semaphore.
	synCh chan struct{}

	// pass ready streams to the client
	acceptCh chan *Stream

	// send messages
	sendCh chan []byte

	// send pings and pongs
	pongCh, pingCh chan uint32

	// recvDoneCh is closed when recv() exits to avoid a race between stream registration and stream shutdown
	recvDoneCh chan struct{}

	// sendDoneCh is closed when send() exits to avoid a race between returning from a Stream.Write and exiting from the send loop
	sendDoneCh chan struct{}

	// client is true if we're the client and our stream IDs should be odd.
	client bool

	// shutdown is used to safely close a session
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

func (s *Session) startKeepalive() {
	s.keepaliveLock.Lock()
	defer s.keepaliveLock.Unlock()

	s.keepaliveTimer = time.AfterFunc(s.config.KeepAliveInterval, func() {
		s.keepaliveLock.Lock()
		if s.keepaliveTimer == nil || s.keepaliveActive {
			s.keepaliveLock.Unlock()
			return
		}

		s.keepaliveActive = true
		s.keepaliveLock.Unlock()

		_, err := s.Ping()

		s.keepaliveLock.Lock()
		s.keepaliveActive = false
		if s.keepaliveTimer != nil {
			s.keepaliveTimer.Reset(s.config.KeepAliveInterval)
		}
		s.keepaliveLock.Unlock()

		if err != nil {
			s.logger.Printf("SESSION ERROR: keepalive failed: %v", err)
			s.shutdownLock.Lock()
			if s.shutdownErr == nil {
				s.shutdownErr = err
			}
			s.shutdownLock.Unlock()
			s.Close()
		}
	})
}

func (s *Session) stopKeepalive() {
	s.keepaliveLock.Lock()
	defer s.keepaliveLock.Unlock()
	if s.keepaliveTimer != nil {
		s.keepaliveTimer.Stop()
		s.keepaliveTimer = nil
	}
}

func (s *Session) extendKeepalive() {
	s.keepaliveLock.Lock()
	if s.keepaliveTimer != nil && !s.keepaliveActive {
		s.keepaliveTimer.Reset(s.config.KeepAliveInterval)
	}
	s.keepaliveLock.Unlock()
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
		s.shutdownErr = errors.New("SHUTDOWN ERROR: session shutdown")
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

func (s *Session) measureRTT() {
	rtt, err := s.Ping()
	if err != nil {
		return
	}
	if !atomic.CompareAndSwapInt64(&s.rtt, 0, rtt.Nanoseconds()) {
		prev := atomic.LoadInt64(&s.rtt)
		smoothedRTT := prev/2 + rtt.Nanoseconds()/2
		atomic.StoreInt64(&s.rtt, smoothedRTT)
	}
}

func (s *Session) startMeasureRTT() {
	s.measureRTT()
	t := time.NewTicker(s.config.MeasureRTTInterval)
	defer t.Stop()
	for {
		select {
		case <-s.shutdownCh:
			return
		case <-t.C:
			s.measureRTT()
		}
	}
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

func (s *Session) exitErr(err error) {
	s.shutdownLock.Lock()
	if s.shutdownErr == nil {
		s.shutdownErr = err
	}
	s.shutdownLock.Unlock()
	s.Close()
}

var (
	handlers = []func(*Session, header) error{
		typeData:         (*Session).handleStreamMessage,
		typeWindowUpdate: (*Session).handleStreamMessage,
		typePing:         (*Session).handlePing,
		typeGoAway:       (*Session).handleGoAway,
	}
)

func (s *Session) sendMsg(hdr header, body []byte, deadline <-chan struct{}) error {
	select {
	case <-s.shutdownCh:
		return s.shutdownErr
	default:
	}

	select {
	case <-deadline:
		return errors.New("SESSIONERR: i/o deadline reached")
	default:
	}

	// duplicate as we're sending this async.
	buf := pool.Get(headerSize + len(body))
	copy(buf[:headerSize], hdr[:])
	copy(buf[headerSize:], body)

	select {
	case <-s.shutdownCh:
		pool.Put(buf)
		return s.shutdownErr
	case s.sendCh <- buf:
		return nil
	case <-deadline:
		pool.Put(buf)
		return errors.New("SESSIONERR: i/o deadline reached")
	}
}

func (s *Session) send() {
	if err := s.sendLoop(); err != nil {
		s.exitErr(err)
	}
}

func (s *Session) sendLoop() (err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			fmt.Fprintf(os.Stderr, "caught panic: %s\n%s\n", rerr, debug.Stack())
			err = fmt.Errorf("panic in yamux send loop: %s", rerr)
		}
	}()

	defer close(s.sendDoneCh)
	var lastWriteDeadline time.Time
	extendWriteDeadline := func() error {
		now := time.Now()
		// If over half of the deadline has elapsed, extend it.
		if now.Add(s.config.ConnectionWriteTimeout / 2).After(lastWriteDeadline) {
			lastWriteDeadline = now.Add(s.config.ConnectionWriteTimeout)
			return s.conn.SetWriteDeadline(lastWriteDeadline)
		}
		return nil
	}

	writer := s.conn

	// TODO: https://github.com/libp2p/go-libp2p/issues/644
	// Write coalescing is disabled for now.

	for {
		select {
		case <-s.shutdownCh:
			return nil
		default:
		}

		var buf []byte
		select {
		case pingID := <-s.pingCh:
			buf = pool.Get(headerSize)
			hdr := encode(typePing, flagSYN, 0, pingID)
			copy(buf, hdr[:])
		case pingID := <-s.pongCh:
			buf = pool.Get(headerSize)
			hdr := encode(typePing, flagACK, 0, pingID)
			copy(buf, hdr[:])
		default:
			// Then send normal data.
			select {
			case buf = <-s.sendCh:
			case pingID := <-s.pingCh:
				buf = pool.Get(headerSize)
				hdr := encode(typePing, flagSYN, 0, pingID)
				copy(buf, hdr[:])
			case pingID := <-s.pongCh:
				buf = pool.Get(headerSize)
				hdr := encode(typePing, flagACK, 0, pingID)
				copy(buf, hdr[:])
			case <-s.shutdownCh:
				return nil
			}
		}

		if err := extendWriteDeadline(); err != nil {
			pool.Put(buf)
			return err
		}

		_, err := writer.Write(buf)
		pool.Put(buf)

		if err != nil {
			if os.IsTimeout(err) {
				err = errors.New("CONNERR: connection write timeout")
			}
			return err
		}
	}
}

func (s *Session) recv() {
	if err := s.recvLoop(); err != nil {
		s.exitErr(err)
	}
}

func (s *Session) recvLoop() (err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			fmt.Fprintf(os.Stderr, "caught panic: %s\n%s\n", rerr, debug.Stack())
			err = fmt.Errorf("panic in yamux receive loop: %s", rerr)
		}
	}()
	defer close(s.recvDoneCh)
	var hdr header
	for {
		if _, err := io.ReadFull(s.reader, hdr[:]); err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "closed") && !strings.Contains(err.Error(), "reset by peer") {
				s.logger.Printf("[ERR] yamux: Failed to read header: %v", err)
			}
			return err
		}
		s.extendKeepalive()

		// Verify the version
		if hdr.Version() != protoVersion {
			s.logger.Printf("[ERR] yamux: Invalid protocol version: %d", hdr.Version())
			return errors.New("SESSIONERR: invalid protocol version")
		}

		mt := hdr.MsgType()
		if mt < typeData || mt > typeGoAway {
			return errors.New("SESSIONERR: invalid message type")
		}

		if err := handlers[mt](s, hdr); err != nil {
			return err
		}
	}
}

func (s *Session) incomingStream(id uint32) error {
	if s.client != (id%2 == 0) {
		s.logger.Printf("[ERR] yamux: both endpoints are clients")
		return fmt.Errorf("both yamux endpoints are clients")
	}

	if atomic.LoadInt32(&s.localGoAway) == 1 {
		hdr := encode(typeWindowUpdate, flagRST, id, 0)
		return s.sendMsg(hdr, nil, nil)
	}

	span, err := s.newMemoryManager()
	if err != nil {
		return fmt.Errorf("failed to create resource span: %w", err)
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

	// Check if this is a window update
	if hdr.MsgType() == typeWindowUpdate {
		stream.incrSendWindow(hdr, flags)
		return nil
	}

	// Read the new data
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

	// Check if this is a query, respond back in a separate context so we
	// don't interfere with the receiving thread blocking for the write.
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
	// If we have an active ping, and this is a response to that active
	// ping, complete the ping.
	if s.activePing != nil && s.activePing.id == pingID {
		// Don't assume that the peer won't send multiple responses for
		// the same ping.
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
		return fmt.Errorf("yamux protocol error")
	case goAwayInternalErr:
		s.logger.Printf("[ERR] yamux: received internal error go away")
		return fmt.Errorf("remote yamux internal error")
	default:
		s.logger.Printf("[ERR] yamux: received unexpected go away")
		return fmt.Errorf("unexpected go away received")
	}
	return nil
}

func (s *Session) goAway(reason uint32) header {
	atomic.SwapInt32(&s.localGoAway, 1)
	hdr := encode(typeGoAway, 0, 0, reason)
	return hdr
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

func (s *Session) Ping() (dur time.Duration, err error) {
	s.pingLock.Lock()
	if activePing := s.activePing; activePing != nil {
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

func (s *Session) closeStream(id uint32) {
	s.streamLock.Lock()
	defer s.streamLock.Unlock()
	if _, ok := s.inflight[id]; ok {
		select {
		case <-s.synCh:
		default:
			s.logger.Printf("[ERR] yamux: SYN tracking out of sync")
		}
		delete(s.inflight, id)
	}
	s.deleteStream(id)
}

func (s *Session) establishStream(id uint32) {
	s.streamLock.Lock()
	if _, ok := s.inflight[id]; ok {
		delete(s.inflight, id)
	} else {
		s.logger.Printf("[ERR] yamux: established stream without inflight SYN (no tracking entry)")
	}
	select {
	case <-s.synCh:
	default:
		s.logger.Printf("[ERR] yamux: established stream without inflight SYN (didn't have semaphore)")
	}
	s.streamLock.Unlock()
}

func (s *Session) getRTT() time.Duration {
	return time.Duration(atomic.LoadInt64(&s.rtt))
}

func (s *Session) OpenStream(ctx context.Context) (*Stream, error) {
	if s.IsClosed() {
		return nil, s.shutdownErr
	}
	if atomic.LoadInt32(&s.remoteGoAway) == 1 {
		return nil, fmt.Errorf("remote end is not accepting connections")
	}

	// Block if we have too many inflight SYNs
	select {
	case s.synCh <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.shutdownCh:
		return nil, s.shutdownErr
	}

	span, err := s.newMemoryManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create resource scope span: %w", err)
	}
	if err := span.ReserveMemory(initialStreamWindow, 255); err != nil {
		return nil, err
	}

GET_ID:
	// Get an ID, and check for stream exhaustion
	id := atomic.LoadUint32(&s.nextStreamID)
	if id >= math.MaxUint32-1 {
		span.Done()
		return nil, fmt.Errorf("streams exhauseted")
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
