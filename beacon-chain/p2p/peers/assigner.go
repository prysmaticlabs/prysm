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
	latest    int
	finalized primitives.Epoch
	best      []peer.ID
}

var ErrNoSuitablePeers = errors.New("no suitable peers")

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
		return nil, ErrNoSuitablePeers
	}
	return peers, nil
}

/*
func filterBusy(busy map[peer.ID], all []peer.ID) []peer.ID {
	avail := make([]peer.ID, 0)
	for
}

*/

func (a *Assigner) Assign(busy map[peer.ID]bool) (peer.ID, error) {
	best, err := a.freshPeers()
	if err != nil {
		return "", err
	}
	for _, p := range best {
		if !busy[p] {
			return p, nil
		}
	}
	return "", errors.Wrapf(ErrNoSuitablePeers, "finalized=%d, max=%d", a.finalized, a.max)
}
