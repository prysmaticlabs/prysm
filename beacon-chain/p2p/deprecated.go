package p2p

import (
	"github.com/gogo/protobuf/proto"
	deprecatedp2p "github.com/prysmaticlabs/prysm/shared/deprecated-p2p"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// DeprecatedSubscriber exists for backwards compatibility.
// DEPRECATED: Do not use. This exists for backwards compatibility but may be removed.
type DeprecatedSubscriber interface {
	Subscribe(msg proto.Message, channel chan deprecatedp2p.Message) event.Subscription
}
