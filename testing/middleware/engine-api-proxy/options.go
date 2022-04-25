package proxy

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type config struct {
	proxyPort      int
	proxyHost      string
	destinationUrl *url.URL
	logger         *logrus.Logger
}

type Option func(p *Proxy) error

// WithHost sets the proxy server host.
func WithHost(host string) Option {
	return func(p *Proxy) error {
		p.cfg.proxyHost = host
		return nil
	}
}

// WithPort sets the proxy server port.
func WithPort(port int) Option {
	return func(p *Proxy) error {
		p.cfg.proxyPort = port
		return nil
	}
}

// WithDestinationAddress sets the forwarding address requests will be proxied to.
func WithDestinationAddress(addr string) Option {
	return func(p *Proxy) error {
		if addr == "" {
			return errors.New("must provide a destination address for proxy")
		}
		u, err := url.Parse(addr)
		if err != nil {
			return errors.Wrapf(err, "could not parse URL for destination address: %s", addr)
		}
		p.cfg.destinationUrl = u
		return nil
	}
}

// WithLogger sets a custom logger for the proxy.
func WithLogger(l *logrus.Logger) Option {
	return func(p *Proxy) error {
		p.cfg.logger = l
		return nil
	}
}
