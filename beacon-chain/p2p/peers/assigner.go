package peers

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/sirupsen/logrus"
)

// handshakePollingInterval is a polling interval for checking the number of received handshakes.
var handshakePollingInterval = 5 * time.Second

func NewAssigner(ctx context.Context, s *Status, max int, finalized primitives.Epoch) *Assigner {
	return &Assigner{
		ctx:       ctx,
		ps:        s,
		max:       max,
		finalized: finalized,
	}
}

type Assigner struct {
	sync.Mutex
	ctx       context.Context
	ps        *Status
	max       int
	finalized primitives.Epoch
}

var ErrInsufficientSuitable = errors.New("no suitable peers")

func (a *Assigner) freshPeers() ([]peer.ID, error) {
	required := params.BeaconConfig().MaxPeersToSync
	if flags.Get().MinimumSyncPeers < required {
		required = flags.Get().MinimumSyncPeers
	}
	_, peers := a.ps.BestFinalized(params.BeaconConfig().MaxPeersToSync, a.finalized)
	if len(peers) < required {
		log.WithFields(logrus.Fields{
			"suitable": len(peers),
			"required": required}).Info("Unable to assign peer while suitable peers < required ")
		return nil, ErrInsufficientSuitable
	}
	return peers, nil
}

func (a *Assigner) Assign(busy map[peer.ID]bool, n int) ([]peer.ID, error) {
	best, err := a.freshPeers()
	ps := make([]peer.ID, 0, n)
	if err != nil {
		return nil, err
	}
	for _, p := range best {
		if !busy[p] {
			ps = append(ps, p)
		}
	}
	return ps, nil
}
