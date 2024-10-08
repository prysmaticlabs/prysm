package events

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

type StreamingResponseWriterRecorder struct {
	http.ResponseWriter
	r             io.Reader
	w             io.Writer
	statusWritten *int
	status        chan int
	bodyRecording []byte
	flushed       bool
}

func (w *StreamingResponseWriterRecorder) StatusChan() chan int {
	return w.status
}

func NewStreamingResponseWriterRecorder() *StreamingResponseWriterRecorder {
	r, w := io.Pipe()
	return &StreamingResponseWriterRecorder{
		ResponseWriter: httptest.NewRecorder(),
		r:              r,
		w:              w,
		status:         make(chan int, 1),
	}
}

// Write implements http.ResponseWriter.
func (w *StreamingResponseWriterRecorder) Write(data []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	n, err := w.w.Write(data)
	if err != nil {
		return n, err
	}
	return w.ResponseWriter.Write(data)
}

// WriteHeader implements http.ResponseWriter.
func (w *StreamingResponseWriterRecorder) WriteHeader(statusCode int) {
	if w.statusWritten != nil {
		return
	}
	w.statusWritten = &statusCode
	w.status <- statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *StreamingResponseWriterRecorder) Body() io.Reader {
	return w.r
}

func (w *StreamingResponseWriterRecorder) RequireStatus(t *testing.T, status int) {
	if w.statusWritten == nil {
		t.Fatal("WriteHeader was not called")
	}
	require.Equal(t, status, *w.statusWritten)
}

func (w *StreamingResponseWriterRecorder) Flush() {
	fw, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		fw.Flush()
	}
	w.flushed = true
}

var _ http.ResponseWriter = &StreamingResponseWriterRecorder{}
