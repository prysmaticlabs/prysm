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
)

var _ pubsub.SubscriptionFilter = (*subscriptionFilter)(nil)

const pubsubSubscriptionRequestLimit = 100

type subscriptionFilter struct {
	ctx                context.Context
	notifier           statefeed.Notifier
	currentForkDigest  string
	previousForkDigest string
}

func (sf *subscriptionFilter) CanSubscribe(topic string) bool {
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
	if parts[2] != sf.currentForkDigest && parts[1] != sf.previousForkDigest {
		return false
	}

	// Check the incoming topic matches any topic mapping.
	for gt, _ := range GossipTopicMappings {
		if _, err := scanfcheck(topic, gt); err != nil {
			return true
		}
	}

	return false
}

func (sf *subscriptionFilter) FilterIncomingSubscriptions(id peer.ID, subs []*pubsubpb.RPC_SubOpts) ([]*pubsubpb.RPC_SubOpts, error) {
	if len(subs) > pubsubSubscriptionRequestLimit {
		return nil, pubsub.ErrTooManySubscriptions
	}

	return pubsub.FilterSubscriptions(subs, sf.CanSubscribe), nil
}

func newSubscriptionFilter(ctx context.Context, notifier statefeed.Notifier) pubsub.SubscriptionFilter {
	sf := &subscriptionFilter{
		ctx:      ctx,
		notifier: notifier,
	}

	go sf.monitorState()

	return sf
}

func (sf *subscriptionFilter) monitorState() {
	ch := make(chan *feed.Event, 1)
	sub := sf.notifier.StateFeed().Subscribe(ch)
	defer sub.Unsubscribe()
	for {
		select {
		case <-sub.Err():
			return
		case <-sf.ctx.Done():
			return
		case evt := <-ch:
			_ = evt
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
