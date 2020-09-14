package logutil

import (
	"io"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/prysmaticlabs/prysm/shared/event"
)

// Compile time interface check.
var _ = io.Writer(&StreamServer{})

type StreamServer struct {
	feed *event.Feed
}

func NewLogStreamServer() *StreamServer {
	ss := &StreamServer{
		feed: new(event.Feed),
	}
	addLogWriter(ss)
	return ss
}

var streamUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (ss *StreamServer) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := streamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err) // TODO
	}

	ch := make(chan []byte)
	sub := ss.feed.Subscribe(ch)
	defer sub.Unsubscribe()

	for {
		select {
		case evt := <-ch:
			if err := conn.WriteMessage(websocket.TextMessage, evt); err != nil {
				panic(err) // TODO
			}
		case err := <-sub.Err():
			panic(err) // TODO
		}
	}
}

func (ss *StreamServer) Write(p []byte) (n int, err error)  {
	ss.feed.Send(p)
	return len(p), nil
}