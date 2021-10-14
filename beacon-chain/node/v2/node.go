package v2

import (
	"context"

	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/runtime"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "node")

type Option func(*BeaconNode) error

type BeaconNode struct {
	services  []runtime.Service
	p2pOpts   []p2p.Option
	stateFeed *event.Feed
}

// StateFeed implements statefeed.Notifier.
func (b *BeaconNode) StateFeed() *event.Feed {
	return b.stateFeed
}

func New(ctx context.Context, opts ...Option) (*BeaconNode, error) {
	bn := &BeaconNode{}
	for _, opt := range opts {
		if err := opt(bn); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (b *BeaconNode) Start() error {
	return nil
}

func (b *BeaconNode) startP2P(ctx context.Context, beaconDB db.ReadOnlyDatabase, opts []p2p.Option) error {
	opts = append(
		opts,
		p2p.WithDatabase(beaconDB),
		p2p.WithStateNotifier(b),
	)
	svc, err := p2p.NewService(ctx, opts...)
	if err != nil {
		return nil
	}
	_ = svc
	return nil
}
