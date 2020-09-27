package logutil

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type fakeAddr int

var (
	localAddr  = fakeAddr(1)
	remoteAddr = fakeAddr(2)
)

func (a fakeAddr) Network() string {
	return "net"
}

func (a fakeAddr) String() string {
	return "str"
}

type fakeNetConn struct {
	io.Reader
	io.Writer
}

func (c fakeNetConn) Close() error                       { return nil }
func (c fakeNetConn) LocalAddr() net.Addr                { return localAddr }
func (c fakeNetConn) RemoteAddr() net.Addr               { return remoteAddr }
func (c fakeNetConn) SetDeadline(t time.Time) error      { return nil }
func (c fakeNetConn) SetReadDeadline(t time.Time) error  { return nil }
func (c fakeNetConn) SetWriteDeadline(t time.Time) error { return nil }

type testResponseWriter struct {
	brw *bufio.ReadWriter
	http.ResponseWriter
}

func (resp *testResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw := bufio.NewReadWriter(bufio.NewReader(strings.NewReader("")), bufio.NewWriter(&bytes.Buffer{}))
	return fakeNetConn{strings.NewReader(""), resp.brw}, rw, nil
}

func TestLogStreamServer_BackfillsMessages(t *testing.T) {
	ss := NewLogStreamServer()
	msgs := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("buzz"),
	}
	for _, msg := range msgs {
		_, err := ss.Write(msg)
		require.NoError(t, err)
	}

	br := bufio.NewReader(strings.NewReader(""))
	buf := new(bytes.Buffer)
	bw := bufio.NewWriter(buf)
	rw := httptest.NewRecorder()
	resp := &testResponseWriter{
		brw:            bufio.NewReadWriter(br, bw),
		ResponseWriter: rw,
	}
	req := &http.Request{
		Method: "GET",
		Header: http.Header{
			"Upgrade":               []string{"websocket"},
			"Connection":            []string{"upgrade"},
			"Sec-Websocket-Key":     []string{"dGhlIHNhbXBsZSBub25jZQ=="},
			"Sec-Websocket-Version": []string{"13"},
		},
	}

	go ss.Handler(resp, req)
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, resp.brw.Flush())
	dst, err := ioutil.ReadAll(buf)
	require.NoError(t, err)
	for _, msg := range msgs {
		if !strings.Contains(string(dst), string(msg)) {
			t.Errorf("Stream does contain message %s", msg)
		}
	}
}
