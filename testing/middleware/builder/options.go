package builder

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type config struct {
	builderPort    int
	builderHost    string
	destinationUrl *url.URL
	beaconUrl      *url.URL
	logger         *logrus.Logger
	secret         string
}

type Option func(p *Builder) error

// WithHost sets the proxy server host.
func WithHost(host string) Option {
	return func(p *Builder) error {
		p.cfg.builderHost = host
		return nil
	}
}

// WithPort sets the proxy server port.
func WithPort(port int) Option {
	return func(p *Builder) error {
		p.cfg.builderPort = port
		return nil
	}
}

// WithDestinationAddress sets the forwarding address requests will be sent to.
func WithDestinationAddress(addr string) Option {
	return func(p *Builder) error {
		if addr == "" {
			return errors.New("must provide a destination address for builder")
		}
		u, err := url.Parse(addr)
		if err != nil {
			return errors.Wrapf(err, "could not parse URL for destination address: %s", addr)
		}
		p.cfg.destinationUrl = u
		return nil
	}
}

// WithJwtSecret adds in support for jwt authenticated
// connections for our proxy.
func WithJwtSecret(secret string) Option {
	return func(p *Builder) error {
		p.cfg.secret = secret
		return nil
	}
}

func WithBeaconAddress(addr string) Option {
	return func(p *Builder) error {
		if addr == "" {
			return errors.New("must provide a beacon address for builder")
		}
		u, err := url.Parse(addr)
		if err != nil {
			return errors.Wrapf(err, "could not parse URL for beacon address: %s", addr)
		}
		p.cfg.beaconUrl = u
		return nil
	}
}
