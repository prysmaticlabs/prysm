package yamux

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"time"
)

type Config struct {
	AcceptBacklog           int
	PingBacklog             int
	EnableKeepAlive         bool
	KeepAliveInterval       time.Duration
	MeasureRTTInterval      time.Duration
	ConnectionWriteTimeout  time.Duration
	MaxIncomingStreams      uint32
	InitialStreamWindowSize uint32
	MaxStreamWindowSize     uint32
	LogOutput               io.Writer
	ReadBufSize             int
	WriteCoalesceDelay      time.Duration
	MaxMessageSize          uint32
}

func DefaultConfig() *Config {
	return &Config{
		AcceptBacklog:           256,
		PingBacklog:             32,
		EnableKeepAlive:         true,
		KeepAliveInterval:       30 * time.Second,
		MeasureRTTInterval:      30 * time.Second,
		ConnectionWriteTimeout:  10 * time.Second,
		MaxIncomingStreams:      math.MaxUint32,
		InitialStreamWindowSize: 256 * 1024,
		MaxStreamWindowSize:     16 * 1024 * 1024,
		LogOutput:               io.Discard,
		ReadBufSize:             0,
		MaxMessageSize:          64 * 1024,
		WriteCoalesceDelay:      100 * time.Microsecond,
	}
}

func VerifyConfig(config *Config) error {
	if config.AcceptBacklog <= 0 {
		return fmt.Errorf("CONFIG ERROR: backlog must be positive")
	}
	if config.EnableKeepAlive && config.KeepAliveInterval == 0 {
		return fmt.Errorf("CONFIG ERROR: keep-alive interval must be positive")
	}
	if config.MeasureRTTInterval == 0 {
		return fmt.Errorf("CONFIG ERROR: measure-rtt interval must be positive")
	}

	if config.InitialStreamWindowSize < (256 * 1024) {
		return errors.New("CONFIG ERROR: InitialStreamWindowSize must be larger or equal to 256 kB")
	}
	if config.MaxStreamWindowSize < config.InitialStreamWindowSize {
		return errors.New("CONFIG ERROR: MaxStreamWindowSize must be larger than the InitialStreamWindowSize")
	}
	if config.MaxMessageSize < 1024 {
		return fmt.Errorf("CONFIG ERROR: MaxMessageSize must be greater than a kB")
	}
	if config.WriteCoalesceDelay < 0 {
		return fmt.Errorf("CONFIG ERROR: WriteCoalesceDelay must be >= 0")
	}
	if config.PingBacklog < 1 {
		return fmt.Errorf("CONFIG ERROR: PingBacklog must be positive")
	}
	return nil
}

func ConnHelper(conn net.Conn, config *Config, mm func() (MemoryManager, error), isClient bool) (*Session, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if err := VerifyConfig(config); err != nil {
		return nil, err
	}
	return newSession(config, conn, isClient, config.ReadBufSize, mm), nil
}

func Server(conn net.Conn, config *Config, mm func() (MemoryManager, error)) (*Session, error) {
	return ConnHelper(conn, config, mm, false)
}

func Client(conn net.Conn, config *Config, mm func() (MemoryManager, error)) (*Session, error) {
	return ConnHelper(conn, config, mm, true)
}
