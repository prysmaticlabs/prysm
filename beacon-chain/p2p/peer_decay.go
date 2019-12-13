package p2p

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
)

// starPeerDecay starts a loop that periodically decays the bad response count of peers, giving reformed peers a chance to rejoin
// the network.
// This runs every hour.
func startPeerDecay(ctx context.Context, peers *peers.Status) {
	go (func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				peers.Decay()
				time.Sleep(1 * time.Hour)
			}
		}
	})()
}
