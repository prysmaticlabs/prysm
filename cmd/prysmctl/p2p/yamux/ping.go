package yamux

import "time"

type ping struct {
	id           uint32
	pingResponse chan struct{}
	done         chan struct{}
	err          error
	duration     time.Duration
}

func newPing(id uint32) *ping {
	return &ping{
		id:           id,
		pingResponse: make(chan struct{}, 1),
		done:         make(chan struct{}),
	}
}

func (p *ping) finish(val time.Duration, err error) {
	p.err = err
	p.duration = val
	close(p.done)
}

func (p *ping) wait() (time.Duration, error) {
	<-p.done
	return p.duration, p.err
}
