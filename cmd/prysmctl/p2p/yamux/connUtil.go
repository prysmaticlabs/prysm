package yamux

import (
	"context"
)

func (c *conn) yamux() *Session {
	return (*Session)(c)
}

func (c *conn) Close() error {
	return c.yamux().Close()
}

func (c *conn) IsClosed() bool {
	return c.yamux().IsClosed()
}

func (c *conn) AcceptStream() (MuxedStream, error) {
	s, err := c.yamux().AcceptStream()
	return (*Stream)(s), err
}

func (c *conn) OpenStream(ctx context.Context) (MuxedStream, error) {
	s, err := c.yamux().OpenStream(ctx)
	if err != nil {
		return nil, err
	}

	return (*Stream)(s), nil
}
