package logutil

import (
	"io"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/rand"
)

const (
	// The number of log entries to keep in memory.
	logCacheSize = 20
)

var (
	// Compile time interface checks.
	_ = io.Writer(&StreamServer{})
	_ = Streamer(&StreamServer{})
)

type Streamer interface {
	GetLastFewLogs() [][]byte
	LogsFeed() *event.Feed
}

// StreamServer defines a a websocket server which can receive events from
// a feed and write them to open websocket connections.
type StreamServer struct {
	feed  *event.Feed
	cache *lru.Cache
	lock  sync.RWMutex
}

// NewStreamServer initializes a new stream server capable of
// streaming log events.
func NewStreamServer() *StreamServer {
	c, err := lru.New(logCacheSize)
	if err != nil {
		panic(err) // This can only occur when the LogCacheSize is negative.
	}
	ss := &StreamServer{
		feed:  new(event.Feed),
		cache: c,
	}
	addLogWriter(ss)
	return ss
}

func (ss *StreamServer) GetLastFewLogs() [][]byte {
	messages := make([][]byte, 0)
	for _, k := range ss.cache.Keys() {
		d, ok := ss.cache.Get(k)
		if ok {
			messages = append(messages, d.([]byte))
		}
	}
	return messages
}

func (ss *StreamServer) LogsFeed() *event.Feed {
	return ss.feed
}

// Write a binary message and send over the event feed.
func (ss *StreamServer) Write(p []byte) (n int, err error) {
	ss.feed.Send(p)
	ss.cache.Add(rand.NewGenerator().Uint64(), p)
	return len(p), nil
}
