package p2p

import (
	"context"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var _ pubsub.SubscriptionFilter = (*subscriptionFilter)(nil)

const pubsubSubscriptionRequestLimit = 100

// subscriptionFilter handles filtering pubsub topic subscription metadata from peers.
// Notice: This filter does not support phase 1 fork schedule yet.
type subscriptionFilter struct {
	ctx      context.Context
	notifier statefeed.Notifier

	// Once the genesis state is initialized, this filter is now aware of the genesis fork digest.
	// This filter does not check the fork schedule for planned forks and their respective digests.
	initialized        bool
	currentForkDigest  string
	previousForkDigest string
}

// CanSubscribe returns true if the topic is of interest and we could subscribe to it.
func (sf *subscriptionFilter) CanSubscribe(topic string) bool {
	if !sf.initialized {
		return false
	}
	parts := strings.Split(topic, "/")
	if len(parts) != 4 {
		return false
	}
	// The topic must start with a slash, which means the first part will be empty.
	if parts[0] != "" {
		return false
	}
	if parts[1] != "eth2" {
		return false
	}
	if parts[2] != sf.currentForkDigest && parts[2] != sf.previousForkDigest {
		return false
	}

	// Check the incoming topic matches any topic mapping.
	for gt := range GossipTopicMappings {
		if _, err := scanfcheck(topic, gt); err == nil {
			return true
		}
	}

	return false
}

// FilterIncomingSubscriptions is invoked for all RPCs containing subscription notifications.
// This method returns only the topics of interest and may return an error if the subscription
// request contains too many topics.
func (sf *subscriptionFilter) FilterIncomingSubscriptions(_ peer.ID, subs []*pubsubpb.RPC_SubOpts) ([]*pubsubpb.RPC_SubOpts, error) {
	if len(subs) > pubsubSubscriptionRequestLimit {
		return nil, pubsub.ErrTooManySubscriptions
	}

	return pubsub.FilterSubscriptions(subs, sf.CanSubscribe), nil
}

func newSubscriptionFilter(ctx context.Context, notifier statefeed.Notifier) pubsub.SubscriptionFilter {
	sf := &subscriptionFilter{
		ctx:                ctx,
		notifier:           notifier,
		currentForkDigest:  fmt.Sprintf("%x", params.BeaconConfig().GenesisForkVersion),
		previousForkDigest: fmt.Sprintf("%x", params.BeaconConfig().GenesisForkVersion),
	}

	if notifier == nil {
		// This should never happen, except maybe in a poorly set up test.
		panic("notifier must not be nil")
	}

	go sf.monitorStateInitialized()

	return sf
}

// Monitor the state feed notifier for the state initialization event.
func (sf *subscriptionFilter) monitorStateInitialized() {
	ch := make(chan *feed.Event, 1)
	sub := sf.notifier.StateFeed().Subscribe(ch)
	defer sub.Unsubscribe()
	for {
		select {
		case err := <-sub.Err():
			log.WithError(err).Error("Failed to initialize subscription filter data")
			return
		case <-sf.ctx.Done():
			return
		case evt := <-ch:
			if evt.Type != statefeed.Initialized {
				continue
			}
			if d, ok := evt.Data.(*statefeed.InitializedData); ok {
				fd, err := p2putils.CreateForkDigest(d.StartTime, d.GenesisValidatorsRoot)
				if err != nil {
					log.WithError(err).Error("Failed to create fork digest")
					continue
				}
				sf.initialized = true
				sfd := fmt.Sprintf("%x", fd)
				sf.currentForkDigest = sfd
				sf.previousForkDigest = sfd
				return
			}
		}
	}
}

// scanfcheck uses fmt.Sscanf to check that a given string matches expected format. This method
// returns the number of formatting substitutions matched and error if the string does not match
// the expected format. Note: this method only accepts integer compatible formatting substitutions
// such as %d or %x.
func scanfcheck(input, format string) (int, error) {
	var t int
	var cnt = strings.Count(format, "%")
	var args = []interface{}{}
	for i := 0; i < cnt; i++ {
		args = append(args, &t)
	}
	return fmt.Sscanf(input, format, args...)
}
