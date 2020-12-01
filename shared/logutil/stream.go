package logutil

import (
	"io"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/rand"
	log "github.com/sirupsen/logrus"
)

const (
	// LogCacheSize is the number of log entries to keep in memory for new
	// websocket connections.
	LogCacheSize = 20
	// Size for the buffered channel used for receiving log messages. The default
	// value should be enough to handle most incoming amount of logs without
	// blocking the thread.
	logBufferSize = 100
)

var (
	// Compile time interface check.
	_              = io.Writer(&StreamServer{})
	streamUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Only allow requests from localhost.
			return true
		},
	}
)

// StreamServer defines a a websocket server which can receive events from
// a feed and write them to open websocket connections.
type StreamServer struct {
	feed    *event.Feed
	cache   *lru.Cache
	clients map[*websocket.Conn]bool
	lock    sync.RWMutex
}

// NewLogStreamServer initializes a new stream server capable of
// streaming log events via a websocket connection.
func NewLogStreamServer() *StreamServer {
	c, err := lru.New(LogCacheSize)
	if err != nil {
		panic(err) // This can only occur when the LogCacheSize is negative.
	}
	ss := &StreamServer{
		feed:    new(event.Feed),
		cache:   c,
		clients: make(map[*websocket.Conn]bool),
	}
	addLogWriter(ss)
	go ss.sendLogsToClients()
	return ss
}

// Handler for new websocket connections to stream new events received
// via an event feed as they occur.
func (ss *StreamServer) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := streamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not write websocket message: %v", err)
		return
	}

	// Backfill stream with recent messages.
	for _, k := range ss.cache.Keys() {
		d, ok := ss.cache.Get(k)
		if ok {
			if err := conn.WriteMessage(websocket.TextMessage, d.([]byte)); err != nil {
				log.Errorf("Could not write websocket message: %v", err)
				if err := conn.Close(); err != nil {
					log.Errorf("Could not close websocket connection: %v", err)
				}
				return
			}
		}
	}
	ss.lock.Lock()
	ss.clients[conn] = true
	ss.lock.Unlock()
}

// Write a binary message and send over the event feed.
func (ss *StreamServer) Write(p []byte) (n int, err error) {
	ss.feed.Send(p)
	ss.cache.Add(rand.NewGenerator().Uint64(), p)
	return len(p), nil
}

func (ss *StreamServer) sendLogsToClients() {
	ch := make(chan []byte, logBufferSize)
	defer close(ch)
	sub := ss.feed.Subscribe(ch)
	defer sub.Unsubscribe()

	for {
		select {
		case evt := <-ch:
			ss.lock.Lock()
			for conn := range ss.clients {
				if err := conn.WriteMessage(websocket.TextMessage, evt); err != nil {
					log.WithError(err).Error("Could not write websocket message")
					ss.removeClient(conn)
				}
			}
			ss.lock.Unlock()
		case err := <-sub.Err():
			ss.lock.Lock()
			for conn := range ss.clients {
				if err := conn.WriteMessage(websocket.CloseInternalServerErr, []byte(err.Error())); err != nil {
					log.WithError(err).Error("Could not write websocket message")
				}
				ss.removeClient(conn)
			}
			ss.lock.Unlock()
		}
	}
}

// The caller of this function needs to acquire a mutex.
func (ss *StreamServer) removeClient(conn *websocket.Conn) {
	delete(ss.clients, conn)
	if err := conn.Close(); err != nil {
		log.Errorf("Could not close websocket connection: %v", err)
	}
}
