package proxy

type config struct {
	proxyPort       int
	destinationAddr string
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
