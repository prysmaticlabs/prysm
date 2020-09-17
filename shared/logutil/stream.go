package logutil

import (
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/rand"
	log "github.com/sirupsen/logrus"
)

// LogCacheSize is the number of log entries to keep in memory for new
// websocket connections.
const LogCacheSize = 20

// Compile time interface check.
var _ = io.Writer(&StreamServer{})

// StreamServer defines a a websocket server which can receive events from
// a feed and write them to open websocket connections.
type StreamServer struct {
	feed  *event.Feed
	cache *lru.Cache
}

// NewLogStreamServer initializes a new stream server capable of
// streaming log events via a websocket connection.
func NewLogStreamServer() *StreamServer {
	c, err := lru.New(LogCacheSize)
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

var streamUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Handler for new websocket connections to stream new events received
// via an event feed as they occur.
func (ss *StreamServer) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := streamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not write websocket message: %v", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Could not close websocket connection: %v", err)
		}
	}()

	ch := make(chan []byte, 1)
	defer close(ch)
	sub := ss.feed.Subscribe(ch)
	defer sub.Unsubscribe()

	// Backfill stream with recent messages.
	for _, k := range ss.cache.Keys() {
		d, ok := ss.cache.Get(k)
		if ok {
			if err := conn.WriteMessage(websocket.TextMessage, d.([]byte)); err != nil {
				log.Errorf("Could not write websocket message: %v", err)
				return
			}
		}
	}

	for {
		select {
		case evt := <-ch:
			if err := conn.WriteMessage(websocket.TextMessage, evt); err != nil {
				log.Errorf("Could not write websocket message: %v", err)
				return
			}
		case <-r.Context().Done():
			if err := conn.WriteMessage(websocket.CloseNormalClosure, []byte("context canceled")); err != nil {
				log.Error(err)
				return
			}
		case err := <-sub.Err():
			if err := conn.WriteMessage(websocket.CloseInternalServerErr, []byte(err.Error())); err != nil {
				log.Error(err)
				return
			}
		}
	}
}

// Write a binary message and send over the event feed.
func (ss *StreamServer) Write(p []byte) (n int, err error) {
	ss.feed.Send(p)
	ss.cache.Add(rand.NewGenerator().Uint64(), p)
	return len(p), nil
}
