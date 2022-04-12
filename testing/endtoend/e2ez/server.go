package e2ez

import (
	"bytes"
	"context"
	"net/http"

	"github.com/gomarkdown/markdown"
	log "github.com/sirupsen/logrus"
)

// Server is an http server that serves all zpages.
// if zpages register using the HandleMarkdown method, their responses will be transformed from markdown
// to html before being streamed back to the client.
type Server struct {
	handler *http.ServeMux
	ec      chan error
}

// NewServer should be used to instantiate a Server, so that it can set up the internal http.Handler
// and http.Server values.
func NewServer() *Server {
	return &Server{
		handler: http.NewServeMux(),
		ec:      make(chan error),
	}
}

// ListenAndServe just starts the underlying http.Server using the provided addr.
// This method does not use a goroutine and will block, call it in a goroutine
// if you do not want the caller to block.
func (s *Server) ListenAndServe(ctx context.Context, addr string) {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.handler,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			s.ec <- err
		}
	}()
	for {
		select {
		case err := <-s.ec:
			log.Error(err)
		case <-ctx.Done():
			err := srv.Shutdown(ctx)
			if err != nil {
				log.Error(err)
			}
		}
	}
}

// HandleMarkdown mirrors http.HandleFunc. It wraps the given handler in a "middleware" enclosure that assumes
// a successful response body is a markdown document, translating the markdown to an html page.
func (s *Server) HandleMarkdown(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.handler.HandleFunc(pattern, handleMarkdown(handler, s.ec))
}

// HandleZPage allows any type that implements the minimal ZPage interface to
// handle requests for information about itself.
func (s *Server) HandleZPages(zps ...ZPage) {
	for i := 0; i < len(zps); i++ {
		z := zps[i]
		f := func(rw http.ResponseWriter, req *http.Request) {
			md, err := z.ZMarkdown()
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				s.ec <- err
				return
			}
			rw.Header().Add("Content-Type", "text/html")
			rw.WriteHeader(http.StatusOK)
			_, err = rw.Write(markdown.ToHTML([]byte(md), nil, nil))
			if err != nil {
				s.ec <- err
				return
			}
		}
		s.handler.HandleFunc(z.ZPath(), f)
	}
}

type markdownResponseWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (w *markdownResponseWriter) Write(i []byte) (int, error) {
	return w.buf.Write(i)
}

func handleMarkdown(wrapped http.HandlerFunc, ec chan error) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		w := &markdownResponseWriter{ResponseWriter: rw, buf: bytes.NewBuffer(nil)}
		wrapped(w, req)
		hb := markdown.ToHTML(w.buf.Bytes(), nil, nil)
		_, err := rw.Write(hb)
		if err != nil {
			ec <- err
			return
		}
	}
}

// ZPage allows a type to generate a markdown zpage without consideration for http server semantics.
// ZPath() is used to claim a path for itself in the zpage namespace, ZMarkdown returns the markdown
// value to translate to HTML.
type ZPage interface {
	ZPath() string
	ZMarkdown() (string, error)
}
