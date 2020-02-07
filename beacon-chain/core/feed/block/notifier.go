package block

import "github.com/prysmaticlabs/prysm/shared/event"

// Notifier interface defines the methods of the service that provides block updates to consumers.
type Notifier interface {
	BlockFeed() *event.Feed
}
