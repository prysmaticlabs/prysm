package yamux

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	// TODO: dummy implementation
	pool "github.com/libp2p/go-buffer-pool"
)

type header [headerSize]byte

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

var (
	handlers = []func(*Session, header) error{
		typeData:         (*Session).handleStreamMessage,
		typeWindowUpdate: (*Session).handleStreamMessage,
		typePing:         (*Session).handlePing,
		typeGoAway:       (*Session).handleGoAway,
	}
)

// Periodically send "ping" messages to the peer, ensuring the connection is alive.
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

// Write messages to the network connection
func (s *Session) send() {
	if err := s.sendLoop(); err != nil {
		s.exitErr(err)
	}
}

func (s *Session) recv() {
	if err := s.recvLoop(); err != nil {
		s.exitErr(err)
	}
}

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

// Sends ping messages if they are received on the pingCh channel.
// Sends pong messages if they are received on the pongCh channel.
// Sends normal data messages if they are available on the sendCh channel.
func (s *Session) sendLoop() (err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			fmt.Fprintf(os.Stderr, "caught panic: %s\n%s\n", rerr, debug.Stack())
			err = fmt.Errorf("SESSIONERR: panic in yamux send loop: %s", rerr)
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

func (s *Session) recvLoop() (err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			fmt.Fprintf(os.Stderr, "caught panic: %s\n%s\n", rerr, debug.Stack())
			err = fmt.Errorf("SESSIONERR: panic in yamux receive loop: %s", rerr)
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
func (s *Session) exitErr(err error) {
	s.shutdownLock.Lock()
	if s.shutdownErr == nil {
		s.shutdownErr = err
	}
	s.shutdownLock.Unlock()
	s.Close()
}
