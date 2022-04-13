package proxy

import (
	"github.com/sirupsen/logrus"
)

type config struct {
	proxyPort       int
	destinationAddr string
	logger          *logrus.Logger
}

type Option func(p *Proxy) error

func WithPort(port int) Option {
	return func(p *Proxy) error {
		p.cfg.proxyPort = port
		return nil
	}
}

func WithDestinationAddress(addr string) Option {
	return func(p *Proxy) error {
		p.cfg.destinationAddr = addr
		return nil
	}
}

func WithLogger(l *logrus.Logger) Option {
	return func(p *Proxy) error {
		p.cfg.logger = l
		return nil
	}
}
